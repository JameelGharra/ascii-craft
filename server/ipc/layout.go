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
	CmdNone         = 0
	CmdForward      = 1
	CmdBackward     = 2
	CmdLeft         = 3
	CmdRight        = 4
	CmdJump         = 5
	CmdFly          = 6
	CmdBuild        = 7
	CmdDestroy      = 8
	CmdSelectSlot   = 9
	CmdTurnLeft     = 10
	CmdTurnRight    = 11
	CmdLookUp       = 12
	CmdLookDown     = 13
	CmdJumpForward  = 14
	CmdJumpBackward = 15
	CmdJumpLeft     = 16
	CmdJumpRight    = 17

	MinSupportedSlotIdx = 0
	MaxSupportedSlotIdx = 9

	IgnoredDefaultValue = 0
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
