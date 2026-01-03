package encoding

import (
	"errors"

	"github.com/JameelGharra/ascii-craft/server/utils"
)

const (
	expansionChunkSize = 128
)

var (
	ErrBufferTooSmall = errors.New("RLE buffer too small")
	ErrNotFinished    = errors.New("RLE encoding not finished")
)

type AsciiRLE struct {
	buffer     []byte
	pos        int
	currByte   byte
	count      int
	isFinished bool
}

func NewAsciiRLE() *AsciiRLE {
	return &AsciiRLE{
		buffer:     nil,
		pos:        0,
		currByte:   0,
		count:      0,
		isFinished: false,
	}
}

func (a *AsciiRLE) Size() int {
	return a.pos
}

func (a *AsciiRLE) update() {
	if a.count == 0 {
		return
	}
	if a.pos+1 >= len(a.buffer) {
		a.buffer = append(a.buffer, make([]byte, expansionChunkSize)...)
	}
	a.buffer[a.pos] = byte(a.count)
	a.buffer[a.pos+1] = a.currByte
	a.pos += 2
}

func (a *AsciiRLE) Write(data []byte) {
	utils.Assert(a.buffer != nil, "AsciiRLE has to have a buffer to write into first")
	utils.Assert(len(data) > 0, "data has to be non empty when RLE writing")
	if a.count == 0 {
		a.currByte = data[0]
	}
	for _, curr := range data {
		if curr == a.currByte && a.count < 255 {
			a.count++
			continue
		}
		a.update()
		a.currByte = curr
		a.count = 1
	}
}

func (a *AsciiRLE) Finish() {
	if a.isFinished {
		return
	}
	a.update()
	a.isFinished = true
}

func (a *AsciiRLE) Result() ([]byte, error) {
	if a.isFinished {
		return a.buffer[:a.pos], nil
	}
	return nil, ErrNotFinished
}

func (a *AsciiRLE) Reset(buffer []byte) {
	a.buffer = buffer
	a.pos = 0
	a.currByte = 0
	a.count = 0
	a.isFinished = false
}
