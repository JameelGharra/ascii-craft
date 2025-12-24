package huffman

import (
	"errors"

	"github.com/JameelGharra/ascii-craft/server/utils"
)

const (
	bitIndicatorLeft  = 0
	bitIndicatorRight = 1
)

var (
	ErrHuffmanDecodeInvalidBitDirection = errors.New("invalid bit direction for huffman decode traverser (has to be 0/1)")
	ErrHuffmanDecodeExceedsTreeBounds   = errors.New("huffman decode traverser exceeded tree bounds")
)

type HuffmanTreeDecodeTraverser struct {
	currentTreeIndex int // 16 bit for indexing
	serializedTree   []byte
}

func NewHuffmanTreeDecodeTraverser(serializedTree []byte) *HuffmanTreeDecodeTraverser {
	return &HuffmanTreeDecodeTraverser{
		currentTreeIndex: 0,
		serializedTree:   serializedTree,
	}
}

// left and right will return the next huff node start index
func (h *HuffmanTreeDecodeTraverser) goLeft() (int, error) {
	leftFieldStart := h.currentTreeIndex + sizePerNodeField
	if leftFieldStart >= len(h.serializedTree) {
		return 0, ErrHuffmanDecodeExceedsTreeBounds
	}
	return utils.Read16(h.serializedTree, leftFieldStart), nil
}

func (h *HuffmanTreeDecodeTraverser) isLeaf() (bool, error) {
	leftFieldStart := h.currentTreeIndex + sizePerNodeField
	rightFieldStart := h.currentTreeIndex + 2*sizePerNodeField
	if leftFieldStart >= len(h.serializedTree) || rightFieldStart >= len(h.serializedTree) {
		return false, ErrHuffmanDecodeExceedsTreeBounds
	}
	return (utils.Read16(h.serializedTree, leftFieldStart) == 0 &&
		utils.Read16(h.serializedTree, rightFieldStart) == 0), nil
}
func (h *HuffmanTreeDecodeTraverser) goRight() (int, error) {
	rightFieldStart := h.currentTreeIndex + 2*sizePerNodeField
	if rightFieldStart >= len(h.serializedTree) {
		return 0, ErrHuffmanDecodeExceedsTreeBounds
	}
	return utils.Read16(h.serializedTree, rightFieldStart), nil
}
func (h *HuffmanTreeDecodeTraverser) TraverseStep(bitDirection byte) (isLeaf bool, value int, err error) {
	switch bitDirection {
	case bitIndicatorLeft:
		h.currentTreeIndex, err = h.goLeft()
	case bitIndicatorRight:
		h.currentTreeIndex, err = h.goRight()
	default:
		return false, 0, ErrHuffmanDecodeInvalidBitDirection
	}

	if err != nil {
		return false, 0, err
	}
	isLeaf, err = h.isLeaf()
	if err != nil {
		return false, 0, err
	}
	value = utils.Read16(h.serializedTree, h.currentTreeIndex)
	if isLeaf {
		h.currentTreeIndex = 0 // for next run
	}
	return isLeaf, value, nil
}
