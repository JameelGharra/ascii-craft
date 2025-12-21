package huffman

import (
	"fmt"

	"github.com/JameelGharra/ascii-craft/server/ascii"
)

const SizePerNodeField = 2

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

// func (h *HuffmanNode) debug(indent int) string {
// 	indentStr := strings.Repeat(" ", indent*2)
// 	if h == nil {
// 		return fmt.Sprintf("%s-> nil\n", indentStr)
// 	}
// 	return fmt.Sprintf("%s->%s\n", indentStr, h.String()) +
// 		h.left.debug(indent+1) +
// 		h.right.debug(indent+1)
// }

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
