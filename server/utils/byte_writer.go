package utils

func Write16(buffer []byte, indexStart int, value int) {
	buffer[indexStart] = byte((value >> 8) & 0xFF)
	buffer[indexStart+1] = byte(value & 0xFF)
}
