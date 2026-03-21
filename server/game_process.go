package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/JameelGharra/ascii-craft/server/ipc"
)

type GameProcess struct {
	cmd  *exec.Cmd
	done chan error
}

func NewGameProcess(cfg Config) (*GameProcess, error) {
	absPath, err := filepath.Abs(cfg.BinaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path of game binary: %w", err)
	}

	cmd := exec.Command(absPath)
	cmd.Dir = filepath.Dir(absPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start game: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	return &GameProcess{
		cmd:  cmd,
		done: done,
	}, nil
}

// polls for the shared memory connection until it succeeds,
// the game crashes, or the context is cancelled
func (g *GameProcess) WaitForIPC(ctx context.Context) (*ipc.Client, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("cancelled while waiting for IPC: %w", ctx.Err())
		case err := <-g.done:
			return nil, fmt.Errorf("game process exited prematurely: %w", err)
		case <-ticker.C:
			client, err := ipc.NewClient()
			if err == nil {
				return client, nil
			}
		}
	}
}

func (g *GameProcess) Done() <-chan error {
	return g.done
}

func (g *GameProcess) Kill() {
	if g.cmd != nil && g.cmd.Process != nil {
		if g.cmd.ProcessState == nil || !g.cmd.ProcessState.Exited() {
			fmt.Println("Stopping game process...")
			g.cmd.Process.Kill()
		}
	}
}
