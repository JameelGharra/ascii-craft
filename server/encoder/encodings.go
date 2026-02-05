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
	// for canonical tree
	frame.RLE.Reset(frame.Temp)
	var table [256]byte
	for val, length := range huff.ValToCodeLength {
		table[val] = byte(length)
	}
	frame.RLE.Write(table[:])
	frame.RLE.Finish()

	var tableSize = frame.RLE.Size()
	var sparseSize = frame.Freq.TotalDifferentChars * 2
	if sparseSize < tableSize {
		tableSize = sparseSize
	}
	if tableSize > 256 {
		tableSize = 256
	}
	// fmt.Printf("Huffman tree size: %d bytes\n", frame.RLE.Size())
	// fmt.Printf("Huffman encoded data size: %d bytes\n", byteLen)
	// result, _ := frame.RLE.Result()
	// fmt.Printf("RLE result: %v\n", result)
	// fmt.Printf("Frequency count for frame: %v\n", frame.Freq.TotalDifferentChars)
	// fmt.Printf("Picked size for table: %d bytes\n", tableSize)
	//
	finalSize := byteLen + tableSize
	if finalSize > len(frame.Curr) {
		frame.Len = len(frame.Curr) + 1
		return nil
	}

	frame.Len = finalSize
	frame.Encoding = HUFFMAN
	return nil
}
