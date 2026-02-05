package encoding

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/JameelGharra/ascii-craft/server/ipc"
)

func TestSimpleAsciiRLE(t *testing.T) {
	input := []byte{
		5, 5, 5,
		1, 1, 1,
		2, 2,
	}
	expected := []byte{
		3, 5,
		3, 1,
		2, 2,
	}
	rle := NewAsciiRLE()
	out := make([]byte, 0, 7)
	rle.Reset(out)
	rle.Write(input)
	rle.Finish()
	result, err := rle.Result()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result) != len(expected) {
		t.Fatalf("Expected result length %d, got %d", len(expected), len(result))
	}
	if !bytes.Equal(result, expected) {
		t.Fatalf("Expected: %v\nGot: %v", expected, result)
	}
}

// elements are unique hence it should panic x2
func TestSimpleAsciiRLEWorse(t *testing.T) {
	input := []byte{
		1, 2, 3, 4, 5, 6,
	}
	rle := NewAsciiRLE()
	rle.Reset(make([]byte, 6))
	rle.Write(input)
	rle.Finish()
	result, err := rle.Result()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) < len(input) {
		t.Fatalf("Expected RLE to be worse, but got a better result")
	}
}

// I made this test to just grab one frame and test whether RLE makes it better or not
func TestSimpleAsciiRealFrame(t *testing.T) {
	const BinaryPath = "../../game/craft.exe"

	abs, err := filepath.Abs(BinaryPath)
	if err != nil {
		log.Fatalf("Path error: %v", err)
	}

	cmd := exec.Command(abs)
	cmd.Dir = filepath.Dir(abs) // did that for assets like textures
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	defer func() {
		fmt.Println("Stopping C process...")
		cmd.Process.Kill()
	}()

	fmt.Println("Starting C process...")
	time.Sleep(2 * time.Second) // to wait for C to create the shared memory
	var client *ipc.Client
	for range 10 { // retry for additional total 5 secs
		time.Sleep(500 * time.Millisecond)
		client, err = ipc.NewClient()
		if err == nil {
			break
		}
	}
	if client == nil {
		t.Fatal("Could not connect to IPC after 5 seconds")
	}
	defer client.Close()
	frame, ok := client.TryReadFrame()
	if !ok {
		t.Fatalf("Failed to read frame from IPC")
	}
	colorFrame := make([]byte, frame.Width*frame.Height)
	rleBufferResult := make([]byte, frame.Width*frame.Height)
	rle := NewAsciiRLE()
	rle.Reset(rleBufferResult)
	frame.ToColor8bit(colorFrame)
	rle.Write(colorFrame)
	rle.Finish()
	result, err := rle.Result()
	if err != nil {
		t.Fatal("RLE failed. Result is nil")

	}
	if len(result) >= len(colorFrame) {
		t.Fatalf("RLE did not improve size. Original: %d, RLE: %d", len(colorFrame), len(result))
	}
	fmt.Printf("Original size: %d, RLE size: %d\n", len(colorFrame), len(result))
}
