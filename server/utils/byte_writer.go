package utils

func Write16(buffer []byte, indexStart int, value int) {
	Assert(len(buffer) > indexStart+1, "Index out of bounds in Write16")
	buffer[indexStart] = byte((value >> 8) & 0xFF)
	buffer[indexStart+1] = byte(value & 0xFF)
}

func Read16(buffer []byte, indexStart int) int {
	Assert(len(buffer) > indexStart+1, "Index out of bounds in Read16")
	return (int(buffer[indexStart]) << 8) | int(buffer[indexStart+1])
}

type ByteWriter interface {
	Write(value int)
	Len() int
}

type ByteWriter8 struct {
	buffer []byte
	index  int
}

func (bw *ByteWriter8) Set(buffer []byte) {
	bw.buffer = buffer
}
func (bw *ByteWriter8) Write(value int) {
	bw.buffer[bw.index] = byte(value & 0xFF)
	bw.index++
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

func (bw *ByteWriter16) Write(value int) {
	Assert(bw.index+2 <= len(bw.buffer), "Index out of bounds in 16-bit writer write")
	Write16(bw.buffer, bw.index, value)
	bw.index += 2
}

func (bw *ByteWriter16) Len() int {
	return bw.index
}
