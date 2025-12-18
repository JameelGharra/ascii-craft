package encoding

import (
	"errors"

	"github.com/JameelGharra/ascii-craft/server/ascii"
)

type AsciiRLE struct {
	buffer []byte
	pos    int
	err    bool
}

func NewAsciiRLE(maxSize uint32) *AsciiRLE {
	return &AsciiRLE{
		buffer: make([]byte, maxSize),
		pos:    0,
		err:    false,
	}
}

func (a *AsciiRLE) Size() int {
	return a.pos
}

var (
	ErrBufferTooSmall = errors.New("RLE buffer too small")
	// ErrWorse          = errors.New("RLE made it worse, not better")
)

func (a *AsciiRLE) RLE(in *ascii.AsciiFrame) error {
	length := len(in.Buffer)
	buffMaxSize := len(a.buffer)
	count := 1
	for index := 0; index < length; {
		count = 1
		current := in.Buffer[index]
		for index+count < length && in.Buffer[index+count] == current && count < 255 {
			count++
		}
		index += count
		// if a.pos+1 >= length {
		// 	a.err = true
		// 	return ErrWorse
		// }
		if a.pos+1 >= buffMaxSize {
			a.err = true
			return ErrBufferTooSmall
		}
		a.buffer[a.pos] = byte(count)
		a.buffer[a.pos+1] = current
		a.pos += 2
	}
	return nil
}

func (a *AsciiRLE) Result() []byte {
	if a.err {
		return nil
	}
	return a.buffer[:a.pos]
}

func (a *AsciiRLE) Reset() {
	a.pos = 0
	a.err = false
}
