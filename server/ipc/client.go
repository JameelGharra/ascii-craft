package ipc

import (
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/JameelGharra/ascii-craft/server/ascii"
)

type Client struct {
	shm     *SharedMemoryLayout
	cmdPtr  uintptr
	dataPtr uintptr

	handle syscall.Handle

	lastSeq  uint32
	pixelBuf []ascii.AsciiPixel
}

func NewClient() (*Client, error) {
	addr, handle, err := mapSharedMemory()
	if err != nil {
		return nil, err
	}
	shm := (*SharedMemoryLayout)(unsafe.Pointer(addr))
	const CmdSizeBytes = CmdBufferSize * CmdEntrySize

	cmdPtr := addr + HeaderSize
	dataPtr := addr + HeaderSize + CmdSizeBytes

	return &Client{
		shm:      shm,
		cmdPtr:   cmdPtr,
		dataPtr:  dataPtr,
		handle:   handle,
		pixelBuf: make([]ascii.AsciiPixel, 0, 4096),
	}, nil
}

func (c *Client) Close() {
	unmapSharedMemory(uintptr(unsafe.Pointer(c.shm)), c.handle)
}

func (c *Client) WriteCommand(cmdType uint32, value int32) error {
	tail := atomic.LoadUint32(&c.shm.CmdTail)
	idx := tail % CmdBufferSize

	entryAddr := c.cmdPtr + uintptr(idx*CmdEntrySize)
	entry := (*IPCCommandEntry)(unsafe.Pointer(entryAddr))
	entry.Type = cmdType
	entry.Value = value
	atomic.StoreUint32(&c.shm.CmdTail, tail+1)
	return nil
}

var Collisions int = 0

func (c *Client) TryReadFrame() (*ascii.Frame, bool) {
	currSeq := atomic.LoadUint32(&c.shm.FrameSeq)
	if currSeq%2 != 0 || currSeq == c.lastSeq {
		return nil, false
	}
	dataLen := atomic.LoadUint32(&c.shm.DataLen)
	width := atomic.LoadUint32(&c.shm.Width)
	height := atomic.LoadUint32(&c.shm.Height)

	if dataLen == 0 || width == 0 || height == 0 { // incase of garbage
		return nil, false
	}
	numPixels := int(dataLen) / PixelSize // data comes from C as bytes (includes sizeof(AsciiPixel))

	if cap(c.pixelBuf) < numPixels {
		newCap := numPixels + (numPixels / 2) // 1.5x capacity for now
		c.pixelBuf = make([]ascii.AsciiPixel, numPixels, newCap)
	}
	c.pixelBuf = c.pixelBuf[:numPixels]
	srcSliceHeader := struct {
		Addr uintptr
		Len  int
		Cap  int
	}{c.dataPtr, numPixels, numPixels}
	src := *(*[]ascii.AsciiPixel)(unsafe.Pointer(&srcSliceHeader))
	copy(c.pixelBuf, src)
	seqAfter := atomic.LoadUint32(&c.shm.FrameSeq)
	if seqAfter != currSeq { // torn frames
		Collisions++
		return nil, false
	}
	c.lastSeq = currSeq
	return &ascii.Frame{
		Width:  width,
		Height: height,
		Pixels: c.pixelBuf[:numPixels],
	}, true
}
