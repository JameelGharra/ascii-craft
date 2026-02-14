package utils

import "errors"

var ErrBufferTooSmall = errors.New("buffer too small")

// implemented this to start from LSB to MSB but the MSB for every
// byte is for continuation, 1 indicates there is another chunk
// the so called LEB128 format
func PutVarint(buf []byte, x uint32) (int, error) {
	i := 0
	for x >= 0x80 { // as long as it is larger than 7 bits
		if i >= len(buf) {
			return 0, ErrBufferTooSmall
		}
		// just takes the last 7 bits from LSB and the MSB set to 1 to continue
		buf[i] = byte(x) | 0x80
		x >>= 7
		i++
	}
	if i >= len(buf) {
		return 0, ErrBufferTooSmall
	}
	buf[i] = byte(x)
	return i + 1, nil
}
