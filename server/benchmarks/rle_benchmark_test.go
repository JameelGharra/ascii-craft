package benchmarks

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/encoding"
	"github.com/JameelGharra/ascii-craft/server/ipc"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

const (
	TotalFrames = 1000
)

type CompressionStat struct {
	FrameIndex      int
	OriginalBytes   int
	CompressedBytes int
	Ratio           float64 // percentage saved
}

func TestRLECompressionWithRandomBot(t *testing.T) {
	absPath, err := filepath.Abs(BinaryPath)
	if err != nil {
		t.Fatalf("Path error: %v", err)
	}

	cmd := exec.Command(absPath)
	cmd.Dir = filepath.Dir(absPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start game: %v", err)
	}

	processDone := make(chan error, 1) // just to signal any process exit like game crash
	go func() {
		processDone <- cmd.Wait()
	}()

	// defer func() {
	// 	if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
	// 		fmt.Println("Stopping game process...")
	// 		cmd.Process.Kill()
	// 	}
	// }()

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
		t.Fatalf("Could not connect to IPC after retries")
	}
	defer client.Close()

	// right now I made it that it outputs to a csv file
	outFileName := "compression_rle_stats.csv"
	outFile, err := os.Create(outFileName)
	if err != nil {
		t.Fatalf("Failed to create statistics file: %v", err)
	}
	defer outFile.Close()

	fmt.Fprintln(outFile, "Frame,Original_Bytes,Compressed_Bytes,Savings_Percent") // csv header
	fmt.Printf("Writing frame statistics to %s...\n", outFileName)

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
			t.Fatalf("Game process exited prematurely: %v", err)
		case <-timeout:
			t.Fatalf("Timed out waiting for first frame")
		default:
			f, ok := client.TryReadFrame()
			if ok {
				width, height = int(f.Width), int(f.Height)
				break initLoop
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	frameEncoder := encoding.NewFrameEncoder(uint32(width), uint32(height))
	planarAsciiFrame := ascii.NewAsciiFrame(uint32(width), uint32(height))

	var stats []CompressionStat
	var totalOriginal int64
	var totalCompressed int64

	startBench := time.Now()
	lastFrameTime := time.Now()

	charsFreq := ascii.NewFrequency()
	colorsFreq := ascii.NewFrequency()
	for frameNum := 0; frameNum < TotalFrames; {

		select {
		case err := <-processDone:
			t.Fatalf("Game CRASHED at frame %d! Error: %v", frameNum, err)
		default:
		}

		if time.Since(lastFrameTime) > 2*time.Second {
			t.Fatalf("Game stopped producing frames at %d (hang/deadlock detected)", frameNum)
		}

		if rng.Float64() < 0.05 {
			cmdType := possibleCmds[rng.Intn(len(possibleCmds))]
			var val int32 = 0
			if cmdType == ipc.CmdSelectSlot {
				val = int32(rng.Intn(10))
			}
			client.WriteCommand(cmdType, val)
		}

		frame, isNew := client.TryReadFrame()
		if !isNew {
			time.Sleep(1 * time.Millisecond)
			continue
		}

		lastFrameTime = time.Now()

		frame.Planar(planarAsciiFrame)
		charsData := utils.New8BitIterator(planarAsciiFrame.Buffer[:len(planarAsciiFrame.Buffer)/2])
		colorsData := utils.New8BitIterator(planarAsciiFrame.Buffer[len(planarAsciiFrame.Buffer)/2:])
		charsFreq.Count(charsData)
		colorsFreq.Count(colorsData)
		result, err := frameEncoder.Encode(planarAsciiFrame)

		origSize := len(planarAsciiFrame.Buffer)
		compSize := 0

		if err != nil {
			t.Fatalf("Encoding Error at frame %d: %v", frameNum, err)
		} else {
			compSize = len(result)
		}

		ratio := 100.0 * (1.0 - (float64(compSize) / float64(origSize))) // saved percentage

		fmt.Fprintf(outFile, "%d,%d,%d,%.2f\n", frameNum, origSize, compSize, ratio)

		stat := CompressionStat{
			FrameIndex:      frameNum,
			OriginalBytes:   origSize,
			CompressedBytes: compSize,
			Ratio:           ratio,
		}
		stats = append(stats, stat)

		totalOriginal += int64(origSize)
		totalCompressed += int64(compSize)

		// if frameNum%1000 == 0 {
		// 	fmt.Printf("Frame %d/%d | Current Compression: %.2f%%\n", frameNum, TotalFrames, ratio)
		// }

		frameNum++
	}

	duration := time.Since(startBench)

	var maxRatio, minRatio float64 = -100.0, 100.0
	var avgRatio float64

	for _, s := range stats {
		if s.Ratio > maxRatio {
			maxRatio = s.Ratio
		}
		if s.Ratio < minRatio {
			minRatio = s.Ratio
		}
	}

	avgRatio = 100.0 * (1.0 - (float64(totalCompressed) / float64(totalOriginal)))

	// killed the process to remove noisy output from the game
	if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
		fmt.Println("Stopping game process...")
		cmd.Process.Kill()
	}
	fmt.Print("\033[0m\033[2J\033[H\033[3J")

	fmt.Println("\n==========================================")
	fmt.Println("       RLE COMPRESSION BENCHMARK          ")
	fmt.Println("==========================================")
	fmt.Printf("Total Frames:   %d\n", TotalFrames)
	fmt.Printf("Time Elapsed:   %v\n", duration)
	fmt.Printf("Average FPS:    %.2f\n", float64(TotalFrames)/duration.Seconds())
	fmt.Println("------------------------------------------")
	fmt.Printf("Total Data (Raw):  %.2f MB\n", float64(totalOriginal)/(1024*1024))
	fmt.Printf("Total Data (RLE):  %.2f MB\n", float64(totalCompressed)/(1024*1024))
	fmt.Printf("Raw Data Written:  %s\n", outFileName)
	fmt.Println("------------------------------------------")
	fmt.Printf("Best Compression:  %.2f%% (Less detail/Sky)\n", maxRatio)
	fmt.Printf("Worst Compression: %.2f%% (High noise/Trees)\n", minRatio)
	fmt.Printf("AVERAGE SAVINGS:   %.2f%%\n", avgRatio)
	fmt.Println("==========================================")

	if avgRatio < 0 {
		t.Errorf("RLE is performing worse than raw data on average!")
	}
	fmt.Printf("CHARS FREQ.: %s\n", charsFreq.Debug())
	fmt.Printf("COLORS FREQ.: %s\n", colorsFreq.Debug())
}
