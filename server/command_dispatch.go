package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/JameelGharra/ascii-craft/server/ipc"
)

type CommandDispatcher struct {
	client   *ipc.Client
	commands map[string]uint32
}

func NewCommandDispatcher(client *ipc.Client) *CommandDispatcher {
	return &CommandDispatcher{
		client: client,
		commands: map[string]uint32{
			"!w":            ipc.CmdForward,
			"!s":            ipc.CmdBackward,
			"!a":            ipc.CmdLeft,
			"!d":            ipc.CmdRight,
			"!jump":         ipc.CmdJump,
			"!fly":          ipc.CmdFly,
			"!build":        ipc.CmdBuild,
			"!destroy":      ipc.CmdDestroy,
			"!turnleft":     ipc.CmdTurnLeft,
			"!turnright":    ipc.CmdTurnRight,
			"!lookup":       ipc.CmdLookUp,
			"!lookdown":     ipc.CmdLookDown,
			"!jumpforward":  ipc.CmdJumpForward,
			"!jumpbackward": ipc.CmdJumpBackward,
			"!jumpleft":     ipc.CmdJumpLeft,
			"!jumpright":    ipc.CmdJumpRight,
		},
	}
}

func (d *CommandDispatcher) Dispatch(rawCmd string) {
	rawCmd = strings.TrimSpace(strings.ToLower(rawCmd))
	if rawCmd == "" {
		return
	}

	if strings.HasPrefix(rawCmd, "!slot ") {
		parts := strings.Split(rawCmd, " ")
		if len(parts) == 2 {
			val, err := strconv.Atoi(parts[1])
			if err == nil && val >= ipc.MinSupportedSlotIdx && val <= ipc.MaxSupportedSlotIdx {
				d.client.WriteCommand(ipc.CmdSelectSlot, int32(val))
			}
		}
		return
	}

	if cmdType, exists := d.commands[rawCmd]; exists {
		d.client.WriteCommand(cmdType, ipc.IgnoredDefaultValue)
		fmt.Printf("Executed remote command: %s\n", rawCmd)
	} else {
		fmt.Printf("Unknown command received: %s\n", rawCmd)
	}
}
