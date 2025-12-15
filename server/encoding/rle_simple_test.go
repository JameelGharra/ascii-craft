package encoding

import (
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
		2, 2}
	expected := []byte{
		3, 5,
		3, 1,
		2, 2,
	}
	rle := NewAsciiRLE(8)
	err := rle.RLE(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	result := rle.Result()
	fmt.Println(result)
	if len(result) != len(expected) {
		t.Fatalf("Expected result length %d, got %d", len(expected), len(result))
	}
}

// It has to throw ErrWorse since the elements are unique
func TestSimpleAsciiRLEWorse(t *testing.T) {
	input := []byte{
		1, 2, 3, 4, 5, 6,
	}
	rle := NewAsciiRLE(6)
	err := rle.RLE(input)
	if err != ErrWorse {
		t.Fatalf("Expected ErrWorse, got %v", err)
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
	time.Sleep(1 * time.Second) // to wait for C to create the shared memory

	client, err := ipc.NewClient()
	if err != nil {
		t.Fatalf("Failed to create IPC client: %v", err)
	}
	defer client.Close()
	frame, ok := client.TryReadFrame()
	if !ok {
		t.Fatalf("Failed to read frame from IPC")
	}
	pixelCount := len(frame.Pixels)
	input := make([]byte, pixelCount*2)
	SeperateCharsColor(frame.Pixels, input)
	rle := NewAsciiRLE(len(input))
	err = rle.RLE(input)
	if err != nil {
		t.Fatalf("Unexpected error during RLE: %v", err)
	}
	result := rle.Result()
	if result == nil {
		t.Fatal("RLE failed. Result is nil")

	}
	fmt.Printf("Original size: %d, RLE size: %d\n", len(input), len(result))
}
