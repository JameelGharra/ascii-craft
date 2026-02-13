package protocol

import (
	"bytes"
	"testing"
)

func TestPacketOverflow(t *testing.T) {
	buf := make([]byte, 1) // tiny buff
	pb := NewPacketBuilder(buf)

	// supposed to demand 2 bytes
	if err := pb.WriteVarint(300); err == nil {
		t.Errorf("Expected ErrPacketTooSmall, got %v", err)
	}

	if err := pb.WriteVarint(10); err != nil {
		t.Errorf("Should be able to write 1 byte varint, got %v", err)
	}

	// demanding a next write should overflow
	if err := pb.WriteByte(0xFF); err != ErrPacketTooSmall {
		t.Errorf("Expected error on full buffer write, got %v", err)
	}
}

func TestWriteVarint(t *testing.T) {
	tests := []struct {
		name     string
		input    uint32
		expected []byte
	}{
		{"Zero", 0, []byte{0x00}},
		{"OneByteMax", 127, []byte{0x7F}},
		{"TwoBytesMin", 128, []byte{0x80, 0x01}},    // 128 = 1000 0000 -> [000 0000] [1]
		{"FrameSeqSample", 300, []byte{0xAC, 0x02}}, // 300 = 10 0101100
		{"MaxUint32", 0xFFFFFFFF, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x0F}},
	}

	buf := make([]byte, 10)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewPacketBuilder(buf)
			err := pb.WriteVarint(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			got := pb.Bytes()
			if !bytes.Equal(got, tt.expected) {
				t.Errorf("Input: %d, Expected: %X, Got: %X", tt.input, tt.expected, got)
			}
		})
	}
}

func BenchmarkWriteVarint(b *testing.B) {
	buf := make([]byte, 10)
	pb := NewPacketBuilder(buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pb.offset = 0
		// benching a typical 2 byte
		_ = pb.WriteVarint(4096) // with current 212x66 resolution thats not bad
	}
}
