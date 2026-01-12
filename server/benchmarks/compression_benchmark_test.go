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
	"github.com/JameelGharra/ascii-craft/server/ascii/quad_tree"
	"github.com/JameelGharra/ascii-craft/server/encoder"
	"github.com/JameelGharra/ascii-craft/server/encoding/huffman"

	"github.com/JameelGharra/ascii-craft/server/ipc"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

const (
	TotalFrames = 1000000
)

type CompressionStat struct {
	FrameIndex               int
	OriginalBytes            int
	CompressedBytes          int
	CompressedHuffBytes      int
	CompressedHuffBoxedBytes int
	Ratio                    float64
	RatioHuff                float64
	RatioHuffBoxed           float64
}

func TestCompressionWithRandomBot(t *testing.T) {
	absPath, err := filepath.Abs(BinaryPath)
	if err != nil {
		t.Fatalf("Path error: %v", err)
	}

	cmd := exec.Command(absPath)
	cmd.Dir = filepath.Dir(absPath)
	// cmd.Stdout = os.Stdout
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

	fmt.Fprintln(outFile, "Frame,Original_Bytes,Compressed_Hybrid_Bytes,Compressed_Huff_Bytes,Compressed_Huff_Boxed,Savings_Percent_RLE,Saving_Percent_Huff,Saving_Percent_Huff_Boxed") // csv header
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

	frameEncoder := encoder.NewEncoder(width*height, quad_tree.QuadTreeParam{
		Depth:  2,
		Rows:   height,
		Cols:   width,
		Stride: 1,
	})
	frameEncoder.AddEncoding(encoder.XorRLE)
	frameEncoder.AddEncoding(encoder.Huffman)

	coloredFrame := make([]byte, width*height)

	var stats []CompressionStat
	var totalOriginal int64
	var totalCompressed int64
	var totalCompressedHuff int64
	var totalCompressedHuffBoxed int64

	// for quad tree huffman approach
	quadParam := quad_tree.QuadTreeParam{
		Rows:   height,
		Cols:   width,
		Depth:  3,
		Stride: 1,
	}

	startBench := time.Now()
	lastFrameTime := time.Now()

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

		frame.ToColor8bit(coloredFrame)
		freq := ascii.NewFrequency()
		freq.Count(utils.New8BitIterator(coloredFrame))
		normalHuffman, err := huffman.NewHuffman(freq)
		if err != nil {
			t.Fatalf("Failed to create huffman encoder at frame %d: %v", frameNum, err)
		}
		huffmanEncodedResult := make([]byte, len(coloredFrame))
		bitLength, err := normalHuffman.Encode(utils.New8BitIterator(coloredFrame), huffmanEncodedResult)
		if err != nil {
			t.Fatalf("Failed to huffman encode frame %d: %v", frameNum, err)
		}

		// quad stuff
		boxes := quad_tree.Partition(coloredFrame, quadParam)
		totalHuffSerialSize, totalHuffBitsBoxes, totalHuffBoxEverything := 0, 0, 0
		for _, box := range boxes {
			boxFreq := ascii.NewFrequency()
			boxFreq.Count(box)
			boxHuffman, err := huffman.NewHuffman(boxFreq)
			if err != nil {
				t.Fatalf("Failed to create box huffman encoder at frame %d: %v", frameNum, err)
			}
			boxHuffEncodedResult := make([]byte, box.BoxRows*box.BoxCols)
			box.ResetIterator()
			boxBitLength, err := boxHuffman.Encode(box, boxHuffEncodedResult)
			if err != nil {
				t.Fatalf("Failed to huffman encode box at frame %d: %v", frameNum, err)
			}
			totalHuffBitsBoxes += boxBitLength
			totalHuffSerialSize += boxHuffman.TreeSize
		}
		totalHuffBitsBoxesInBytes := 0
		if totalHuffBitsBoxes%8 != 0 {
			totalHuffBitsBoxesInBytes = totalHuffBitsBoxes/8 + 1
		} else {
			totalHuffBitsBoxesInBytes = totalHuffBitsBoxes / 8
		}
		totalHuffBoxEverything += totalHuffSerialSize + totalHuffBitsBoxesInBytes
		//
		var newFrame = make([]byte, len(coloredFrame))
		copy(newFrame, coloredFrame)
		result := frameEncoder.PushFrame(newFrame)

		origSize := len(coloredFrame)
		compSize := 0
		if result != nil {
			compSize = result.Len // for choose the best approach
		} else {
			compSize = len(coloredFrame)
		}
		ratio := 100.0 * (1.0 - (float64(compSize) / float64(origSize))) // saved percentage

		var huffCompSize int
		if bitLength%8 == 0 {
			huffCompSize = bitLength / 8
		} else {
			huffCompSize = bitLength/8 + 1
		}
		huffCompSize += normalHuffman.TreeSize
		ratioHuff := 100.0 * (1.0 - (float64(huffCompSize) / float64(origSize)))
		ratioHuffBoxed := 100.0 * (1.0 - (float64(totalHuffBoxEverything) / float64(origSize)))
		fmt.Fprintf(outFile, "%d,%d,%d,%d,%d,%.2f,%.2f, %.2f\n", frameNum, origSize, compSize, huffCompSize, totalHuffBoxEverything, ratio, ratioHuff, ratioHuffBoxed)

		stat := CompressionStat{
			FrameIndex:               frameNum,
			OriginalBytes:            origSize,
			CompressedBytes:          compSize,
			CompressedHuffBytes:      huffCompSize,
			CompressedHuffBoxedBytes: totalHuffBoxEverything,
			Ratio:                    ratio,
			RatioHuff:                ratioHuff,
			RatioHuffBoxed:           ratioHuffBoxed,
		}
		stats = append(stats, stat)

		totalOriginal += int64(origSize)
		totalCompressed += int64(compSize)
		totalCompressedHuff += int64(huffCompSize)
		totalCompressedHuffBoxed += int64(totalHuffBoxEverything)

		frameNum++
	}

	duration := time.Since(startBench)

	var avgRatio, avgRatioHuff, avgRatioHuffBoxed float64
	var maxRatio, minRatio float64 = -100.0, 100.0
	var maxRatioHuff, minRatioHuff float64 = -100.0, 100.0
	var maxRatioHuffBoxed, minRatioHuffBoxed float64 = -100.0, 100.0

	for _, s := range stats {
		if s.Ratio > maxRatio {
			maxRatio = s.Ratio
		}
		if s.Ratio < minRatio {
			minRatio = s.Ratio
		}
		if s.RatioHuff > maxRatioHuff {
			maxRatioHuff = s.RatioHuff
		}
		if s.RatioHuff < minRatioHuff {
			minRatioHuff = s.RatioHuff
		}
		if s.RatioHuffBoxed > maxRatioHuffBoxed {
			maxRatioHuffBoxed = s.RatioHuffBoxed
		}
		if s.RatioHuffBoxed < minRatioHuffBoxed {
			minRatioHuffBoxed = s.RatioHuffBoxed
		}
	}

	avgRatio = 100.0 * (1.0 - (float64(totalCompressed) / float64(totalOriginal)))
	avgRatioHuff = 100.0 * (1.0 - (float64(totalCompressedHuff) / float64(totalOriginal)))
	avgRatioHuffBoxed = 100.0 * (1.0 - (float64(totalCompressedHuffBoxed) / float64(totalOriginal)))

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
	fmt.Println("       CHOOSE BEST COMPRESSION BENCHMARK    ")
	fmt.Println("============================================")
	fmt.Printf("Total Data:  	   %.2f MB\n", float64(totalOriginal)/(1024*1024))
	fmt.Printf("Total Data (comp): %.2f MB\n", float64(totalCompressed)/(1024*1024))
	fmt.Printf("Raw Data Written:  %s\n", outFileName)
	fmt.Println("------------------------------------------")
	fmt.Printf("Best Compression:  %.2f%% (Less detail/Sky)\n", maxRatio)
	fmt.Printf("Worst Compression: %.2f%% (High noise/Trees)\n", minRatio)
	fmt.Printf("AVERAGE SAVINGS:   %.2f%%\n", avgRatio)
	fmt.Println("\n==========================================")
	fmt.Println("     HUFFMAN COMPRESSION BENCHMARK          ")
	fmt.Println("==========================================")
	fmt.Printf("Total Data (Huffman):  %.2f MB\n", float64(totalCompressedHuff)/(1024*1024))
	fmt.Println("------------------------------------------")
	fmt.Printf("Best Compression:  %.2f%% (Less detail/Sky)\n", maxRatioHuff)
	fmt.Printf("Worst Compression: %.2f%% (High noise/Trees)\n", minRatioHuff)
	fmt.Printf("AVERAGE SAVINGS:   %.2f%%\n", avgRatioHuff)
	fmt.Println("================================================")
	fmt.Println("\n==============================================")
	fmt.Println("     QUAD-TREES + HUFFMAN COMPRESSION BENCHMARK")
	fmt.Println("================================================")
	fmt.Println("Depth of Quad Tree:", quadParam.Depth)
	fmt.Printf("Total Data (Quad-Huffman):  %.2f MB\n", float64(totalCompressedHuffBoxed)/(1024*1024))
	fmt.Println("------------------------------------------")
	fmt.Printf("Best Compression:  %.2f%% (Less detail/Sky)\n", maxRatioHuffBoxed)
	fmt.Printf("Worst Compression: %.2f%% (High noise/Trees)\n", minRatioHuffBoxed)
	fmt.Printf("AVERAGE SAVINGS:   %.2f%%\n", avgRatioHuffBoxed)
	fmt.Println("==========================================")
	if avgRatio < 0 {
		t.Errorf("RLE is performing worse than raw data on average!")
	}
	if avgRatioHuff < 0 {
		t.Errorf("Huffman is performing worse than raw data on average!")
	}
	if avgRatioHuffBoxed < 0 {
		t.Errorf("Boxed Huffman is performing worse than raw data on average!")
	}
}
