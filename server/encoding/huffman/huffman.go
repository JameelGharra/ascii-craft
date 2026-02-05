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
	ErrHuffmanFailedToWrite       = errors.New("huffman failed to write to output")
	ErrHuffmanPackingDoesntFit    = errors.New("huffman packing doesn't fit in output buffer")
)

type Huffman struct {
	encodeTable         map[int][]byte
	treeDecodeTraverser *HuffmanTreeDecodeTraverser
	ValToCodeLength     map[int]int
}

func NewHuffman(freqTable *ascii.FreqTable) (*Huffman, error) {
	// if freqTable.TotalDifferentChars < MinimumAllowingHuffmanChars {
	// 	return nil, ErrNoNeedHuffmanEncoding
	// }
	huffmanTreeRoot, err := buildEncodeTree(freqTable)
	if err != nil {
		return nil, err
	}
	valToCodeLength := make(map[int]int)
	huffmanTreeRoot.calculateDepths(0, valToCodeLength)
	valToCode := generateCanonicalCodes(valToCodeLength)
	decodeTreeRoot := buildCanonicalDecodeTree(valToCode)
	return (&Huffman{
		encodeTable:         valToCode,
		treeDecodeTraverser: NewHuffmanTreeDecodeTraverser(decodeTreeRoot),
		ValToCodeLength:     valToCodeLength,
	}), nil
}

// will return the number of bits written
func (h *Huffman) Encode(dataToEncode utils.IByteIterator, out []byte) (int, error) {
	byteIndexShift, currentEncoding := 7, byte(0)
	outIndex, dirty, bitLength := 0, false, 0
	for dataToEncode.HasNext() {
		value := dataToEncode.Next()
		huffmanCode, exists := h.encodeTable[value]
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

// would need this in client side, but decided to implement it anyway being hyped
func (h *Huffman) Decode(dataToDecode []byte, encodingBitLen int, writer utils.ByteWriter) error {
	currentBitIndex := 7
	var bitDirection byte
	stillOnIt := true
	for _, encodedByte := range dataToDecode {
		for currentBitIndex >= 0 && encodingBitLen > 0 {
			bitDirection = (encodedByte >> currentBitIndex) & 1
			isLeaf, value, err := h.treeDecodeTraverser.TraverseStep(bitDirection)
			if err != nil {
				return errors.Join(ErrHuffmanDecodeTraverse, err)
			}
			stillOnIt = true
			if isLeaf {
				err = writer.Write(value)
				if err != nil {
					return errors.Join(ErrHuffmanFailedToWrite, err)
				}
				stillOnIt = false
			}
			currentBitIndex--
			encodingBitLen--
		}
		currentBitIndex = 7
	}
	if stillOnIt {
		return ErrHuffmanDecodeTraverse
	}
	return nil
}

// encodes the huffman tree and bit length into the output buffer at the given offset
func IntoBytes(h *Huffman, bitLen int, out []byte, offset int) {

}
