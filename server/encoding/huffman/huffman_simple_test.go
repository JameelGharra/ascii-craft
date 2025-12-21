package huffman

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

func TestSimpleHuffman(t *testing.T) {
	freq := ascii.NewFrequency()
	freq.Count(utils.New8BitIterator(
		[]byte{
			'A', 'A', 'A',
			'B', 'B',
			'C', 'D',
		}))

	data, err := Encode(freq)
	if err != nil {
		t.Fatalf("Failed to encode huffman: %v", err)
	}
	fmt.Printf("huffman data: %v\n", data)

	if !bytes.Equal(data, []byte{
		0, 0, 0, HuffmanEncodeLength, 0, HuffmanEncodeLength * 2,
		// 0, 3, 6,
		0, 'A', 0, 0, 0, 0, // 0
		// 'A', 0, 0, // 0
		0, 0, 0, HuffmanEncodeLength * 3, 0, HuffmanEncodeLength * 4,
		// 0, 9, 12,
		0, 'B', 0, 0, 0, 0, // 10
		// 'B', 0, 0, // 10
		0, 0, 0, HuffmanEncodeLength * 5, 0, HuffmanEncodeLength * 6,
		// 0, 15, 18,
		0, 'D', 0, 0, 0, 0, // 110
		// 'D', 0, 0, // 110
		0, 'C', 0, 0, 0, 0, // 111
		// 'C', 0, 0, // 111
	}) {
		t.Fatalf("Huffman data encoded is incorrect.")
	}
}
