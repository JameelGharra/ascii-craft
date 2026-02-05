package huffman

import (
	"container/heap"
	"errors"

	"github.com/JameelGharra/ascii-craft/server/ascii"
)

var ErrHuffmanEmptyFrame = errors.New("huffman encoding on empty frame")

// const HuffmanEncodeLength = 6

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

// func serializeTree(node *HuffmanNode, data *[]byte, res *HuffmanEncodeResultTable, index int) int { // zero waste space strategy
// 	if node == nil {
// 		return index
// 	}

// 	utils.Assert(index+5 < len(*data), "Index will exceed data length.")

// 	if node.left == nil && node.right == nil {
// 		res.Update(node.value)
// 	}

// 	leftIndex := index + HuffmanEncodeLength

// 	utils.Write16(*data, index, node.value)

// 	leftFieldStart := index + sizePerNodeField
// 	rightFieldStart := leftFieldStart + sizePerNodeField
// 	utils.Write16(*data, leftFieldStart, 0)
// 	utils.Write16(*data, rightFieldStart, 0)

// 	next := leftIndex

// 	if node.left != nil {
// 		utils.Write16(*data, leftFieldStart, leftIndex)
// 		res.Left()
// 		next = serializeTree(node.left, data, res, leftIndex)
// 		res.Back()
// 	}

// 	if node.right != nil {
// 		utils.Write16(*data, rightFieldStart, next)
// 		res.Right()
// 		next = serializeTree(node.right, data, res, next)
// 		res.Back()
// 	}
// 	return next
// }
