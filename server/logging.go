package main

import (
	"log/slog"
	"os"
)

// going to output to stderr instead of stdout incase we want to pipe the video stream
func InitLogger() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, opts))
	slog.SetDefault(logger)
}