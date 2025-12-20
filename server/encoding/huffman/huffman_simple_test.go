package huffman

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/JameelGharra/ascii-craft/server/ascii"
)

func TestSimpleHuffman(t *testing.T) {
	freq := ascii.NewFrequency()
	freq.Count([]byte{
		'A', 'A', 'A',
		'B', 'B',
		'C', 'D',
	})

	data, err := Encode(freq)
	if err != nil {
		t.Fatalf("Failed to encode huffman: %v", err)
	}
	fmt.Printf("huffman data: %v\n", data)
	if !bytes.Equal(data, []byte{
		0, 3, 6,
		'A', 0, 0, // 0
		0, 9, 12,
		'B', 0, 0, // 10
		0, 15, 18,
		'D', 0, 0, // 110
		'C', 0, 0, // 111
	}) {
		t.Fatalf("Huffman data encoded is incorrect.")
	}
}
