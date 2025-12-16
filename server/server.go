package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/ipc"
)

// Path to your compiled C binary
const BinaryPath = "../game/craft.exe"
const Port = ":9000"

func main() {
	absPath, _ := filepath.Abs(BinaryPath)
	cmd := exec.Command(absPath)
	cmd.Dir = filepath.Dir(absPath) // for texture etc..
	cmd.Stdout = nil                // not using it anyway
	cmd.Stderr = os.Stderr

	fmt.Println("Launching the C engine...")
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start game: %v", err)
	}
	defer func() {
		cmd.Process.Kill()
		fmt.Println("\nGame stopped.")
	}()

	// Retry loop because C takes a moment to alloc SHM
	var client *ipc.Client
	var err error
	for i := 0; i < 10; i++ {
		time.Sleep(2000 * time.Millisecond)
		client, err = ipc.NewClient()
		if err == nil {
			break
		}
	}
	if client == nil {
		log.Fatalf("Could not connect to IPC: %v", err)
	}
	defer client.Close()
	fmt.Println("Connected to shared mem.")

	go startCommandServer(client)

	fmt.Println("Starting render loop..")
	time.Sleep(1 * time.Second)

	// Clear screen once
	fmt.Print("\033[2J")

	var buffer []byte

	for {
		frame, isNew := client.TryReadFrame()
		if !isNew {
			// just to make sure that there is a frame done without busy wait
			time.Sleep(1 * time.Millisecond)
			continue
		}

		renderANSI(frame, &buffer)
	}
}

func startCommandServer(client *ipc.Client) {
	ln, err := net.Listen("tcp", Port)
	if err != nil {
		log.Fatalf("TCP listen error: %v", err)
	}
	// log.Printf("Command Server listening on %s", Port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn, client)
	}
}

func handleConnection(conn net.Conn, client *ipc.Client) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		processCommand(client, text)
	}
}

func processCommand(client *ipc.Client, text string) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]
	var op uint32 = ipc.CmdNone
	var val int32 = 0

	switch cmd {
	// movement
	case "!forward", "!w":
		op = ipc.CmdForward
	case "!back", "!s":
		op = ipc.CmdBackward
	case "!left", "!a":
		op = ipc.CmdLeft
	case "!right", "!d":
		op = ipc.CmdRight

	// camera
	case "!turnleft", "!l":
		op = ipc.CmdTurnLeft
	case "!turnright", "!r":
		op = ipc.CmdTurnRight
	case "!up", "!lookup":
		op = ipc.CmdLookUp
	case "!down", "!lookdown":
		op = ipc.CmdLookDown

	// actions
	case "!jump", "!j":
		op = ipc.CmdJump
	case "!fly":
		op = ipc.CmdFly
	case "!build", "!b":
		op = ipc.CmdBuild
	case "!destroy", "!x":
		op = ipc.CmdDestroy

	// slot selection
	case "!slot":
		if len(parts) > 1 {
			if v, err := strconv.Atoi(parts[1]); err == nil {
				op = ipc.CmdSelectSlot
				val = int32(v - 1) // 1-based to 0-based
			}
		}
	}

	if op != ipc.CmdNone {
		client.WriteCommand(op, val)
	}
}

func renderANSI(frame *ascii.Frame, buffer *[]byte) {
	// Reset buffer
	*buffer = (*buffer)[:0]

	// Move cursor to top-left (0,0)
	*buffer = append(*buffer, "\033[H"...)

	width := int(frame.Width)
	height := int(frame.Height)
	pixels := frame.Pixels

	var lastR, lastG, lastB uint8 = 255, 255, 255 // force initial set

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			p := pixels[idx]

			// same concept as last_color, only on change
			if p.R != lastR || p.G != lastG || p.B != lastB {
				// ANSI TrueColor: \033[38;2;R;G;Bm
				*buffer = append(*buffer, fmt.Sprintf("\033[38;2;%d;%d;%dm", p.R, p.G, p.B)...)
				lastR, lastG, lastB = p.R, p.G, p.B
			}
			*buffer = append(*buffer, p.CharCode)
		}
		*buffer = append(*buffer, '\n')
	}

	os.Stdout.Write(*buffer)
}
