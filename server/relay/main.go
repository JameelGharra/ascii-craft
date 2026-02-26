package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

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

const maxMessageSize = 32

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
		log.Println("WS Upgrade error:", err)
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
func startTCPServer(hub *Hub) {
	// 1. Listen on a raw TCP port
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Error starting TCP server: %v", err)
	}
	defer listener.Close()

	log.Println("TCP Game Ingest listening on :9000")

	for {
		// 2. Accept new connections
		// This blocks until your Game Engine calls connect()
		conn, err := listener.Accept()
		if err != nil {
			log.Println("TCP Accept error:", err)
			continue
		}

		// 3. Handle the connection
		// We use a Goroutine so if you restart the game,
		// the relay can accept the new connection without restarting.
		go handleGameConnection(conn, hub)
	}
}

func handleGameConnection(conn net.Conn, hub *Hub) {
	defer conn.Close()
	log.Println("Game Engine Connected!")

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
					log.Println("Failed to write command to game server:", err)
					return
				}
			}
		}
	}()

	// tcp reader (game server -> relay)
	// 4. Create a Buffered Reader
	// CRITICAL PERFORMANCE OPTIMIZATION
	reader := bufio.NewReader(conn)
	packetCount := 0
	for {
		// 5. Use our Framer (from utils.go)
		packet, err := ReadFullPacket(reader, reader) // i know sounds like a typo but its not
		if err != nil {
			// If err is EOF, the game crashed or closed.
			log.Println("Game Engine Disconnected:", err)
			return
		}

		// 6. Send to the Hub
		// This sends the raw bytes to the broadcast channel.
		// The Hub will pick this up and fan it out to the 1000 websockets.
		fmt.Printf("Received packet (%d): %d bytes\n", packetCount, len(packet))
		hub.broadcast <- packet
		packetCount++
	}
}

func main() {
	fmt.Println("--- Starting ASCII Craft Relay ---")

	// 1. Create and Start the Hub
	// The Hub runs in its own background thread (Goroutine)
	hub := NewHub()
	go hub.Run()

	// 2. Start the TCP Server (Game Ingest)
	// We run this in a Goroutine so it doesn't block the HTTP server below
	go startTCPServer(hub)

	// 3. Configure the HTTP Route for WebSockets
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	})

	// 4. Start the HTTP Server (User Connections)
	// This blocks the main thread forever (keeping the program alive)
	fmt.Println(" -> TCP Listening on :9000 (Game Engine)")
	fmt.Println(" -> HTTP Listening on :8080 (Web Clients)")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("HTTP Server failed:", err)
	}
}
