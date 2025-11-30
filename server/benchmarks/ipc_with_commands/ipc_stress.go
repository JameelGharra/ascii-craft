package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"syscall"
	"time"
	"unsafe"
)

const (
	// Adjust path relative to where you run this file
	BinaryPath    = "../../../game/craft.exe"
	ShmName       = "Local\\CraftSharedMemory"
	ShmSize       = 1024 * 1024 * 4
	CmdBufferSize = 256
)

// Matches C struct layout
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
	Value int32 // Changed to int32 to match C
}

const (
	CmdSelectSlot = 9
)

func main() {
	// 1. Start C Binary
	abs, err := filepath.Abs(BinaryPath)
	if err != nil {
		log.Fatalf("Path error: %v", err)
	}

	cmd := exec.Command(abs)
	cmd.Dir = filepath.Dir(abs) // Run in the game directory
	// Redirect Stderr so we see C errors if they happen
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	// Ensure we kill the C process when Go exits
	defer func() {
		fmt.Println("Stopping C process...")
		cmd.Process.Kill()
	}()

	fmt.Println("Starting C process...")
	time.Sleep(1 * time.Second) // Wait for C to init SHM

	// 2. Open Shared Memory
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procOpenFileMapping := kernel32.NewProc("OpenFileMappingA")
	procMapViewOfFile := kernel32.NewProc("MapViewOfFile")

	const FILE_MAP_ALL_ACCESS = 0xF001F // Need Write access to send commands!
	namePtr, _ := syscall.BytePtrFromString(ShmName)

	hMapFile, _, _ := procOpenFileMapping.Call(
		uintptr(FILE_MAP_ALL_ACCESS),
		0,
		uintptr(unsafe.Pointer(namePtr)),
	)
	if hMapFile == 0 {
		log.Fatal("Could not open shared memory. Is craft.exe running?")
	}

	addr, _, _ := procMapViewOfFile.Call(
		hMapFile,
		uintptr(FILE_MAP_ALL_ACCESS),
		0, 0, 0,
	)
	if addr == 0 {
		log.Fatal("MapViewOfFile failed")
	}

	// 3. Setup Pointers
	shm := (*SharedMemoryLayout)(unsafe.Pointer(addr))
	const HeaderSize = 24
	const CmdEntrySize = 8

	cmdArrayPtr := addr + HeaderSize
	dataPtr := addr + HeaderSize + (CmdBufferSize * CmdEntrySize)

	fmt.Println("--- STARTING IPC STRESS TEST ---")

	lastSeq := uint32(0)
	frameCount := 0
	startBench := time.Now()
	latencies := make([]float64, 0, 10000)

	// 4. Background Command Sender (The "Stress" part)
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond) // Fast switching
		defer ticker.Stop()
		itemSelection := 0

		for {
			select {
			case <-done:
				return // Stop cleanly
			case <-ticker.C:
				// Logic to change items
				itemSelection = (itemSelection + 1) % 9

				// Write to Ring Buffer
				tail := *(*uint32)(unsafe.Pointer(&shm.CmdTail))
				idx := tail % CmdBufferSize
				entryAddr := cmdArrayPtr + uintptr(idx*CmdEntrySize)

				entry := (*IPCCommandEntry)(unsafe.Pointer(entryAddr))
				entry.Type = CmdSelectSlot
				entry.Value = int32(itemSelection + 1) // Slots 1-9

				// Increment Tail (Signal C)
				*(*uint32)(unsafe.Pointer(&shm.CmdTail)) = tail + 1
			}
		}
	}()

	// 5. Main Reader Loop
	for {
		currSeq := *(*uint32)(unsafe.Pointer(&shm.FrameSeq))

		if currSeq > lastSeq && currSeq%2 == 0 {
			readStart := time.Now()

			dataLen := *(*uint32)(unsafe.Pointer(&shm.DataLen))

			// Zero-copy read
			var _ []byte
			sliceHeader := struct {
				Addr uintptr
				Len  int
				Cap  int
			}{dataPtr, int(dataLen), int(dataLen)}
			shmSlice := *(*[]byte)(unsafe.Pointer(&sliceHeader))
			if frameCount%60 == 0 {
				fmt.Print("\033[H\033[2J")
				// Convert bytes to string and print.
				// Since it contains ANSI codes, your terminal should render colors!
				fmt.Println(string(shmSlice))
			}
			lat := time.Since(readStart)
			latencies = append(latencies, float64(lat.Nanoseconds()))

			lastSeq = currSeq
			frameCount++

			if frameCount%60 == 0 {
				fmt.Print(".")
			}
		}

		// Run for exactly 10 seconds
		if time.Since(startBench) > 10*time.Second {
			break
		}

		// Busy wait for precision
		for i := 0; i < 100; i++ {
		}
	}

	// 6. Cleanup
	close(done) // Tell the background thread to stop
	fmt.Println("\n\n--- RESULTS ---")
	printIPCStats(frameCount, latencies)
}

func printIPCStats(count int, times []float64) {
	if count == 0 {
		fmt.Println("No frames detected.")
		return
	}

	sort.Float64s(times)

	var sum float64
	for _, t := range times {
		sum += t
	}

	// times holds Nanoseconds.
	// To get Microseconds, we divide by 1000.0

	avgNs := sum / float64(len(times))
	p99Ns := times[int(float64(len(times))*0.99)]
	fmt.Printf("Total Frames: %d\n", count)
	fmt.Printf("Read Latency Average: %.2f ns  (%.4f µs)\n", avgNs, avgNs/1000.0)
	fmt.Printf("Read Latency P99:     %.2f ns  (%.4f µs)\n", p99Ns, p99Ns/1000.0)
	fmt.Printf("Read Latency Max:     %.2f ns  (%.4f µs)\n", times[len(times)-1], times[len(times)-1]/1000.0)
	fmt.Println("-------------------------------------------------")
}
