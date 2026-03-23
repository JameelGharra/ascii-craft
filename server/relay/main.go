package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
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
	client.readPump()
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
