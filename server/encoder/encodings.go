package encoder

import (
	"errors"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/encoding/huffman"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

var ErrEncoderExceededBufferSize = errors.New("encoder exceeded out buffer, not good")

func Raw(frame *EncodingFrame, curr, prev []byte) error {
	copy(frame.Out, curr)
	frame.Len = len(curr)
	frame.FinalSize = frame.Len
	frame.Encoding = NONE
	return nil
}

func XorRLE(frame *EncodingFrame, curr, prev []byte) error {
	var input []byte

	if frame.IsKeyFrame {
		input = curr
	} else {
		ascii.Xor(prev, curr, frame.Temp)
		input = frame.Temp
	}
	frame.RLE.Reset(frame.Out)
	frame.RLE.Write(input)
	frame.RLE.Finish()
	frame.Len = frame.RLE.Size()
	frame.FinalSize = frame.Len
	frame.Encoding = XOR_RLE
	return nil
}

// side note that len property does not include meta for e.g. huff code length tables
func Huffman(frame *EncodingFrame, curr, prev []byte) error {
	var input []byte

	if frame.IsKeyFrame {
		input = curr
	} else {
		ascii.Xor(prev, curr, frame.Temp)
		input = frame.Temp
	}
	frame.Freq.Reset()
	frame.Freq.Count(utils.New8BitIterator(input))
	huff, err := huffman.NewHuffman(&frame.Freq)
	if err != nil {
		return err
	}
	bitLen, err := huff.Encode(utils.New8BitIterator(input), frame.Out)
	if err != nil {
		return err
	}
	var byteLen int
	if bitLen%8 != 0 {
		byteLen = bitLen/8 + 1
	} else {
		byteLen = bitLen / 8
	}

	if byteLen > len(curr) { // dont bother with huff at this point
		frame.Len = len(curr) + 1
		frame.FinalSize = frame.Len
		return nil
	}

	tableSize, err := huff.GetTableSize()
	frame.FinalSize = byteLen + tableSize
	frame.Len = byteLen
	frame.Huff = huff
	if frame.IsKeyFrame {
		frame.Encoding = HUFFMAN
	} else {
		frame.Encoding = XOR_HUFFMAN
	}
	return nil
}
