package protocol

import (
	"errors"
)

var (
	ErrPacketTooSmall = errors.New("packet buffer too small for writing")
)

type PacketBuilder struct {
	buffer []byte
	offset int
}

func NewPacketBuilder(buffer []byte) *PacketBuilder {
	return &PacketBuilder{
		buffer: buffer,
		offset: 0,
	}
}

func (pb *PacketBuilder) WriteByte(b byte) error {
	if pb.offset >= len(pb.buffer) {
		return ErrPacketTooSmall
	}
	pb.buffer[pb.offset] = b
	pb.offset++
	return nil
}

// implemented this to start from LSB to MSB but the MSB for every
// byte is for continuation, 1 indicates there is another chunk
// the so called LEB128 format
func (pb *PacketBuilder) WriteVarint(val uint32) error {
	for val >= 128 { // as long as it is larger than 7 bits
		if pb.offset >= len(pb.buffer) {
			return ErrPacketTooSmall
		}
		// just takes the last 7 bits from LSB and the MSB set to 1 to continue
		pb.buffer[pb.offset] = byte(val&0x7F) | 0x80
		pb.offset++
		val >>= 7
	}
	if pb.offset >= len(pb.buffer) {
		return ErrPacketTooSmall
	}
	pb.buffer[pb.offset] = byte(val)
	pb.offset++
	return nil
}

// would return the metamode of huffman table encoding
// func (pb *PacketBuilder) WriteHuffman(h *huffman.Huffman, bitLen int) (byte, error) {
// 	mode, bytesWritten, err := huffman.IntoBytes(h, bitLen, pb.buffer, pb.offset)
// 	if err != nil {
// 		return 0, err
// 	}
// 	pb.offset += bytesWritten
// 	return mode, nil
// }

func (pb *PacketBuilder) Bytes() []byte {
	return pb.buffer[:pb.offset]
}
func (pb *PacketBuilder) CurrentSize() int {
	return pb.offset
}
