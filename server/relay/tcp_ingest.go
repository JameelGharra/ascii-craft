package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
)

const (
	ConfigPacketType          = 0x01
	VideoGameserverPacketType = 0x02
)

// Start the TCP server in a blocking way or a goroutine
func startTCPServer(hub *Hub, cm *ConfigManager) {
	// 1. Listen on a raw TCP port
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		slog.Error("Error starting TCP server", "error", err)
	}
	defer listener.Close()

	slog.Info("TCP game ingest listening", "port", 9000)

	for {
		// 2. Accept new connections
		// This blocks until your Game Engine calls connect()
		conn, err := listener.Accept()
		if err != nil {
			slog.Error("TCP accept error", "error", err)
			continue
		}

		// 3. Handle the connection
		// We use a Goroutine so if you restart the game,
		// the relay can accept the new connection without restarting.
		go handleGameConnection(conn, hub, cm)
	}
}

func handleGameConnection(conn net.Conn, hub *Hub, cm *ConfigManager) {
	defer conn.Close()
	slog.Info("game engine connected!")

	// context to cleanly kill the writer goroutine if game connection drops
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// tcp writer (relay -> game server)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return // reading failed
			case cmd := <-hub.gameCmds:
				_, err := fmt.Fprintf(conn, "%s\n", cmd)
				if err != nil {
					slog.Error("failed to write command to game server", "error", err)
					return
				}
			}
		}
	}()

	// tcp reader (game server -> relay)
	// 4. Create a Buffered Reader
	// CRITICAL PERFORMANCE OPTIMIZATION
	reader := bufio.NewReader(conn)
	packetCount := 0 // only counting video frame packets
	for {

		packetType, err := reader.ReadByte()
		if err != nil {
			slog.Error("game engine disconnected", "error", err)
			return
		}

		// 5. Use our Framer (from utils.go)
		packet, err := ReadFullPacket(reader, reader) // i know sounds like a typo but its not
		if err != nil {
			// If err is EOF, the game crashed or closed.
			slog.Error("Game Engine Disconnected", "error", err)
			return
		}

		switch packetType {
		case ConfigPacketType:
			// Type 0x01: Handshake / Config
			var incoming struct {
				Video struct {
					Width  uint32 `json:"width"`
					Height uint32 `json:"height"`
				} `json:"video"`
				Commands map[string]interface{} `json:"commands"`
			}
			if err := json.Unmarshal(packet, &incoming); err == nil {
				cm.UpdateGameConfig(incoming.Video.Width, incoming.Video.Height, incoming.Commands)
				slog.Info("Game Config Updated", "width", incoming.Video.Width, "height", incoming.Video.Height)
				// Tell all frontend clients to reload their config
				hub.broadcastText <- `{"type":"reload_config"}`
			}
		case VideoGameserverPacketType:
			// Type 0x02: Video Frame
			// 6. Send to the Hub
			// This sends the raw bytes to the broadcast channel.
			// The Hub will pick this up and fan it out to the 1000 websockets.
			hub.broadcast <- packet
			slog.Info("Received packet", "count", packetCount, "size", len(packet))
			packetCount++
		}
	}
}
