package ipc

const (
	ShmName       = "Local\\CraftSharedMemory"
	ShmSize       = 1024 * 1024 * 4
	HeaderSize    = 24
	CmdBufferSize = 256
	CmdEntrySize  = 8
	PixelSize     = 4
)

const (
	CmdNone       = 0
	CmdForward    = 1
	CmdBackward   = 2
	CmdLeft       = 3
	CmdRight      = 4
	CmdJump       = 5
	CmdFly        = 6
	CmdBuild      = 7
	CmdDestroy    = 8
	CmdSelectSlot = 9
)

type SharedMemoryLayout struct {
	FrameSeq uint32
	Width    uint32
	Height   uint32
	DataLen  uint32
	CmdHead  uint32
	CmdTail  uint32
}

type IPCCommandEntry struct {
	Type  uint32
	Value int32
}
