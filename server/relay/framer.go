package main

import (
	"errors"
	"io"
)

var ErrVarintTooLarge = errors.New("varint too large")

func ReadVarint(r io.ByteReader) (uint32, int, error) {
	var res uint32
	var s uint

	bytesRead := 0
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, bytesRead, err
		}
		bytesRead++
		if b < 128 {
			res |= uint32(b) << s
			return res, bytesRead, nil
		}
		res |= uint32(b&0x7F) << s
		s += 7
		if s >= 32 { // just some defense here it means bad varint
			return 0, bytesRead, ErrVarintTooLarge
		}
	}
}

// the byteReader for the prefix length and the reader to read the framed data
func ReadFullPacket(reader io.Reader, byteReader io.ByteReader) ([]byte, error) {
	length, _, err := ReadVarint(byteReader)
	if err != nil {
		return nil, err
	}
	data := make([]byte, length)
	_, err = io.ReadFull(reader, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
