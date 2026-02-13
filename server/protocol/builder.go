package protocol

import (
	"errors"

	"github.com/JameelGharra/ascii-craft/server/encoding/huffman"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

var (
	ErrPacketTooSmall   = errors.New("packet buffer too small for writing")
	ErrIndexOutOfBounds = errors.New("index out of bounds for setting byte in packet builder")
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

func (pb *PacketBuilder) WriteBytes(data []byte) error {
	if pb.offset+len(data) > len(pb.buffer) {
		return ErrPacketTooSmall
	}
	copy(pb.buffer[pb.offset:], data)
	pb.offset += len(data)
	return nil
}

// for backpatching specially for flags after constructing compression meta
func (pb *PacketBuilder) SetByte(index int, b byte) error {
	if index < 0 || index >= pb.offset {
		return ErrIndexOutOfBounds
	}
	pb.buffer[index] = b
	return nil
}

func (pb *PacketBuilder) WriteVarint(val uint32) error {
	writtenBytes, err := utils.PutVarint(pb.buffer[pb.offset:], val)
	if err != nil {
		return err
	}
	pb.offset += writtenBytes
	return nil
}

// would return the metamode of huffman table encoding for flags later on
func (pb *PacketBuilder) WriteHuffmanMeta(h *huffman.Huffman) (byte, error) {
	mode, bytesWritten, err := h.WriteTableBytes(pb.buffer[pb.offset:])
	if err != nil {
		return 0, err
	}
	pb.offset += bytesWritten
	return mode, nil
}

func (pb *PacketBuilder) Bytes() []byte {
	return pb.buffer[:pb.offset]
}
func (pb *PacketBuilder) CurrentSize() int {
	return pb.offset
}

func (pb *PacketBuilder) Reset() {
	pb.offset = 0
}
