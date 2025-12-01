package benchmarks

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// FrameStats holds metrics for a single frame
type FrameStats struct {
	FrameID       int
	SizeBytes     int
	TransportTime time.Duration // Time spent inside the pipe/buffers
}

func BenchmarkSideCar(B *testing.B) {
	// 1. Setup Command
	abs, err := filepath.Abs(BinaryPath)
	if err != nil {
		log.Fatal(err)
	}
	cmd := exec.Command(abs)
	cmd.Dir = "../../game/"
	cmd.Stderr = os.Stderr // Pass errors through

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	// 2. Start
	fmt.Println("Starting Benchmark for Sidecar/Pipe Implementation...")
	fmt.Println("Collecting data for 10 seconds...")
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	// 3. Data Collection
	reader := bufio.NewReader(stdoutPipe)
	stats := make([]FrameStats, 0)

	benchmarkStart := time.Now()
	frameCount := 0

	var currentFrameStart time.Time
	var currentFrameBytes int
	isInFrame := false

	// Loop reading lines
	for {
		// ReadLine is slightly more low-level/efficient than ReadString
		lineBytes, _, err := reader.ReadLine()
		if err != nil {
			break
		}

		// If isPrefix is true, the line is too long for buffer, but for ASCII
		// craft lines are usually short enough. We count bytes anyway.
		currentFrameBytes += len(lineBytes) + 1 // +1 for newline we lost

		line := string(lineBytes)

		if strings.HasPrefix(line, ":::FRAME_START:::") {
			// Start of a new frame transmission
			currentFrameStart = time.Now()
			currentFrameBytes = 0
			isInFrame = true
		} else if strings.HasPrefix(line, ":::FRAME_END:::") {
			// End of frame transmission
			if isInFrame {
				transportDuration := time.Since(currentFrameStart)

				stats = append(stats, FrameStats{
					FrameID:       frameCount,
					SizeBytes:     currentFrameBytes,
					TransportTime: transportDuration,
				})

				frameCount++
				isInFrame = false

				// Optional: Print progress dots
				if frameCount%60 == 0 {
					fmt.Print(".")
				}
			}
		}

		// Stop after 10 seconds or 1000 frames
		if time.Since(benchmarkStart) > 10*time.Second || frameCount >= 1000 {
			break
		}
	}

	cmd.Process.Kill()
	fmt.Println("\n\n--- BENCHMARK RESULTS (Pipe/Stdout) ---")
	printStats(stats)
}

func printStats(stats []FrameStats) {
	if len(stats) == 0 {
		fmt.Println("No data collected.")
		return
	}

	var totalBytes int
	var totalTransport time.Duration
	var transportTimes []float64 // Microseconds

	for _, s := range stats {
		totalBytes += s.SizeBytes
		totalTransport += s.TransportTime
		transportTimes = append(transportTimes, float64(s.TransportTime.Microseconds()))
	}

	sort.Float64s(transportTimes)

	avgTransport := totalTransport / time.Duration(len(stats))
	p50 := transportTimes[int(float64(len(stats))*0.50)]
	p95 := transportTimes[int(float64(len(stats))*0.95)]
	p99 := transportTimes[int(float64(len(stats))*0.99)]

	fmt.Printf("Total Frames:      %d\n", len(stats))
	fmt.Printf("Avg Frame Size:    %.2f KB\n", float64(totalBytes/len(stats))/1024.0)
	fmt.Printf("Total Throughput:  %.2f MB/s\n", float64(totalBytes)/(1024*1024*10.0)) // Approx over 10s
	fmt.Println("-" + strings.Repeat("-", 30))
	fmt.Printf("Transport Latency (Time spent in Pipe/Go Reader):\n")
	fmt.Printf("  Average:   %v\n", avgTransport)
	fmt.Printf("  P50 (Med): %.2f µs\n", p50)
	fmt.Printf("  P95:       %.2f µs\n", p95)
	fmt.Printf("  P99:       %.2f µs (Worst 1%%)\n", p99)
	fmt.Println("-" + strings.Repeat("-", 30))
	fmt.Println("Analysis:")
	fmt.Println("If P99 is high (> 16ms), the game stutters visually due to pipe blocking.")
	fmt.Println("This measures overhead purely from moving text from C to Go.")
}
