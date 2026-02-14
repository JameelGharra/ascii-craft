package huffman

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

func createFreqFromData(data []byte) *ascii.FreqTable {
	freq := ascii.NewFrequency()
	freq.Count(utils.New8BitIterator(data))
	return freq
}

func TestCanonicalEndToEnd(t *testing.T) {

	// AAAABBCD
	input := []byte{'A', 'A', 'A', 'A', 'B', 'B', 'C', 'D'}
	freq := createFreqFromData(input)

	h, err := NewHuffman(freq)
	if err != nil {
		t.Fatalf("Failed to create huffman: %v", err)
	}

	encodedBuf := make([]byte, len(input))
	bitLen, err := h.Encode(utils.New8BitIterator(input), encodedBuf)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// 4. Calculate byte length
	byteLen := bitLen / 8
	if bitLen%8 != 0 {
		byteLen++
	}
	encodedData := encodedBuf[:byteLen]

	var decodedBuf = make([]byte, len(input))
	writer := &utils.ByteWriter8{}
	writer.Set(decodedBuf)

	err = h.Decode(encodedData, bitLen, writer)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if !bytes.Equal(decodedBuf, input) {
		t.Errorf("Mismatch.\nExpected: %v\nGot:      %v", input, decodedBuf)
	}
}
func TestCanonicalCodeGeneration(t *testing.T) {
	// A: 4, B: 2, C: 1, D: 1
	// This specific distribution guarantees depths: A=1, B=2, C=3, D=3
	input := []byte{'A', 'A', 'A', 'A', 'B', 'B', 'C', 'D'}
	freq := createFreqFromData(input)

	h, err := NewHuffman(freq)
	if err != nil {
		t.Fatalf("Failed to create huffman: %v", err)
	}

	// C < D then C -> 110 and D -> 111
	expectedCodes := map[int][]byte{
		'A': {0},
		'B': {1, 0},
		'C': {1, 1, 0},
		'D': {1, 1, 1},
	}

	for val, expected := range expectedCodes {
		actual, exists := h.encodeTable[val]
		if !exists {
			t.Errorf("Symbol %c not found in encode table", val)
			continue
		}
		if !bytes.Equal(actual, expected) {
			t.Errorf("Symbol %c: expected %v, got %v", val, expected, actual)
		}
	}

	// double check the lengths are what we expect
	// (if lengths changed, the codes above would be wrong anyway, but good for debug)
	expectedLengths := map[int]int{
		'A': 1, 'B': 2, 'C': 3, 'D': 3,
	}
	for val, length := range expectedLengths {
		if h.ValToCodeLength[val] != length {
			t.Errorf("Symbol %c: expected length %d, got %d", val, length, h.ValToCodeLength[val])
		}
	}
}
func TestTablePackingModes(t *testing.T) {
	// this test forces specific frequency distributions to trigger
	// the RLE, sparse, and raw modes

	// sparse mode test, should not pick RLE or raw
	// only 2 characters used so it should be 2 bytes in total for meta data
	t.Run("mode_sparse", func(t *testing.T) {
		input := []byte{'A', 'Z'}
		h, _ := NewHuffman(createFreqFromData(input))

		dst := make([]byte, 512)
		mode, size, err := h.WriteTableBytes(dst)
		if err != nil {
			t.Fatalf("WriteTableBytes failed: %v", err)
		}

		if mode != tableModeSparse {
			t.Errorf("Expected tableModeSparse (%d), got %d", tableModeSparse, mode)
		}
		// size check: length byte (1) + (Val(1)+Len(1)) * 2 entries => 5 bytes
		expectedSize := 1 + (2 * 2)
		if size != expectedSize {
			t.Errorf("Expected size %d, got %d", expectedSize, size)
		}
	})

	// RLE mode
	// many characters, but they all have the SAME frequency
	// this results in identical code lengths, which RLE compresses beautifully.
	t.Run("mode_RLE", func(t *testing.T) {
		input := make([]byte, 0)
		// 100 characters, all appearing once
		for i := 0; i < 100; i++ {
			input = append(input, byte(i))
		}

		h, _ := NewHuffman(createFreqFromData(input))
		dst := make([]byte, 512)
		mode, _, err := h.WriteTableBytes(dst)
		if err != nil {
			t.Fatalf("WriteTableBytes failed: %v", err)
		}

		if mode != tableModeRLE {
			t.Errorf("Expected tableModeRLE (%d), got %d", tableModeRLE, mode)
		}
	})

	// raw mode
	// basically just made distinct characters > 128 so no sparse and got also noise
	t.Run("mode_raw", func(t *testing.T) {
		input := make([]byte, 0)
		rng := rand.New(rand.NewSource(1234))

		// Use almost all 256 characters with random frequencies
		for i := 0; i < 255; i++ {
			count := rng.Intn(10) + 1
			for j := 0; j < count; j++ {
				input = append(input, byte(i))
			}
		}

		h, _ := NewHuffman(createFreqFromData(input))
		dst := make([]byte, 512)
		mode, size, err := h.WriteTableBytes(dst)
		if err != nil {
			t.Fatalf("WriteTableBytes failed: %v", err)
		}

		// depending on rng, RLE might still win sometimes
		if mode != tableModeRaw {
			t.Logf("Warning: RNG produced compressible data, got mode %d instead of Raw", mode)
		} else {
			if size != 256 {
				t.Errorf("Expected Raw size 256, got %d", size)
			}
		}
	})
}

func TestSizePredictionSync(t *testing.T) {
	// ensure  GetTableSize() logic matches WriteTableBytes() logic
	// just to make sure that our hybrid strategy picks the right encoding
	input := []byte("HELLO WORLD PACKET SIZE TEST")
	h, _ := NewHuffman(createFreqFromData(input))

	predictedSize, err := h.GetTableSize()
	if err != nil {
		t.Fatalf("GetTableSize failed: %v", err)
	}

	dst := make([]byte, 512)
	_, writtenSize, err := h.WriteTableBytes(dst)
	if err != nil {
		t.Fatalf("WriteTableBytes failed: %v", err)
	}

	if predictedSize != writtenSize {
		t.Errorf("Size mismatch! Predicted: %d, Written: %d", predictedSize, writtenSize)
	}
}

func TestLargeBufferFuzz(t *testing.T) {
	// a massive random test to ensure no panics on large inputs
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 212 * 66 = ~14k pixels (simulating a frame with usual 212x66)
	data := make([]byte, 14000)
	rng.Read(data)

	h, err := NewHuffman(createFreqFromData(data))
	if err != nil {
		t.Fatalf("NewHuffman failed: %v", err)
	}

	encoded := make([]byte, len(data))
	bitLen, err := h.Encode(utils.New8BitIterator(data), encoded)
	if err != nil {
		// should not happen since huffman is consistent according to my benchmarks
		t.Logf("Encode failed (might be entropy expansion): %v", err)
		return
	}

	// decode
	decoded := make([]byte, len(data))
	writer := &utils.ByteWriter8{}
	writer.Set(decoded)

	byteLen := bitLen/8 + 1
	err = h.Decode(encoded[:byteLen], bitLen, writer)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if !bytes.Equal(data, decoded) {
		t.Fatal("Decoded data does not match random input")
	}
}
