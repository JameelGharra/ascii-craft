package huffman

import (
	"container/heap"
	"errors"

	"github.com/JameelGharra/ascii-craft/server/ascii"
)

var ErrHuffmanEmptyFrame = errors.New("huffman encoding on empty frame")

func buildEncodeTree(freq *ascii.FreqTable) (*HuffmanNode, error) {
	if freq.TotalDifferentChars == 0 {
		return nil, ErrHuffmanEmptyFrame
	}
	if freq.TotalDifferentChars == 1 { // manually building for 1 entry (black screen)
		head := &HuffmanNode{
			value: 0,
			count: freq.Entries[0].Count,
		}
		head.left = &HuffmanNode{
			value: freq.Entries[0].Value,
			count: freq.Entries[0].Count,
		}
		return head, nil
	}
	nodes := make(PriorityQueue, freq.TotalDifferentChars)
	for i, point := range freq.Entries {
		nodes[i] = fromFreq(point)
	}
	heap.Init(&nodes)

	count := 1
	for len(nodes) > 1 {
		a := heap.Pop(&nodes).(*HuffmanNode)
		b := heap.Pop(&nodes).(*HuffmanNode)
		c := join(a, b)
		heap.Push(&nodes, c)
		count += 2
	}
	head := heap.Pop(&nodes).(*HuffmanNode)
	return head, nil
}
