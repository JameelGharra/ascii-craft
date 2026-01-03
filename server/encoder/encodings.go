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
	frame.Freq.Reset()
	iter := createIteratorFromFrame(frame) // for current frame though
	frame.Freq.Count(iter)
	huff, err := huffman.NewHuffman(&frame.Freq)
	if err != nil {
		return err
	}
	bitLen, err := huff.Encode(createIteratorFromFrame(frame), frame.Out)
	if err != nil {
		return err
	}
	var byteLen int
	if bitLen%8 != 0 {
		byteLen = bitLen/8 + 1
	} else {
		byteLen = bitLen / 8
	}
	if huff.TreeSize+byteLen > len(frame.Curr) {
		frame.Len = len(frame.Curr) + 1
	}
	frame.Len = huff.TreeSize + byteLen
	frame.Encoding = HUFFMAN
	return nil
}
