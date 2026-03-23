package huffman

import (
	"errors"
	"sort"

	"github.com/JameelGharra/ascii-craft/server/encoding/rle"
)

const (
	depthUpBoundary = 33 // max depth usually won't exceed 32
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
	out := make([]byte, length)
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

const (
	tableModeRaw          byte = 0
	tableModeRLE          byte = 1
	tableModeSparse       byte = 2
	canonicalTableRawSize      = 256
)

var (
	ErrInvalidCanonicalTable               = errors.New("invalid canonical table, cannot be packed")
	ErrEmptyCanonicalTable                 = errors.New("empty canonical table provided")
	tableRLEBuffer           []byte        = make([]byte, canonicalTableRawSize)
	tableSparseBuffer        []byte        = make([]byte, canonicalTableRawSize) // wont bother using if input not half of raw so its ok
	tableRawBuffer           []byte        = make([]byte, canonicalTableRawSize)
	RLEEncoder               *rle.AsciiRLE = rle.NewAsciiRLE()
)

// it picks the best strategy out of the meta modes and pack accordingly
// would return the meta mode, amoutnt of bytes written and an error if any
// current strategies are raw, rle and spare (pairs of value and length)
func packCanonicalTable(codeLengths map[int]int) (byte, []byte, error) {
	if len(codeLengths) == 0 {
		return 0, nil, ErrEmptyCanonicalTable
	}
	if len(codeLengths) > canonicalTableRawSize {
		return 0, nil, ErrInvalidCanonicalTable
	}
	for i := 0; i < canonicalTableRawSize; i++ {
		tableRawBuffer[i] = 0
	}
	RLEEncoder.Reset(tableRLEBuffer)
	for val, length := range codeLengths {
		if val < 0 || val >= canonicalTableRawSize {
			return 0, nil, ErrInvalidCanonicalTable
		}
		tableRawBuffer[val] = byte(length)
	}
	RLEEncoder.Write(tableRawBuffer)
	RLEEncoder.Finish()
	rleSize := RLEEncoder.Size()
	sparseSize := canonicalTableRawSize
	if len(codeLengths) < canonicalTableRawSize/2 { // would send raw if equal
		sparseSize = 0
		// I know that iterating over the map is faster, but keeping
		// a deterministic order sounds a better idea for tests
		for i := 0; i < canonicalTableRawSize; i++ {
			if tableRawBuffer[i] > 0 {
				tableSparseBuffer[sparseSize] = byte(i)
				sparseSize++
				tableSparseBuffer[sparseSize] = tableRawBuffer[i]
				sparseSize++
			}
		}
	}
	// this would rather rle over sparse if they equal
	if rleSize < canonicalTableRawSize && rleSize <= sparseSize {
		return tableModeRLE, tableRLEBuffer[:rleSize], nil
	}
	if sparseSize < canonicalTableRawSize {
		return tableModeSparse, tableSparseBuffer[:sparseSize], nil
	}
	return tableModeRaw, tableRawBuffer, nil
}
