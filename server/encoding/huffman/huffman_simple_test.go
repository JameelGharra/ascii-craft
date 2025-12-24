package huffman

import (
	"bytes"
	"testing"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

func constructSimpleFreqTable() *ascii.FreqTable {
	freq := ascii.NewFrequency()
	freq.Count(utils.New8BitIterator(
		[]byte{
			'A', 'A', 'A',
			'B', 'B',
			'C', 'D',
		}))

	return freq
}

func TestSimpleHuffman(t *testing.T) {

	res := NewHuffmanEncodeResultTable()
	freq := constructSimpleFreqTable()
	data, err := getEncodeTree(freq, res)
	if err != nil {
		t.Fatalf("Failed to encode huffman: %v", err)
	}
	expectedSerialization := []byte{
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
	}
	if !bytes.Equal(data, expectedSerialization) {
		t.Fatalf("Huffman data encoded is incorrect. got: %v, expected: %v", data, expectedSerialization)
	}
	// checking table results
	expectedCodes := map[int][]byte{
		'A': {0},
		'B': {1, 0},
		'D': {1, 1, 0},
		'C': {1, 1, 1},
	}
	for value, expectedCode := range expectedCodes {
		code, exists := res.ValToCode[value]
		if !exists {
			t.Fatalf("Expected huffman code for value %d to exist", value)
		}
		if !bytes.Equal(code, expectedCode) {
			t.Fatalf("Expected huffman code for value %d to be %v, got %v", value, expectedCode, code)
		}
	}
	if res.CodeMaxLen != 3 {
		t.Fatalf("Expected max huffman code length to be 3, got %d", res.CodeMaxLen)
	}
}

func TestSimpleHuffmanEncodeStream(t *testing.T) {
	freq := constructSimpleFreqTable()
	huffman, err := NewHuffman(freq)
	if err != nil {
		t.Fatalf("Failed to create huffman: %v", err)
	}
	dataToEncode := utils.New8BitIterator(
		[]byte{
			'A', 'B', 'C', 'D',
		})
	out := make([]byte, 10) // out should be 2 bytes here, just seeing if bitlength logic works
	bitLength, err := huffman.Encode(dataToEncode, out)
	if err != nil {
		t.Fatalf("Failed to encode data: %v", err)
	}
	// A(0) B(10) C(111) D(110) should give 0b010111110 where the last zero is padding
	expectedOut := []byte{
		0b01011111,
		0,
	}
	var tillIndex int
	if bitLength%8 == 0 {
		tillIndex = bitLength / 8
	} else {
		tillIndex = bitLength/8 + 1
	}
	if !bytes.Equal(out[:tillIndex], expectedOut) {
		t.Fatalf("Expected encoded output to be %v, got %v", expectedOut, out[:tillIndex])
	}
}

func TestSimpleHuffmanDecodeStream(t *testing.T) {
	freq := constructSimpleFreqTable()
	huffman, err := NewHuffman(freq)
	if err != nil {
		t.Fatalf("Failed to create huffman: %v", err)
	}
	encodedData := []byte{
		0b01011111,
		0,
	}
	bitLength := 9
	writer := utils.ByteWriter8{}
	writerBuffer := make([]byte, 10)
	writer.Set(writerBuffer)
	err = huffman.Decode(encodedData, bitLength, &writer)
	if err != nil {
		t.Fatalf("Failed to decode data: %v", err)
	}
	expectedDecoded := []byte{
		'A', 'B', 'C', 'D',
	}
	if !bytes.Equal(writerBuffer[:writer.Len()], expectedDecoded) {
		t.Fatalf("Expected decoded data to be %v, got %v", expectedDecoded, writerBuffer[:writer.Len()])
	}
}
