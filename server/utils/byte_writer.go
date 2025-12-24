package utils

import "errors"

func Write16(buffer []byte, indexStart int, value int) {
	Assert(len(buffer) > indexStart+1, "Index out of bounds in Write16")
	buffer[indexStart] = byte((value >> 8) & 0xFF)
	buffer[indexStart+1] = byte(value & 0xFF)
}

func Read16(buffer []byte, indexStart int) int {
	Assert(len(buffer) > indexStart+1, "Index out of bounds in Read16")
	return (int(buffer[indexStart]) << 8) | int(buffer[indexStart+1])
}

var ErrByteWriterExceedsBuffer = errors.New("byte writer exceeds buffer size")

type ByteWriter interface {
	Write(value int) error
	Len() int
}

type ByteWriter8 struct {
	buffer []byte
	index  int
}

func (bw *ByteWriter8) Set(buffer []byte) {
	bw.buffer = buffer
}
func (bw *ByteWriter8) Write(value int) error {
	if bw.index >= len(bw.buffer) {
		return ErrByteWriterExceedsBuffer
	}
	bw.buffer[bw.index] = byte(value & 0xFF)
	bw.index++
	return nil
}
func (bw *ByteWriter8) Len() int {
	return bw.index
}

type ByteWriter16 struct {
	buffer []byte
	index  int
}

func (bw *ByteWriter16) Set(buffer []byte) {
	Assert(len(buffer)&0x1 == 0, "a buffer passed to a 16-bit writer must have even length")
	bw.buffer = buffer
}

func (bw *ByteWriter16) Write(value int) error {
	if bw.index+1 >= len(bw.buffer) {
		return ErrByteWriterExceedsBuffer
	}
	Write16(bw.buffer, bw.index, value)
	bw.index += 2
	return nil
}

func (bw *ByteWriter16) Len() int {
	return bw.index
}
