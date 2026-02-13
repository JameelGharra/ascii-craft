package encoder

import (
	"errors"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/ascii/quad_tree"
	"github.com/JameelGharra/ascii-craft/server/encoding"
	"github.com/JameelGharra/ascii-craft/server/encoding/huffman"
	"github.com/JameelGharra/ascii-craft/server/protocol"
)

type EncodingFrame struct {
	Prev []byte
	Curr []byte

	CurrQT quad_tree.QuadTree
	PrevQT quad_tree.QuadTree

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
	Encoding   byte
}

func NewEncodingFrame(size int, params quad_tree.QuadTreeParam) *EncodingFrame {
	out := make([]byte, size)
	prevQt := quad_tree.Partition(out, params)
	currQt := quad_tree.Partition(out, params)
	return &EncodingFrame{
		Prev:   nil,
		PrevQT: prevQt,

		Curr:   nil,
		CurrQT: currQt,

		RLE:       *encoding.NewAsciiRLE(),
		Huff:      nil,
		Out:       out,
		Len:       0,
		FinalSize: 0,
		Stride:    params.Stride,
		Temp:      make([]byte, size),
		TempLen:   0,
		Freq:      *ascii.NewFrequency(),
	}
}

func (ef *EncodingFrame) pushFrame(frame []byte) {
	ef.Prev = ef.Curr
	ef.Curr = frame

	ef.CurrQT.UpdateBuffer(ef.Curr)
	if ef.Prev != nil {
		ef.PrevQT.UpdateBuffer(ef.Prev)
	}
}

type EncodingCall func(frame *EncodingFrame) error

type Encoder struct {
	encodings      []EncodingCall
	frames         []*EncodingFrame
	size           int
	quadTreeParams quad_tree.QuadTreeParam
	lastBestFrame  *EncodingFrame
	sequence       uint32
}

func NewEncoder(size int, treeParams quad_tree.QuadTreeParam) *Encoder {
	return &Encoder{
		encodings:      make([]EncodingCall, 0),
		frames:         make([]*EncodingFrame, 0),
		size:           size,
		quadTreeParams: treeParams,
		lastBestFrame:  nil,
		sequence:       0,
	}
}

func (e *Encoder) AddEncoding(encoding EncodingCall) {
	e.encodings = append(e.encodings, encoding)
	e.frames = append(e.frames, NewEncodingFrame(e.size, e.quadTreeParams))
}

// if this is the very first frame no matter what value for forceKeyFrame, it would be
// treated as a key frame
func (e *Encoder) PushFrame(rawData []byte, forceKeyFrame bool) *EncodingFrame {
	minSizeTarget := len(rawData)
	var outFrame *EncodingFrame
	for i, encoding := range e.encodings {
		frame := e.frames[i]
		frame.IsKeyFrame = forceKeyFrame
		if frame.Prev == nil {
			frame.IsKeyFrame = true
		}
		frame.pushFrame(rawData)
		err := encoding(frame)
		if err != nil {
			continue
		}
		if frame.FinalSize < minSizeTarget {
			minSizeTarget = frame.FinalSize
			outFrame = frame
		}
	}
	e.lastBestFrame = outFrame
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

// [FLAGS] [SEQ] [DATA LEN (varint)] [META (varint)] [DATA]
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
	if !frame.IsKeyFrame { // force refresh or the very first frame
		flags |= FlagIsDelta
	}
	switch frame.Encoding {
	case XOR_HUFFMAN:
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
