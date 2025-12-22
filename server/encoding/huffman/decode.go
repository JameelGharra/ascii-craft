package huffman

import (
	"errors"

	"github.com/JameelGharra/ascii-craft/server/utils"
)

const InvalidDecodeResult = -1 // nor color or character can have that value
var ErrHuffmanDecodeInvalidBit = errors.New("huffman decode invalid bit input")
var ErrHuffmanDecodeNotFound = errors.New("huffman decode did not find a valid value")
var ErrHuffmanDecodeInvalidStepCount = errors.New("huffman decode exceeded valid step count")

type HuffmanDecoder struct {
	serializedTree    *[]byte
	serializationSize int
}

func NewHuffmanDecoder(serializedTree *[]byte) *HuffmanDecoder {
	return &HuffmanDecoder{
		serializedTree:    serializedTree,
		serializationSize: len(*serializedTree),
	}
}

func isLeaf(nodeIndex int, serializedTree *[]byte) bool {
	leftIndex := utils.Read16(*serializedTree, nodeIndex+sizePerNodeField)
	rightIndex := utils.Read16(*serializedTree, nodeIndex+2*sizePerNodeField)
	return leftIndex == 0 && rightIndex == 0
}

func (h *HuffmanDecoder) decodeAttempt(encodedBits utils.IByteIterator) (int, error) {
	currentIndex, stepsCount := 0, 0
	for encodedBits.HasNext() {
		bit := encodedBits.Next()
		nextIndex := 0
		switch bit {
		case 0:
			nextIndex = utils.Read16(*h.serializedTree, currentIndex+sizePerNodeField)
		case 1:
			nextIndex = utils.Read16(*h.serializedTree, currentIndex+2*sizePerNodeField)
		default:
			return -1, ErrHuffmanDecodeInvalidBit
		}
		if nextIndex == 0 {
			return InvalidDecodeResult, ErrHuffmanDecodeNotFound
		}
		// checked now to not eat the next bit for nothing
		if isLeaf(nextIndex, h.serializedTree) {
			return utils.Read16(*h.serializedTree, nextIndex), nil
		}
		currentIndex = nextIndex
		stepsCount++
		if stepsCount > h.serializationSize {
			return InvalidDecodeResult, ErrHuffmanDecodeInvalidStepCount
		}
	}
	return InvalidDecodeResult, ErrHuffmanDecodeNotFound
}

func (h *HuffmanDecoder) Decode(encodedBits utils.IByteIterator, encodedBitsLen int, writer utils.ByteWriter) error {
	// for an edge case where the tree has 1 node
	if isLeaf(0, h.serializedTree) {
		for range encodedBitsLen { // a val is 1 bit here
			writer.Write(utils.Read16(*h.serializedTree, 0))
		}
		return nil
	}
	for encodedBits.HasNext() {
		result, err := h.decodeAttempt(encodedBits)
		if err != nil {
			return err
		}
		writer.Write(result)
	}
	return nil
}
