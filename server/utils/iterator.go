package utils

type IByteIterator interface {
	HasNext() bool
	Next() int
}

type EightBitIterator struct {
	buffer  []byte
	index   int
	current int
}

func New8BitIterator(buffer []byte) *EightBitIterator {
	return &EightBitIterator{
		buffer:  buffer,
		index:   0,
		current: 0,
	}
}

func (it *EightBitIterator) HasNext() bool {
	return it.index < len(it.buffer)
}

func (it *EightBitIterator) Next() int {
	Assert(it.HasNext(), "called Next() when no more data available")
	it.current = int(it.buffer[it.index])
	it.index++
	return it.current
}

type SixteenBitIterator struct {
	buffer  []byte
	index   int
	current int
}

func New16BitIterator(buffer []byte) *SixteenBitIterator {
	Assert(len(buffer)%2 == 0, "buffer length must be even for 16-bit iterator")
	return &SixteenBitIterator{
		buffer:  buffer,
		index:   0,
		current: 0,
	}
}

func (it *SixteenBitIterator) HasNext() bool {
	return it.index+1 < len(it.buffer)
}

func (it *SixteenBitIterator) Next() int {
	Assert(it.HasNext(), "called Next() when no more data available")
	it.current = Read16(it.buffer, it.index)
	it.index += 2
	return it.current
}
