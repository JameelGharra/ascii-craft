package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/ipc"
)

//go:generate go run tools/gen_protocol/main.go

func main() {
	fmt.Println("Starting ASCII-Craft stream server...")
	cfg := DefaultConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nInterrupt received, initiating graceful shutdown...")
		cancel()
	}()
	game, err := NewGameProcess(cfg)
	if err != nil {
		fmt.Printf("Failed to launch game process %v\n", err)
		return
	}
	defer game.Kill()

	fmt.Println("Game launched. Now connecting to IPC...")
	ipcClient, err := game.WaitForIPC(ctx)
	if err != nil {
		fmt.Printf("IPC connection failed: %v\n", err)
		return
	}
	defer ipcClient.Close()

	// gonna wait for first frame for resolution
	frame, _ := waitForFirstFrame(ctx, ipcClient, game.Done())
	width, height := int(frame.Width), int(frame.Height)
	fmt.Printf("First frame received. Resolution: %dx%d\n", width, height)

	pipeline := NewStreamPipeline(width, height, cfg)
	dispatcher := NewCommandDispatcher(ipcClient)
	const retryDelaySeconds = 2
	for {
		fmt.Println("Connecting to relay...")
		relay, err := DialRelay(ctx, cfg.RelayAddr, width, height)
		if err != nil {
			if ctx.Err() != nil {
				return // prolly a ctrl+c
			}
			fmt.Printf("Relay connection failed: %v. Retrying in %ds\n", err, retryDelaySeconds)
			time.Sleep(retryDelaySeconds * time.Second)
			continue
		}
		fmt.Println("Connected to relay. Starting stream...")
		relay.StartCommandReader(dispatcher)

		err = pipeline.Run(ctx, ipcClient, relay, game.Done()) // blocky

		relay.Close()

		if err != nil {
			fmt.Printf("Pipeline stopped: %v\n", err)
			// if it is a game crash or a ctx cancellation => no retrying
			if ctx.Err() != nil || err.Error() == "game crashed" {
				return
			}
			fmt.Printf("Retrying relay connection in %ds...\n", retryDelaySeconds)
			time.Sleep(retryDelaySeconds * time.Second)
		} else {
			fmt.Println("Stream completed requested frames.")
			return
		}
	}
}

func waitForFirstFrame(ctx context.Context, client *ipc.Client, gameDone <-chan error) (*ascii.Frame, error) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-gameDone:
			return nil, fmt.Errorf("game crashed before first frame: %w", err)
		case <-ticker.C:
			frame, ok := client.TryReadFrame()
			if ok {
				return frame, nil
			}
		}
	}
}
