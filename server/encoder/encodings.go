package encoder

import (
	"errors"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/encoding/huffman"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

var ErrEncoderExceededBufferSize = errors.New("encoder exceeded out buffer, not good")

func XorRLE(frame *EncodingFrame) error {
	if frame.Prev == nil {
		frame.Len = len(frame.Out) + 1
		return nil
	}
	ascii.Xor(frame.Curr, frame.Prev, frame.Temp)
	frame.RLE.Reset(frame.Out)
	frame.RLE.Write(frame.Temp)
	frame.RLE.Finish()
	frame.Len = frame.RLE.Size()
	frame.Encoding = XOR_RLE
	return nil
}

func createIteratorFromFrame(frame *EncodingFrame) (iter utils.IByteIterator) {
	if frame.Stride == 2 {
		iter = utils.New16BitIterator(frame.Curr)
	} else {
		iter = utils.New8BitIterator(frame.Curr)
	}
	return iter
}

func Huffman(frame *EncodingFrame) error {
	if frame.Prev == nil {
		frame.Len = len(frame.Out) + 1
		return nil
	}
	ascii.Xor(frame.Prev, frame.Curr, frame.Temp)
	frame.Freq.Reset()
	frame.Freq.Count(utils.New8BitIterator(frame.Temp))
	huff, err := huffman.NewHuffman(&frame.Freq)
	if err != nil {
		return err
	}
	bitLen, err := huff.Encode(utils.New8BitIterator(frame.Temp), frame.Out)
	if err != nil {
		return err
	}
	var byteLen int
	if bitLen%8 != 0 {
		byteLen = bitLen/8 + 1
	} else {
		byteLen = bitLen / 8
	}
	finalSize := byteLen + huff.TreeSize
	if finalSize > len(frame.Curr) {
		frame.Len = len(frame.Curr) + 1
		return nil
	}

	frame.Len = finalSize
	frame.Encoding = HUFFMAN
	return nil
}
