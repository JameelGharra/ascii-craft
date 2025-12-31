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
	"github.com/JameelGharra/ascii-craft/server/encoding/huffman"

	"github.com/JameelGharra/ascii-craft/server/ipc"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

const (
	TotalFrames = 10000
)

type CompressionStat struct {
	FrameIndex          int
	OriginalBytes       int
	CompressedBytes     int
	CompressedHuffBytes int
	Ratio               float64
	RatioHuff           float64
}

func TestCompressionWithRandomBot(t *testing.T) {
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
	outFileName := "compression_stats.csv"
	outFile, err := os.Create(outFileName)
	if err != nil {
		t.Fatalf("Failed to create statistics file: %v", err)
	}
	defer outFile.Close()

	fmt.Fprintln(outFile, "Frame,Original_Bytes,Compressed_RLE_Bytes,Compressed_Huff_Bytes,Savings_Percent_RLE, Saving_Percent_Huff") // csv header
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
	currentAsciiFrame := ascii.NewAsciiFrame(uint32(width), uint32(height))

	var stats []CompressionStat
	var totalOriginal int64
	var totalCompressedRLE int64
	var totalCompressedHuff int64

	startBench := time.Now()
	lastFrameTime := time.Now()

	buffers := [2]*ascii.AsciiFrame{
		ascii.NewAsciiFrame(uint32(width), uint32(height)),
		ascii.NewAsciiFrame(uint32(width), uint32(height)),
	}
	diff := ascii.NewAsciiFrame(uint32(width), uint32(height))
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

		frame.ToAsciiFrame(currentAsciiFrame)
		freq := ascii.NewFrequency()
		// to be properly refactored
		curr := buffers[frameNum%2]
		prev := buffers[(frameNum+1)%2]
		curr.Push(currentAsciiFrame.Buffer)
		diff.Xor(curr, prev)
		//
		freq.Count(utils.New8BitIterator(diff.Buffer)) // only color data
		huffman, err := huffman.NewHuffman(freq)
		if err != nil {
			t.Fatalf("Failed to create huffman encoder at frame %d: %v", frameNum, err)
		}
		huffmanEncodedResult := make([]byte, len(diff.Buffer))
		bitLength, err := huffman.Encode(utils.New8BitIterator(diff.Buffer), huffmanEncodedResult)
		if err != nil {
			t.Fatalf("Failed to huffman encode frame %d: %v", frameNum, err)
		}
		result, err := frameEncoder.Encode(currentAsciiFrame)

		origSize := len(currentAsciiFrame.Buffer)
		compSize := 0

		if err != nil {
			t.Fatalf("Encoding Error at frame %d: %v", frameNum, err)
		}
		compSize = len(result)

		ratio := 100.0 * (1.0 - (float64(compSize) / float64(origSize))) // saved percentage

		var huffCompSize int
		if bitLength%8 == 0 {
			huffCompSize = bitLength / 8
		} else {
			huffCompSize = bitLength/8 + 1
		}
		ratioHuff := 100.0 * (1.0 - (float64(huffCompSize) / float64(origSize)))
		fmt.Fprintf(outFile, "%d,%d,%d,%d,%.2f,%.2f\n", frameNum, origSize, compSize, huffCompSize, ratio, ratioHuff)

		stat := CompressionStat{
			FrameIndex:          frameNum,
			OriginalBytes:       origSize,
			CompressedBytes:     compSize,
			CompressedHuffBytes: huffCompSize,
			Ratio:               ratio,
			RatioHuff:           ratioHuff,
		}
		stats = append(stats, stat)

		totalOriginal += int64(origSize)
		totalCompressedRLE += int64(compSize)
		totalCompressedHuff += int64(huffCompSize)

		// if frameNum%1000 == 0 {
		// 	fmt.Printf("Frame %d/%d | Current Compression: %.2f%%\n", frameNum, TotalFrames, ratio)
		// }
		frameNum++
	}

	duration := time.Since(startBench)

	var maxRatioRLE, minRatioRLE float64 = -100.0, 100.0
	var avgRatioRLE, avgRatioHuff float64
	var maxRatioHuff, minRatioHuff float64 = -100.0, 100.0

	for _, s := range stats {
		if s.Ratio > maxRatioRLE {
			maxRatioRLE = s.Ratio
		}
		if s.Ratio < minRatioRLE {
			minRatioRLE = s.Ratio
		}
		if s.RatioHuff > maxRatioHuff {
			maxRatioHuff = s.RatioHuff
		}
		if s.RatioHuff < minRatioHuff {
			minRatioHuff = s.RatioHuff
		}
	}

	avgRatioRLE = 100.0 * (1.0 - (float64(totalCompressedRLE) / float64(totalOriginal)))
	avgRatioHuff = 100.0 * (1.0 - (float64(totalCompressedHuff) / float64(totalOriginal)))

	// killed the process to remove noisy output from the game
	if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
		fmt.Println("Stopping game process...")
		cmd.Process.Kill()
	}
	fmt.Print("\033[0m\033[2J\033[H\033[3J")

	fmt.Println("\n==========================================")
	fmt.Println("       GENERAL BENCHMARK          			")
	fmt.Println("==========================================")
	fmt.Printf("Total Frames:   %d\n", TotalFrames)
	fmt.Printf("Time Elapsed:   %v\n", duration)
	fmt.Printf("Average FPS:    %.2f\n", float64(TotalFrames)/duration.Seconds())
	fmt.Println("------------------------------------------")
	fmt.Println("\n==========================================")
	fmt.Println("       RLE COMPRESSION BENCHMARK          ")
	fmt.Println("==========================================")
	fmt.Printf("Total Data (Raw):  %.2f MB\n", float64(totalOriginal)/(1024*1024))
	fmt.Printf("Total Data (RLE):  %.2f MB\n", float64(totalCompressedRLE)/(1024*1024))
	fmt.Printf("Raw Data Written:  %s\n", outFileName)
	fmt.Println("------------------------------------------")
	fmt.Printf("Best Compression:  %.2f%% (Less detail/Sky)\n", maxRatioRLE)
	fmt.Printf("Worst Compression: %.2f%% (High noise/Trees)\n", minRatioRLE)
	fmt.Printf("AVERAGE SAVINGS:   %.2f%%\n", avgRatioRLE)
	fmt.Println("\n==========================================")
	fmt.Println("     HUFFMAN COMPRESSION BENCHMARK          ")
	fmt.Println("==========================================")
	fmt.Printf("Total Data (Huffman):  %.2f MB\n", float64(totalCompressedHuff)/(1024*1024))
	fmt.Println("------------------------------------------")
	fmt.Printf("Best Compression:  %.2f%% (Less detail/Sky)\n", maxRatioHuff)
	fmt.Printf("Worst Compression: %.2f%% (High noise/Trees)\n", minRatioHuff)
	fmt.Printf("AVERAGE SAVINGS:   %.2f%%\n", avgRatioHuff)
	fmt.Println("==========================================")
	if avgRatioRLE < 0 {
		t.Errorf("RLE is performing worse than raw data on average!")
	}
	if avgRatioHuff < 0 {
		t.Errorf("Huffman is performing worse than raw data on average!")
	}

}
