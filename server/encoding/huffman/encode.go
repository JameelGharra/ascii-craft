package huffman

import (
	"container/heap"
	"errors"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

var ErrHuffmanTooLarge = errors.New("huffman tree is so large")

const HuffmanEncodeLength = 6

func Encode(freq *ascii.FreqTable) ([]byte, error) {
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
	data := make([]byte, count*HuffmanEncodeLength)
	serializeTree(head, data, 0)

	return data, nil
}

func serializeTree(node *HuffmanNode, data []byte, index int) int { // zero waste space strategy
	if node == nil {
		return index
	}

	utils.Assert(index+5 < len(data), "Index will exceed data length.")

	leftIndex := index + HuffmanEncodeLength

	utils.Write16(data, index, node.value)

	leftFieldStart := index + SizePerNodeField
	rightFieldStart := leftFieldStart + SizePerNodeField
	utils.Write16(data, leftFieldStart, 0)
	utils.Write16(data, rightFieldStart, 0)

	next := leftIndex

	if node.left != nil {
		utils.Write16(data, leftFieldStart, leftIndex)
		next = serializeTree(node.left, data, leftIndex)
	}

	if node.right != nil {
		utils.Write16(data, rightFieldStart, next)
		next = serializeTree(node.right, data, next)
	}
	return next
}
