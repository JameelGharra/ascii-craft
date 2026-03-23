package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/ipc"
)

//go:generate go run tools/gen_protocol/main.go

func main() {
	InitLogger()
	slog.Info("Starting ASCII-Craft stream server")
	cfg := DefaultConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		slog.Warn("Interrupt received, initiating graceful shutdown")
		cancel()
	}()
	game, err := NewGameProcess(cfg)
	if err != nil {
		slog.Error("failed to launch game process", "error", err)
		return
	}
	defer game.Kill()

	slog.Info("Game launched. Now connecting to IPC")
	ipcClient, err := game.WaitForIPC(ctx)
	if err != nil {
		slog.Error("IPC connection failed", "error", err)
		return
	}
	defer ipcClient.Close()

	// gonna wait for first frame for resolution
	frame, _ := waitForFirstFrame(ctx, ipcClient, game.Done())
	width, height := int(frame.Width), int(frame.Height)
	slog.Info("first frame received", "width", width, "height", height)

	pipeline := NewStreamPipeline(width, height, cfg)
	dispatcher := NewCommandDispatcher(ipcClient)
	const retryDelaySeconds = 2
	for {
		slog.Info("connecting to relay", "address", cfg.RelayAddr)
		relay, err := DialRelay(ctx, cfg.RelayAddr, width, height)
		if err != nil {
			if ctx.Err() != nil {
				return // prolly a ctrl+c
			}
			slog.Warn("relay connection failed", "error", err, "retry_delay_seconds", retryDelaySeconds)
			time.Sleep(retryDelaySeconds * time.Second)
			continue
		}
		slog.Info("Connected to relay. Starting stream...")
		relay.StartCommandReader(dispatcher)

		err = pipeline.Run(ctx, ipcClient, relay, game.Done()) // blocky

		relay.Close()

		if err != nil {
			slog.Error("Pipeline stopped", "error", err)
			// if it is a game crash or a ctx cancellation => no retrying
			if ctx.Err() != nil || err.Error() == "game crashed" {
				return
			}
			slog.Info("Retrying relay connection..", "retry_delay_seconds", retryDelaySeconds)
			time.Sleep(retryDelaySeconds * time.Second)
		} else {
			slog.Info("Stream completed requested frames.")
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
