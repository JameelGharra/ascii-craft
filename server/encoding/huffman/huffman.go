package huffman

import (
	"errors"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

const MinimumAllowingHuffmanChars = 2

var (
	ErrHuffmanValueNotFound       = errors.New("huffman value not found in encode table")
	ErrHuffmanExceedsOutputBuffer = errors.New("huffman encoding exceeds output buffer size")
	ErrHuffmanDecodeTraverse      = errors.New("huffman decode traverser error")
	ErrNoNeedHuffmanEncoding      = errors.New("no need for huffman encoding, data too small")
)

type Huffman struct {
	serializedTree      []byte
	HuffmanEncodeTable  *HuffmanEncodeResultTable
	treeDecodeTraverser *HuffmanTreeDecodeTraverser
}

func NewHuffman(freqTable *ascii.FreqTable) (*Huffman, error) {
	if freqTable.TotalDifferentChars < MinimumAllowingHuffmanChars {
		return nil, ErrNoNeedHuffmanEncoding
	}
	huffmanEncodeResultTable := NewHuffmanEncodeResultTable()
	serializedTree, err := getEncodeTree(freqTable, huffmanEncodeResultTable)
	if err != nil {
		return nil, err
	}
	return (&Huffman{
		serializedTree:      serializedTree,
		HuffmanEncodeTable:  huffmanEncodeResultTable,
		treeDecodeTraverser: NewHuffmanTreeDecodeTraverser(serializedTree),
	}), nil
}

// will return the number of bits written
func (h *Huffman) Encode(dataToEncode utils.IByteIterator, out []byte) (int, error) {
	byteIndexShift, currentEncoding := 7, byte(0)
	outIndex, dirty, bitLength := 0, false, 0
	for dataToEncode.HasNext() {
		value := dataToEncode.Next()
		huffmanCode, exists := h.HuffmanEncodeTable.ValToCode[value]
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
	if dirty { // padding
		if outIndex >= len(out) {
			return 0, ErrHuffmanExceedsOutputBuffer
		}
		out[outIndex] = currentEncoding
	}
	return bitLength, nil
}

func (h *Huffman) Decode(dataToDecode []byte, encodingBitLen int, writer utils.ByteWriter) error {
	currentBitIndex := 7
	var bitDirection byte
	for _, encodedByte := range dataToDecode {
		for currentBitIndex >= 0 && encodingBitLen > 0 {
			bitDirection = (encodedByte >> currentBitIndex) & 1
			isLeaf, value, err := h.treeDecodeTraverser.TraverseStep(bitDirection)
			if err != nil {
				return errors.Join(ErrHuffmanDecodeTraverse, err)
			}
			if isLeaf {
				err = writer.Write(value)
				if err != nil {
					return errors.Join(ErrHuffmanDecodeTraverse, err)
				}
			}
			currentBitIndex--
			encodingBitLen--
		}
		currentBitIndex = 7
	}
	return nil
}
