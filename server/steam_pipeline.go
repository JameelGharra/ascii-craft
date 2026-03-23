package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/JameelGharra/ascii-craft/server/encoder"
	"github.com/JameelGharra/ascii-craft/server/ipc"
	"github.com/JameelGharra/ascii-craft/server/protocol"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

type StreamPipeline struct {
	enc       *encoder.Encoder
	frameBuf  []byte
	packetBuf []byte
	config    Config
	rng       *rand.Rand
}

func NewStreamPipeline(width, height int, cfg Config) *StreamPipeline {
	enc := encoder.NewEncoder(width*height, cfg.Stride)
	enc.AddEncoding(encoder.Raw)
	enc.AddEncoding(encoder.XorRLE)
	enc.AddEncoding(encoder.Huffman)

	return &StreamPipeline{
		enc:       enc,
		frameBuf:  make([]byte, width*height),
		packetBuf: make([]byte, 1024*1024), // 1mb packet buff
		config:    cfg,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// blocks and pushes frames to the relay, returns an error if the relay disconnects
// (so we can retry) or if the context/game ends (so we can exit).
func (s *StreamPipeline) Run(ctx context.Context, client *ipc.Client, conn *RelayConnection, gameDone <-chan error) error {
	lastFrameTime := time.Now()
	packetBuilder := protocol.NewPacketBuilder(s.packetBuf)
	var headerbuff [5]byte

	fpsTicker := time.NewTicker(s.config.FrameInterval)
	defer fpsTicker.Stop()

	possibleCmds := []uint32{
		ipc.CmdForward, ipc.CmdBackward,
		ipc.CmdTurnLeft, ipc.CmdTurnRight,
		ipc.CmdLookUp, ipc.CmdLookDown,
		ipc.CmdJump, ipc.CmdJumpForward,
		ipc.CmdSelectSlot,
	}

	frameNum := 0
	for s.config.TotalFrames <= 0 || frameNum < s.config.TotalFrames {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-gameDone:
			return fmt.Errorf("game crashed at frame %d: %w", frameNum, err)
		case <-fpsTicker.C:
		}

		if time.Since(lastFrameTime) > s.config.KeyFrameInterval {
			return fmt.Errorf("game stopped producing frames at %d (hang/deadlock detected)", frameNum)
		}

		if s.config.RunMode == BotMode && s.rng.Float64() < 0.05 {
			cmdType := possibleCmds[s.rng.Intn(len(possibleCmds))]
			var val int32 = 0
			if cmdType == ipc.CmdSelectSlot {
				val = int32(s.rng.Intn(10))
			}
			client.WriteCommand(cmdType, val)
		}

		frame, isNew := client.TryReadFrame()
		if !isNew {
			continue
		}

		lastFrameTime = time.Now()
		frame.ToColor8bit(s.frameBuf)

		isKeyFrame := frameNum%s.config.RefreshRate == 0
		result := s.enc.PushFrame(s.frameBuf, isKeyFrame)

		slog.Info("frame encoded",
			"frame_num", frameNum,
			"original_bytes", len(s.frameBuf),
			"compressed_bytes", result.FinalSize,
			"encoding", result.Encoding.String(),
			"is_keyframe", isKeyFrame,
		)

		packetBuilder.Reset()
		if err := s.enc.WriteTo(packetBuilder); err != nil {
			return fmt.Errorf("encoding error at frame %d: %w", frameNum, err)
		}

		payload := packetBuilder.Bytes()
		payloadLen := uint32(len(payload))
		headerbuff[0] = 0x02 // 0x02 = Video Frame
		n, _ := utils.PutVarint(headerbuff[1:], payloadLen)

		if err := conn.WriteFrame(headerbuff[:n+1], payload); err != nil {
			return fmt.Errorf("relay disconnected at frame %d: %w", frameNum, err)
		}
		frameNum++
	}

	return nil // incase totalframes is finite
}
