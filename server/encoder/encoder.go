package encoder

import (
	"errors"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/encoding"
	"github.com/JameelGharra/ascii-craft/server/encoding/huffman"
	"github.com/JameelGharra/ascii-craft/server/protocol"
)

type EncodingFrame struct {
	RLE  encoding.AsciiRLE
	Freq ascii.FreqTable
	Huff *huffman.Huffman

	Temp    []byte // to avoid overwriting myself while RLEing
	TempLen int

	Out        []byte
	Len        int // length of the bitstream data itself no meta overhead
	FinalSize  int // this one should include meta overhaed and not only length of bitstream
	Stride     int
	IsKeyFrame bool // whether it is an I-frame or P-frame
	Encoding   EncodingType
}

func NewEncodingFrame(size, stride int) *EncodingFrame {
	return &EncodingFrame{
		RLE:       *encoding.NewAsciiRLE(),
		Huff:      nil,
		Out:       make([]byte, size),
		Len:       0,
		FinalSize: 0,
		Stride:    stride,
		Temp:      make([]byte, size),
		TempLen:   0,
		Freq:      *ascii.NewFrequency(),
	}
}

type EncodingCall func(frame *EncodingFrame, curr, prev []byte) error

type Encoder struct {
	encodings     []EncodingCall
	frames        []*EncodingFrame
	size          int
	prevFrame     []byte
	lastBestFrame *EncodingFrame
	sequence      uint32
	stride        int
}

func NewEncoder(size, stride int) *Encoder {
	return &Encoder{
		encodings:     make([]EncodingCall, 0),
		frames:        make([]*EncodingFrame, 0),
		size:          size,
		prevFrame:     nil,
		lastBestFrame: nil,
		stride:        stride,
		sequence:      0,
	}
}

func (e *Encoder) AddEncoding(encoding EncodingCall) {
	e.encodings = append(e.encodings, encoding)
	e.frames = append(e.frames, NewEncodingFrame(e.size, e.stride))
}

// if this is the very first frame no matter what value for forceKeyFrame, it would be
// treated as a key frame
// another note, this assumes that the caller does not cache the buffer passed otherwise
func (e *Encoder) PushFrame(rawData []byte, forceKeyFrame bool) *EncodingFrame {
	minSizeTarget := len(rawData) + 1
	var outFrame *EncodingFrame
	for i, encoding := range e.encodings {
		frame := e.frames[i]
		frame.IsKeyFrame = forceKeyFrame
		if e.prevFrame == nil {
			frame.IsKeyFrame = true
		}
		err := encoding(frame, rawData, e.prevFrame)
		if err != nil {
			continue
		}
		if frame.FinalSize < minSizeTarget {
			minSizeTarget = frame.FinalSize
			outFrame = frame
		}
	}
	e.lastBestFrame = outFrame
	if e.prevFrame == nil || len(e.prevFrame) != len(rawData) {
		e.prevFrame = make([]byte, len(rawData))
	}
	copy(e.prevFrame, rawData)
	e.sequence++
	return outFrame
}

var ErrNoBestEncoding = errors.New("no encoding has happened yet, cannot write to packet")
var ErrInvalidEncodingType = errors.New("invalid encoding type in frame")
var ErrHuffmanNotPerformed = errors.New("huffman encoding not performed yet in frame")

const (
	FlagIsCompressed = 1 << 0 // bit 0
	FlagMethod       = 1 << 1 // bit 1 (0 = XOR_RLE, 1 = XOR_HUFFMAN)
	FlagIsDelta      = 1 << 2 // (0 = I-Frame, 1 = P-frame)
	// bits 3-4 for meta mode
)

// [FLAGS] [SEQ] [DATA LEN (varint)] [META LEN + TABLE (variable)] [DATA]
func (e *Encoder) WriteTo(pb *protocol.PacketBuilder) error {
	frame := e.lastBestFrame
	if frame == nil {
		return ErrNoBestEncoding
	}
	// just reserving space for later use and backpatch
	headerStartIndex := pb.CurrentSize() // incase there is some data already written
	if err := pb.WriteByte(0); err != nil {
		return err
	}
	// varint chunks
	if err := pb.WriteVarint(e.sequence); err != nil {
		return err
	}
	if err := pb.WriteVarint(uint32(frame.Len)); err != nil {
		return err
	}
	// flag setting
	var flags, metaMode byte
	var err error
	if frame.Encoding != NONE {
		flags |= FlagIsCompressed
	}
	if !frame.IsKeyFrame && frame.Encoding != NONE { // force refresh or the very first frame
		flags |= FlagIsDelta
	}
	switch frame.Encoding {
	case XOR_HUFFMAN, HUFFMAN:
		flags |= FlagMethod
		if frame.Huff == nil {
			return ErrHuffmanNotPerformed
		}
		metaMode, err = pb.WriteHuffmanMeta(frame.Huff)
		if err != nil {
			return err
		}
		flags |= metaMode << 3
	case XOR_RLE:

	case NONE:
	default:
		return ErrInvalidEncodingType
	}
	if err := pb.WriteBytes(frame.Out[:frame.Len]); err != nil {
		return err
	}
	// getting back to fully write flags
	if err := pb.SetByte(headerStartIndex, flags); err != nil {
		return err
	}
	return nil
}
