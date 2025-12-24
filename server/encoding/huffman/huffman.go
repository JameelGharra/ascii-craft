package huffman

import (
	"errors"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

var ErrHuffmanValueNotFound = errors.New("huffman value not found in encode table")
var ErrHuffmanExceedsOutputBuffer = errors.New("huffman encoding exceeds output buffer size")

type Huffman struct {
	serializedTree []byte
	HuffmanResult  *HuffmanEncodeResultTable
}

func NewHuffman(freqTable *ascii.FreqTable) (*Huffman, error) {
	huffmanEncodeResultTable := NewHuffmanEncodeResultTable()
	serializedTree, err := getEncodeTree(freqTable, huffmanEncodeResultTable)
	if err != nil {
		return nil, err
	}
	return (&Huffman{
		serializedTree: serializedTree,
		HuffmanResult:  huffmanEncodeResultTable,
	}), nil
}

// will return the number of bits written
func (h *Huffman) Encode(dataToEncode utils.IByteIterator, out []byte) (int, error) {
	byteIndexShift, currentEncoding := 7, byte(0)
	outIndex, dirty, bitLength := 0, false, 0
	for dataToEncode.HasNext() {
		value := dataToEncode.Next()
		huffmanCode, exists := h.HuffmanResult.ValToCode[value]
		if !exists {
			return 0, ErrHuffmanValueNotFound
		}
		for _, bit := range huffmanCode { // msb to lsb here
			dirty = true
			currentEncoding = currentEncoding | bit<<byteIndexShift
			byteIndexShift--
			bitLength++
			if byteIndexShift < 0 {
				if outIndex >= len(out) {
					return 0, ErrHuffmanExceedsOutputBuffer
				}
				out[outIndex] = currentEncoding
				outIndex++
				currentEncoding = 0
				dirty = false
				byteIndexShift = 7
			}
		}
	}
	if dirty {
		if outIndex >= len(out) {
			return 0, ErrHuffmanExceedsOutputBuffer
		}
		out[outIndex] = currentEncoding
	}
	return bitLength, nil
}
