package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// 1. The Upgrader configuration
var upgrader = websocket.Upgrader{
	// High-performance buffers for the physical network connection
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// For development, we allow anyone to connect.
	// In production, you might restrict this to your specific website domain.
	CheckOrigin: func(r *http.Request) bool { return true },
}

const (
	maxMessageSize            = 32
	ConfigPacketType          = 0x01
	VideoGameserverPacketType = 0x02
)

type AppConfig struct {
	Video struct {
		Width  uint32 `json:"width"`
		Height uint32 `json:"height"`
	} `json:"video"`
	Chat struct {
		CooldownMs  int `json:"cooldown_ms"`
		MaxMessages int `json:"max_messages"`
	} `json:"chat"`
	Commands map[string]interface{} `json:"commands"`
}

type ConfigManager struct {
	mu     sync.RWMutex
	config AppConfig
}

func NewDefaultConfigManager() *ConfigManager {
	return &ConfigManager{
		config: AppConfig{
			// Default Relay UX settings
			Chat: struct {
				CooldownMs  int `json:"cooldown_ms"`
				MaxMessages int `json:"max_messages"`
			}{CooldownMs: 500, MaxMessages: 35},
			// Safe defaults just in case UI fetches before game connects
			Video: struct {
				Width  uint32 `json:"width"`
				Height uint32 `json:"height"`
			}{Width: 212, Height: 66},
			// these just to make sure that we live even gameserver off
			Commands: map[string]interface{}{
				"standard":      []string{},
				"parameterized": map[string]any{},
			},
		},
	}
}

func (cm *ConfigManager) UpdateGameConfig(width, height uint32, commands map[string]interface{}) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config.Video.Width = width
	cm.config.Video.Height = height
	cm.config.Commands = commands
}

func (cm *ConfigManager) GetJSON() []byte {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	data, _ := json.Marshal(cm.config)
	return data
}

func readPump(c *Client) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		// Block until user sends message or disconnects
		msgType, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		if msgType == websocket.TextMessage {
			// ping-pong RTT (the ping from client side btw)
			if len(msg) > 0 && msg[0] == '{' {
				var ping struct {
					Type string  `json:"type"`
					T    float64 `json:"t"` // float64 just to keep high res same as js but i might avoid it
				}

				if err := json.Unmarshal(msg, &ping); err == nil && ping.Type == "ping" {
					pongMsg := fmt.Sprintf(`{"type":"pong","t":%f}`, ping.T)
					select {
					case c.sendText <- pongMsg:
					default:
					}
					continue
				}
			}

			// chats/cmds
			text := strings.TrimSpace(string(msg))
			if len(text) > 0 && len(text) <= maxMessageSize {
				c.hub.commands <- clientCommand{client: c, cmd: text}
			}
		}
	}
}
func handleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// 1. Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WS upgrade error", "error", err)
		return
	}

	// 2. Create Client
	client := &Client{
		hub:       hub,
		conn:      conn,
		sendVideo: make(chan []byte, 64), // Buffer of 64 frames (~2 seconds)
		sendText:  make(chan string, 16), // Buffer of 16 chat messages
	}

	// 3. Register with Hub
	hub.register <- client

	// 4. Start Threads
	// Thread A: Writes data TO the user
	go client.writePump()

	// Thread B: Reads data FROM the user (keep-alive/chat)
	// This runs in the *current* goroutine allocated by http.HandleFunc
	readPump(client)
}

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

func handleConfig(cm *ConfigManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.Write(cm.GetJSON())
	}
}

func main() {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, opts)))

	slog.Info("--- Starting ASCII Craft Relay ---")

	// create and start the hub
	// the hub runs in its own background thread (goroutine)
	cm := NewDefaultConfigManager()
	hub := NewHub()
	go hub.Run()

	// start the TCP Server (game ingest)
	// we run this in a goroutine so it doesn't block the HTTP server below
	go startTCPServer(hub, cm)

	// config endpoint
	http.HandleFunc("/api/config", handleConfig(cm))

	// configure the HTTP route for websockets
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	})

	// start the HTTP Server (User Connections)
	// this blocks the main thread forever (keeping the program alive)
	slog.Info("TCP Listening for game engine", "port", 9000)
	slog.Info("HTTP Listening for users", "port", 8080)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		slog.Error("HTTP Server failed", "error", err)
	}
}
