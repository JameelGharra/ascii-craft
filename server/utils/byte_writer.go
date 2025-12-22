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
