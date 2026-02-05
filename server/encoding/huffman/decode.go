package huffman

import (
	"errors"
)

const (
	bitIndicatorLeft  = 0
	bitIndicatorRight = 1
)

var (
	ErrHuffmanDecodeInvalidBitDirection = errors.New("invalid bit direction for huffman decode traverser (has to be 0/1)")
	ErrHuffmanDecodeExceedsTreeBounds   = errors.New("huffman decode traverser exceeded tree bounds")
)

type HuffmanDecodeNode struct {
	left  *HuffmanDecodeNode
	right *HuffmanDecodeNode
	value int // would be -1 if not leaf
}

type HuffmanTreeDecodeTraverser struct {
	decodeTreeRoot *HuffmanDecodeNode
	currentNode    *HuffmanDecodeNode
}

func NewHuffmanTreeDecodeTraverser(root *HuffmanDecodeNode) *HuffmanTreeDecodeTraverser {
	return &HuffmanTreeDecodeTraverser{
		decodeTreeRoot: root,
		currentNode:    root,
	}
}

func (h *HuffmanTreeDecodeTraverser) TraverseStep(bitDirection byte) (isLeaf bool, value int, err error) {
	switch bitDirection {
	case bitIndicatorLeft:
		h.currentNode = h.currentNode.left
	case bitIndicatorRight:
		h.currentNode = h.currentNode.right
	default:
		return false, 0, ErrHuffmanDecodeInvalidBitDirection
	}
	if h.currentNode == nil {
		return false, 0, ErrHuffmanDecodeExceedsTreeBounds
	}
	isLeaf = h.currentNode.left == nil && h.currentNode.right == nil
	if isLeaf {
		value = h.currentNode.value
		h.currentNode = h.decodeTreeRoot
	}
	return isLeaf, value, nil
}
