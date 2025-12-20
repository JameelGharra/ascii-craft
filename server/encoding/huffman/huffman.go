package huffman

import (
	"fmt"
	"strings"

	"github.com/JameelGharra/ascii-craft/server/ascii"
)

type HuffmanNode struct {
	value byte
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

func (h *HuffmanNode) debug(indent int) string {
	indentStr := strings.Repeat(" ", indent*2)
	if h == nil {
		return fmt.Sprintf("%s-> nil\n", indentStr)
	}
	return fmt.Sprintf("%s->%s\n", indentStr, h.String()) +
		h.left.debug(indent+1) +
		h.right.debug(indent+1)
}

func fromValue(value byte) *HuffmanNode {
	return &HuffmanNode{
		value: value,
		count: 1,
		left:  nil,
		right: nil,
	}
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
