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
	BinaryPath    = "../../../game/craft.exe"
	ShmName       = "Local\\CraftSharedMemory"
	ShmSize       = 1024 * 1024 * 4
	CmdBufferSize = 256
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

const (
	CmdSelectSlot = 9
)

func main() {
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

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procOpenFileMapping := kernel32.NewProc("OpenFileMappingA")
	procMapViewOfFile := kernel32.NewProc("MapViewOfFile")

	const FILE_MAP_ALL_ACCESS = 0xF001F
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
	lengths := make([]int, 0, 10000)

	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		var itemSelection int32 = 0

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				itemSelection = (itemSelection + 1) % 10

				tail := *(*uint32)(unsafe.Pointer(&shm.CmdTail))
				idx := tail % CmdBufferSize
				entryAddr := cmdArrayPtr + uintptr(idx*CmdEntrySize)

				entry := (*IPCCommandEntry)(unsafe.Pointer(entryAddr))
				entry.Type = CmdSelectSlot
				entry.Value = int32(itemSelection) // did slots 1-9 just like for keys slotindex choosing

				*(*uint32)(unsafe.Pointer(&shm.CmdTail)) = tail + 1
			}
		}
	}()

	collisions := 0 // count of frame tears detected
	for {
		currSeq := *(*uint32)(unsafe.Pointer(&shm.FrameSeq))

		if currSeq > lastSeq && currSeq%2 == 0 {
			readStart := time.Now()

			dataLen := *(*uint32)(unsafe.Pointer(&shm.DataLen))
			var _ []byte
			sliceHeader := struct {
				Addr uintptr
				Len  int
				Cap  int
			}{dataPtr, int(dataLen), int(dataLen)}
			_ = *(*[]byte)(unsafe.Pointer(&sliceHeader))
			lengths = append(lengths, int(dataLen))

			seqAfter := *(*uint32)(unsafe.Pointer(&shm.FrameSeq))
			if seqAfter != currSeq {
				collisions++
				continue // I am not counting the torn frames at the moment (we are not accepting things like half frame old & half a new)
			}

			// if frameCount%60 == 0 {
			// fmt.Print("\033[H\033[2J")
			// fmt.Print(string(shmSlice))
			// }
			lat := time.Since(readStart)
			latencies = append(latencies, float64(lat.Nanoseconds()))

			lastSeq = currSeq
			frameCount++

			if frameCount%60 == 0 {
				fmt.Print(".")
			}
		}

		if time.Since(startBench) > 10*time.Second { // made it run for 10 seconds
			break
		}

		for i := 0; i < 100; i++ { // busy wait for precision

		}
	}

	close(done)
	fmt.Println("\n\n--- RESULTS ---")
	if collisions > 0 {
		fmt.Printf("WARNING: Detected %d frame tears (Collisions avoided)\n", collisions)
	} else {
		fmt.Println("No collisions detected (Synchronization perfect).")
	}
	printIPCStats(frameCount, latencies, lengths)
}

func printIPCStats(count int, times []float64, lengths []int) {
	if count == 0 {
		fmt.Println("No frames detected.")
		return
	}

	fmt.Printf("Average bytes sent: %d bytes per time\n", func() int {
		var sum int
		for _, l := range lengths {
			sum += l
		}
		return sum / len(lengths)
	}())

	sort.Float64s(times)

	var sum float64
	for _, t := range times {
		sum += t
	}

	avgNs := sum / float64(len(times))
	p99Ns := times[int(float64(len(times))*0.99)]
	fmt.Printf("Total Frames: %d\n", count)
	fmt.Printf("Read Latency Average: %.2f ns  (%.4f µs)\n", avgNs, avgNs/1000.0)
	fmt.Printf("Read Latency P99:     %.2f ns  (%.4f µs)\n", p99Ns, p99Ns/1000.0)
	fmt.Printf("Read Latency Max:     %.2f ns  (%.4f µs)\n", times[len(times)-1], times[len(times)-1]/1000.0)
	fmt.Println("-------------------------------------------------")
}
