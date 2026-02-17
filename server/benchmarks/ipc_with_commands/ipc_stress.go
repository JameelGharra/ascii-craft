package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/ipc"
)

const BinaryPath = "../../../game/craft.exe"

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

	client, err := ipc.NewClient()
	if err != nil {
		log.Fatalf("Failed to create IPC client: %v", err)
	}
	defer client.Close()

	fmt.Println("--- STARTING IPC STRESS TEST ---")

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
				if err := client.WriteCommand(ipc.CmdSelectSlot, itemSelection); err != nil {
					log.Printf("Failed to write command: %v", err)
				}
			}
		}
	}()

	var bufferPtr *[]byte
	for {
		if time.Since(startBench) > 10*time.Second { // made it run for 10 seconds
			break
		}
		frame, isNew := client.TryReadFrame()
		if !isNew {
			continue
		}
		readStart := time.Now()
		lengths = append(lengths, int(len(frame.Pixels)*ipc.PixelSize)) // pixels * sizeof(AsciiPixel)

		// if frameCount%60 == 0 {
		PrintFrameInANSII(bufferPtr, frame, int(frame.Width), int(frame.Height))

		// }
		lat := time.Since(readStart)
		latencies = append(latencies, float64(lat.Nanoseconds()))
		frameCount++

		for i := 0; i < 100; i++ { // busy wait for precision

		}
	}

	close(done)
	fmt.Println("\n\n--- RESULTS ---")
	// if ipc.Collisions > 0 {
	// 	fmt.Printf("WARNING: Detected %d frame tears (Collisions avoided)\n", ipc.Collisions)
	// } else {
	// 	fmt.Println("No collisions detected (Synchronization perfect).")
	// }
	printIPCStats(frameCount, latencies, lengths)
}

func PrintFrameInANSII(bufferPtr *[]byte, frame *ascii.Frame, width, height int) {
	data := frame.Pixels
	if bufferPtr == nil {
		buff := make([]byte, 0, (19+1)*width*height+height+9)
		bufferPtr = &buff
	}
	*bufferPtr = (*bufferPtr)[:0]

	*bufferPtr = append(*bufferPtr, '\033', '[', 'H') // ANSI escape code to move cursor to top-left
	var lastColor uint32 = 0xFFFFFFFF
	for y := range height {
		for x := range width {
			idx := y*width + x
			currentColor := (uint32(data[idx].R) << 16) | (uint32(data[idx].G) << 8) | uint32(data[idx].B)
			if currentColor != lastColor {
				*bufferPtr = append(*bufferPtr, '\033', '[', '3', '8', ';', '2', ';')
				*bufferPtr = strconv.AppendInt(*bufferPtr, int64(data[idx].R), 10)
				*bufferPtr = append(*bufferPtr, ';')
				*bufferPtr = strconv.AppendInt(*bufferPtr, int64(data[idx].G), 10)
				*bufferPtr = append(*bufferPtr, ';')
				*bufferPtr = strconv.AppendInt(*bufferPtr, int64(data[idx].B), 10)
				*bufferPtr = append(*bufferPtr, 'm')
				lastColor = currentColor
			}
			*bufferPtr = append(*bufferPtr, data[idx].CharCode)
		}
		*bufferPtr = append(*bufferPtr, '\n')
	}
	*bufferPtr = append(*bufferPtr, '\033', '[', '0', 'm')
	os.Stdout.Write(*bufferPtr)

	// for comparison with 8 bit color
	// lastColor = 0xFFFFFFFF

	// for y := range height {
	// 	for x := range width {
	// 		idx := y*width + x
	// 		// 1. Quantize down (0-7, 0-7, 0-3)
	// 		rSmall, gSmall, bSmall := rgb.RGBToColor8BitANSII(data[idx].R, data[idx].G, data[idx].B)

	// 		// 2. Scale UP for display (0-255)
	// 		rDisp, gDisp, bDisp := rgb.Scale8BitToTrueColor(rSmall, gSmall, bSmall)

	// 		currentColor := (uint32(rDisp) << 16) | (uint32(gDisp) << 8) | uint32(bDisp)
	// 		if currentColor != lastColor {
	// 			*bufferPtr = append(*bufferPtr, '\033', '[', '3', '8', ';', '2', ';')
	// 			*bufferPtr = strconv.AppendInt(*bufferPtr, int64(rDisp), 10)
	// 			*bufferPtr = append(*bufferPtr, ';')
	// 			*bufferPtr = strconv.AppendInt(*bufferPtr, int64(gDisp), 10)
	// 			*bufferPtr = append(*bufferPtr, ';')
	// 			*bufferPtr = strconv.AppendInt(*bufferPtr, int64(bDisp), 10)
	// 			*bufferPtr = append(*bufferPtr, 'm')
	// 			lastColor = currentColor
	// 		}
	// 		*bufferPtr = append(*bufferPtr, data[idx].CharCode)
	// 	}
	// 	*bufferPtr = append(*bufferPtr, '\n')
	// }
	// *bufferPtr = append(*bufferPtr, '\033', '[', '0', 'm')
	// os.Stdout.Write(*bufferPtr)
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
