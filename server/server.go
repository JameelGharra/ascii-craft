package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/JameelGharra/ascii-craft/server/encoder"
	"github.com/JameelGharra/ascii-craft/server/protocol"
	"github.com/JameelGharra/ascii-craft/server/utils"

	"github.com/JameelGharra/ascii-craft/server/ipc"
)

// producing the protocol constants before running
//go:generate go run tools/gen_protocol/main.go

const (
	TotalFrames    = 10000
	BinaryPath     = "../game/craft.exe"
	Stride         = 1
	RefreshRate    = 120 // after how much frames to send key frame (i-frame)
	BotMode        = 0   // rng based cmds
	ControlledMode = 1
)

var commandMap = map[string]uint32{
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
}

func main() {
	absPath, err := filepath.Abs(BinaryPath)
	if err != nil {
		fmt.Printf("Failed to get absolute path of game binary: %v\n", err)
		return
	}

	cmd := exec.Command(absPath)
	cmd.Dir = filepath.Dir(absPath)
	// cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start game: %v\n", err)
		return
	}

	processDone := make(chan error, 1) // just to signal any process exit like game crash
	go func() {
		processDone <- cmd.Wait()
	}()

	fmt.Println("Game launched. Now connecting to IPC...")

	var client *ipc.Client
	for range 20 {
		time.Sleep(100 * time.Millisecond) // wait before retrying (busy wait)
		client, err = ipc.NewClient()
		if err == nil {
			break
		}
	}
	if client == nil {
		fmt.Println("Could not connect to IPC after retries")
		return
	}
	defer client.Close()

	fmt.Printf("Starting benchmark: %d Frames with random commands...\n", TotalFrames)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	possibleCmds := []uint32{
		ipc.CmdForward, ipc.CmdBackward,
		ipc.CmdTurnLeft, ipc.CmdTurnRight,
		ipc.CmdLookUp, ipc.CmdLookDown,
		ipc.CmdJump, ipc.CmdJumpForward,
		ipc.CmdSelectSlot,
	}

	var width, height int
	timeout := time.After(2 * time.Second)

initLoop:
	for {
		select {
		case err := <-processDone:
			fmt.Printf("Game process exited prematurely: %v\n", err)
			return
		case <-timeout:
			fmt.Println("Timed out waiting for first frame")
			return
		default:
			f, ok := client.TryReadFrame()
			if ok {
				width, height = int(f.Width), int(f.Height)
				break initLoop
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	var packetInternalBuffer []byte = make([]byte, 1024*1024)

	frameEncoder := encoder.NewEncoder(width*height, Stride)
	frameEncoder.AddEncoding(encoder.Raw)
	frameEncoder.AddEncoding(encoder.XorRLE)
	frameEncoder.AddEncoding(encoder.Huffman)

	coloredFrame := make([]byte, width*height)
	runMode := ControlledMode

	for {
		// 1. Dial the Relay
		var conn net.Conn
		var err error
		for {
			conn, err = net.Dial("tcp", "localhost:9000")
			if err == nil {
				fmt.Println("Connected to Relay!")
				break
			}
			// Retry every 1s if Relay is down
			time.Sleep(1 * time.Second)
		}

		if runMode == ControlledMode {
			go func(c net.Conn, gc *ipc.Client) {
				scanner := bufio.NewScanner(c)
				for scanner.Scan() {
					cmdStr := scanner.Text()
					handleRemoteCommand(cmdStr, gc)
				}
				if err := scanner.Err(); err != nil {
					fmt.Printf("Relay connection error: %v\n", err)
				}
			}(conn, client)
		}

		// 2. The Frame Loop (Existing logic)
		// We moved frameNum loop inside here
		// Note: You might want to preserve frameNum outside if you want continuity
		responseVal := loopingForFrames(FrameLooperConfig{
			numOfFrames:          TotalFrames,
			gameClient:           client,
			encoderToUse:         frameEncoder,
			frameBuffer:          coloredFrame,
			gameProcessDone:      processDone,
			rng:                  rng,
			commands:             possibleCmds,
			subprocessCmd:        cmd,
			packetInternalBuffer: packetInternalBuffer,
			conn:                 conn,
			refreshAfterFrames:   RefreshRate,
			runMode:              runMode,
		})

		// 3. Cleanup before retrying
		conn.Close()
		if responseVal {
			fmt.Println("Finished all game frames. Exiting.")
			return
		}
		fmt.Println("Relay connection lost. Retrying in 2s...")
		time.Sleep(2 * time.Second)
	}
}

type FrameLooperConfig struct {
	numOfFrames          int
	gameClient           *ipc.Client
	encoderToUse         *encoder.Encoder
	frameBuffer          []byte     // just to keep reusing even on reconnect
	gameProcessDone      chan error // just passing to get game feedback
	rng                  *rand.Rand
	commands             []uint32
	subprocessCmd        *exec.Cmd
	packetInternalBuffer []byte
	conn                 net.Conn
	refreshAfterFrames   int
	runMode              int
}

// if we looped total frames successfully returns true, otherwise false
// however, a quick note if the game itself crashed or for some reason stopped
// producing i did not want for a reconnection to happen, it would just end the program
// to get me noticing that since it should not happen technically and its C game related
func loopingForFrames(config FrameLooperConfig) bool {
	lastFrameTime := time.Now()
	packetBuilder := protocol.NewPacketBuilder(config.packetInternalBuffer)
	var headerbuff [5]byte
	var isKeyFrame bool = true
	for frameNum := 0; frameNum < TotalFrames; {
		select {
		case err := <-config.gameProcessDone:
			fmt.Printf("Game CRASHED at frame %d! Error: %v\n", frameNum, err)
			return true
		default:
		}

		if time.Since(lastFrameTime) > 2*time.Second {

			fmt.Printf("Game stopped producing frames at %d (hang/deadlock detected)\n", frameNum)
			return true
		}
		if config.runMode == BotMode && config.rng.Float64() < 0.05 {
			cmdType := config.commands[config.rng.Intn(len(config.commands))]
			var val int32 = 0
			if cmdType == ipc.CmdSelectSlot {
				val = int32(config.rng.Intn(10))
			}
			config.gameClient.WriteCommand(cmdType, val)
		}

		frame, isNew := config.gameClient.TryReadFrame()
		if !isNew {
			time.Sleep(1 * time.Millisecond)
			continue
		}

		lastFrameTime = time.Now()

		frame.ToColor8bit(config.frameBuffer)

		isKeyFrame = frameNum%config.refreshAfterFrames == 0
		result := config.encoderToUse.PushFrame(config.frameBuffer, isKeyFrame)
		fmt.Printf("Frame %d: Original=%d bytes, Compressed (Best)=%d bytes - Type: (%v)\n", frameNum, len(config.frameBuffer), result.FinalSize, result.Encoding)
		packetBuilder.Reset()
		err := config.encoderToUse.WriteTo(packetBuilder)
		if err != nil {
			fmt.Printf("Encoding error at frame %d: %v\n", frameNum, err)
			return true
		}
		// just some prefix length framing right here
		payload := packetBuilder.Bytes()
		payloadLen := uint32(len(payload))
		n, _ := utils.PutVarint(headerbuff[:], payloadLen)
		buffers := net.Buffers{headerbuff[:n], payload}
		if _, err := buffers.WriteTo(config.conn); err != nil {
			fmt.Printf("Failed to write frame %d to connection: %v\n", frameNum, err)
			return false
		}
		frameNum++
	}

	// killed the process to remove noisy output from the game
	if config.subprocessCmd.ProcessState == nil || !config.subprocessCmd.ProcessState.Exited() {
		fmt.Println("Stopping game process...")
		config.subprocessCmd.Process.Kill()
	}
	return true
}

func handleRemoteCommand(rawCmd string, gameClient *ipc.Client) {
	rawCmd = strings.TrimSpace((strings.ToLower(rawCmd)))
	if rawCmd == "" {
		return
	}
	if strings.HasPrefix(rawCmd, "!slot ") {
		parts := strings.Split(rawCmd, " ")
		if len(parts) == 2 {
			val, err := strconv.Atoi(parts[1])
			if err == nil && val >= ipc.MinSupportedSlotIdx && val <= ipc.MaxSupportedSlotIdx {
				gameClient.WriteCommand(ipc.CmdSelectSlot, int32(val))
			}
		}
		return
	}
	if cmdType, exists := commandMap[rawCmd]; exists {
		gameClient.WriteCommand(cmdType, ipc.IgnoredDefaultValue)
		fmt.Printf("Execute remote command: %s\n", rawCmd)
	} else {
		fmt.Printf("Unknown command received: %s\n", rawCmd)
	}
}
