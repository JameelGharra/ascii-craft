package huffman

import "sort"

var (
	depthUpBoundary = 33                            // max depth usually won't exceed 32
	bitBuffer       = make([]byte, depthUpBoundary) // just to avoid multiple allocations
)

func generateCanonicalCodes(codeLengths map[int]int) map[int][]byte {
	lengthCategories := make([][]int, depthUpBoundary)
	minLength, maxLength := depthUpBoundary, 0
	for value, length := range codeLengths {
		if length == 0 {
			continue
		}
		lengthCategories[length] = append(lengthCategories[length], value)
		if length < minLength {
			minLength = length
		}
		if length > maxLength {
			maxLength = length
		}
	}
	// internal sorting for each category
	for i := minLength; i <= maxLength; i++ {
		sort.Ints(lengthCategories[i])
	}
	valToCode := make(map[int][]byte)
	var currentCode int = 0
	for length := minLength; length <= maxLength; length++ {
		for _, value := range lengthCategories[length] {
			valToCode[value] = intToBits(currentCode, length)
			currentCode++
		}
		currentCode <<= 1 // most likely a zero on right
	}
	return valToCode
}

func intToBits(code, length int) []byte {
	out := bitBuffer[:length]
	for i := 0; i < length; i++ {
		bit := (code >> (length - i - 1)) & 1
		out[i] = byte(bit)
	}
	return out
}

func buildCanonicalDecodeTree(valToCode map[int][]byte) *HuffmanDecodeNode {
	root := &HuffmanDecodeNode{value: -1}
	for val, code := range valToCode {
		currentNode := root
		for _, bit := range code {
			if bit == 0 {
				if currentNode.left == nil {
					currentNode.left = &HuffmanDecodeNode{value: -1}
				}
				currentNode = currentNode.left
			} else {
				if currentNode.right == nil {
					currentNode.right = &HuffmanDecodeNode{value: -1}
				}
				currentNode = currentNode.right
			}
		}
		currentNode.value = val
	}
	return root
}
