package encoder

import (
	"errors"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/encoding/huffman"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

var ErrEncoderExceededBufferSize = errors.New("encoder exceeded out buffer, not good")

func Raw(frame *EncodingFrame) error {
	copy(frame.Out, frame.Curr)
	frame.Len = len(frame.Curr)
	frame.FinalSize = frame.Len
	frame.Encoding = NONE
	return nil
}

func XorRLE(frame *EncodingFrame) error {
	var input []byte

	if frame.IsKeyFrame {
		input = frame.Curr
	} else {
		ascii.Xor(frame.Curr, frame.Prev, frame.Temp)
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

// func createIteratorFromFrame(frame *EncodingFrame) (iter utils.IByteIterator) {
// 	if frame.Stride == 2 {
// 		iter = utils.New16BitIterator(frame.Curr)
// 	} else {
// 		iter = utils.New8BitIterator(frame.Curr)
// 	}
// 	return iter
// }

// side note that len property does not include meta for e.g. huff code length tables
func Huffman(frame *EncodingFrame) error {
	var input []byte

	if frame.IsKeyFrame {
		input = frame.Curr
	} else {
		ascii.Xor(frame.Prev, frame.Curr, frame.Temp)
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

	if byteLen > len(frame.Curr) { // dont bother with huff at this point
		frame.Len = len(frame.Curr) + 1
		frame.FinalSize = frame.Len
		return nil
	}

	tableSize, err := huff.GetTableSize()
	frame.FinalSize = byteLen + tableSize
	frame.Len = byteLen
	frame.Huff = huff
	frame.Encoding = HUFFMAN
	return nil
}
