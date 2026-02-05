package huffman

import (
	"fmt"

	"github.com/JameelGharra/ascii-craft/server/ascii"
)

const (
	sizePerNodeField = 2
)

type HuffmanNode struct {
	value int
	count int
	left  *HuffmanNode
	right *HuffmanNode
}

func (h *HuffmanNode) String() string {
	if h == nil {
		return "nil"
	}
	return fmt.Sprintf("HuffmanNode(value=%d, count=%d)", h.value, h.count)
}

func join(a, b *HuffmanNode) *HuffmanNode {
	return &HuffmanNode{
		value: 0,
		count: a.count + b.count,
		left:  a,
		right: b,
	}
}

func fromFreq(freq *ascii.FreqEntry) *HuffmanNode {
	return &HuffmanNode{
		value: freq.Value,
		count: freq.Count,
		left:  nil,
		right: nil,
	}
}

func (h *HuffmanNode) calculateDepths(currentDepth int, lengths map[int]int) {
	if h.left == nil && h.right == nil {
		lengths[h.value] = currentDepth
		return
	}
	// technically added the null checks just for the edge case of single node
	if h.left != nil {
		h.left.calculateDepths(currentDepth+1, lengths)
	}
	if h.right != nil {
		h.right.calculateDepths(currentDepth+1, lengths)
	}
}
