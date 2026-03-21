package main

import "time"

const (
	BotMode        = 0 // rng based cmds
	ControlledMode = 1
)

type Config struct {
	TotalFrames      int
	BinaryPath       string
	Stride           int
	RefreshRate      int
	RunMode          int
	FrameInterval    time.Duration
	KeyFrameInterval time.Duration
	RelayAddr        string
}

func DefaultConfig() Config {
	return Config{
		TotalFrames:      0, // 0 for infinite frames
		BinaryPath:       "../game/craft.exe",
		Stride:           1,
		RefreshRate:      120, // after how many frames to send key frame (i-frame)
		RunMode:          ControlledMode,
		FrameInterval:    time.Second / 60, // capping it so it won't ruin the browser
		KeyFrameInterval: 2 * time.Second,  // keyframe refresh/timeout
		RelayAddr:        "localhost:9000",
	}
}
