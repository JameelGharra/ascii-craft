package benchmarks

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"sort"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

const (
	ShmName = "Local\\CraftSharedMemory"
	// Must match C header!
	ShmSize       = 1024 * 1024 * 4
	CmdBufferSize = 256
)

// Matches C struct layout exactly
type SharedMemoryLayout struct {
	FrameSeq uint32
	Width    uint32
	Height   uint32
	DataLen  uint32
	CmdHead  uint32
	CmdTail  uint32
	// Data follows immediately after
}

type IPCCommandEntry struct {
	Type  uint32
	Value uint32
}

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

func BenchmarkIPC(b *testing.B) {
	// 1. Start C Binary
	abs, _ := filepath.Abs(BinaryPath)
	cmd := exec.Command(abs)
	cmd.Dir = "../../game/"
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	defer cmd.Process.Kill()

	fmt.Println("Starting C process...")
	time.Sleep(1 * time.Second) // Wait for C to init SHM

	// 2. Open Shared Memory
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procOpenFileMapping := kernel32.NewProc("OpenFileMappingA")
	procMapViewOfFile := kernel32.NewProc("MapViewOfFile")

	const FILE_MAP_READ = 0x0004
	namePtr, _ := syscall.BytePtrFromString(ShmName)

	hMapFile, _, err := procOpenFileMapping.Call(
		uintptr(FILE_MAP_READ),
		0,
		uintptr(unsafe.Pointer(namePtr)),
	)
	if hMapFile == 0 {
		log.Fatalf("Could not open shared memory: %v", err)
	}

	addr, _, err := procMapViewOfFile.Call(
		hMapFile,
		uintptr(FILE_MAP_READ),
		0, 0, 0,
	)
	if addr == 0 {
		log.Fatal("MapViewOfFile failed")
	}

	shm := (*SharedMemoryLayout)(unsafe.Pointer(addr))
	const HeaderSize = 24
	const CmdEntrySize = 8
	CmdArrayPtr := addr + HeaderSize
	dataPtr := addr + HeaderSize + (CmdBufferSize * CmdEntrySize)

	fmt.Println("--- STARTING IPC BENCHMARK ---")

	lastSeq := uint32(0)
	frameCount := 0
	startBench := time.Now()

	var latencies []float64
	var itemSelection int = 0
	go func() {
		ticker := time.NewTicker(2 * time.Second)

		for range ticker.C {
			itemSelection = (itemSelection + 1) % 10
			fmt.Printf("Sending a selection item command with %d\n", itemSelection)
			tail := *(*uint32)(unsafe.Pointer(&shm.CmdTail))
			idx := tail % CmdBufferSize
			entryAddr := CmdArrayPtr + uintptr(idx*CmdEntrySize)

			entry := (*IPCCommandEntry)(unsafe.Pointer(entryAddr))
			entry.Type = CmdSelectSlot
			entry.Value = uint32(itemSelection)
		}
		*(*uint32)(unsafe.Pointer(&shm.CmdTail)) += 1
	}()
	for {
		currSeq := *(*uint32)(unsafe.Pointer(&shm.FrameSeq))

		if currSeq > lastSeq && currSeq%2 == 0 {

			// --- MEASUREMENT START ---
			readStart := time.Now()

			// 1. Read Metadata
			dataLen := *(*uint32)(unsafe.Pointer(&shm.DataLen))

			// 2. Construct Go Slice (Zero Copy)
			var _ []byte
			sliceHeader := struct {
				Addr uintptr
				Len  int
				Cap  int
			}{dataPtr, int(dataLen), int(dataLen)}
			shmSlice := *(*[]byte)(unsafe.Pointer(&sliceHeader))

			// Printing every 60 divisible frame for visual verification
			if frameCount%60 == 0 {
				// Clear terminal (optional, helps visibility)
				fmt.Print("\033[H\033[2J")

				fmt.Println("--- VERIFYING FRAME DATA (Frame #60) ---")
				// Convert bytes to string and print.
				// Since it contains ANSI codes, your terminal should render colors!
				fmt.Println(string(shmSlice))
				fmt.Println("\n--- END OF FRAME ---")

				// Optional: Exit after verification so you can look at it
				// return
			}
			// --- MEASUREMENT END ---
			lat := time.Since(readStart)

			// CRITICAL FIX: Use Nanoseconds (float64) to prevent truncation to 0
			latencies = append(latencies, float64(lat.Nanoseconds()))

			lastSeq = currSeq
			frameCount++

			if frameCount%60 == 0 {
				fmt.Print(".")
			}
		}

		if time.Since(startBench) > 10*time.Second {
			break
		}

		// Busy wait for high precision benchmark
		// (removes sleep jitter from the equation)
		for i := 0; i < 100; i++ {
		}
	}

	fmt.Println("\n\n--- BENCHMARK RESULTS (IPC/SharedMem) ---")
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

	// Previous Pipe result for comparison
	pipeP99Us := 2888.0
	ipcP99Us := p99Ns / 1000.0

	// Prevent division by zero in print
	if ipcP99Us < 0.0001 {
		ipcP99Us = 0.0001
	}

	fmt.Println("Comparison (P99):")
	fmt.Printf("Pipe: ~%.2f µs\n", pipeP99Us)
	fmt.Printf("IPC:  ~%.4f µs\n", ipcP99Us)
	fmt.Printf("Speedup: %.0fx Faster\n", pipeP99Us/ipcP99Us)
}
