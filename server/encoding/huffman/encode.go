package huffman

import (
	"container/heap"
	"errors"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

var ErrHuffmanTooLarge = errors.New("huffman tree is so large")

const HUFFMAN_ENCODE_LENGTH = 5

func Encode(freq *ascii.FreqTable) ([]byte, error) {
	nodes := make(PriorityQueue, freq.TotalDifferentChars)
	for i, point := range freq.Points {
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
	data := make([]byte, count*HUFFMAN_ENCODE_LENGTH)
	serializeTree(head, data, 0)

	return data, nil
}

func serializeTree(node *HuffmanNode, data []byte, index int) int { // zero waste space strategy
	if node == nil {
		return index
	}

	utils.Assert(index+2 < len(data), "Index will exceed data length.")

	leftIndex := index + HUFFMAN_ENCODE_LENGTH
	data[index] = node.value

	data[index+1] = 0
	data[index+2] = 0

	next := leftIndex

	if node.left != nil {
		data[index+1] = byte(leftIndex)
		next = serializeTree(node.left, data, leftIndex)
	}

	if node.right != nil {
		data[index+2] = byte(next)
		next = serializeTree(node.right, data, next)
	}
	return next
}
