package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/JameelGharra/ascii-craft/server/utils"
)

type RelayConnection struct {
	conn net.Conn
}

// connects and retries, until context cancelled
func DialRelay(ctx context.Context, addr string, width, height int) (*RelayConnection, error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	if r, err := attemptConnect(addr, width, height); err == nil {
		return r, nil
	}

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled while dialing relay: %w", ctx.Err())
		case <-ticker.C:
			if r, err := attemptConnect(addr, width, height); err == nil {
				return r, nil
			}
		}
	}
}

func attemptConnect(addr string, width, height int) (*RelayConnection, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	if err := dynamicHandshake(conn, width, height); err != nil {
		conn.Close()
		return nil, fmt.Errorf("handshake failed: %w", err)
	}
	return &RelayConnection{conn: conn}, nil
}

func dynamicHandshake(conn net.Conn, width, height int) error {
	gameConfig := map[string]interface{}{
		"video": map[string]int{
			"width":  width,
			"height": height,
		},
		"commands": map[string]interface{}{
			"standard": []string{
				"!w", "!a", "!s", "!d", "!jump", "!fly", "!build", "!destroy",
				"!turnleft", "!turnright", "!lookup", "!lookdown", "!lookleft", "!lookright",
				"!jumpforward", "!jumpbackward", "!jumpleft", "!jumpright",
			},
			"parameterized": map[string]interface{}{
				"!slot": map[string]int{"min": 0, "max": 9},
			},
		},
	}

	configBytes, err := json.Marshal(gameConfig)
	if err != nil {
		return err
	}

	//  [type (1 byte)] [varint Len] [data]
	var header [10]byte
	header[0] = 0x01 // 0x01 for handshake btw
	n, _ := utils.PutVarint(header[1:], uint32(len(configBytes)))

	if _, err := conn.Write(header[:n+1]); err != nil {
		return err
	}
	if _, err := conn.Write(configBytes); err != nil {
		return err
	}
	return nil
}

func (r *RelayConnection) StartCommandReader(dispatcher *CommandDispatcher) {
	go func() {
		scanner := bufio.NewScanner(r.conn)
		for scanner.Scan() {
			cmdStr := scanner.Text()
			dispatcher.Dispatch(cmdStr)
		}
		if err := scanner.Err(); err != nil {
			slog.Error("Relay command reader failed", "error", err)
		}
	}()
}

func (r *RelayConnection) WriteFrame(header, payload []byte) error {
	buffers := net.Buffers{header, payload}
	_, err := buffers.WriteTo(r.conn)
	return err
}

func (r *RelayConnection) Close() error {
	return r.conn.Close()
}
