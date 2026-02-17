package main

import (
	"github.com/gorilla/websocket"
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	clients    map[*Client]struct{} // surprised that Go doesn't have a set
	broadcast  chan []byte          // msgs to send from gameserver
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		// CASE 1: A new user joined
		case client := <-h.register:
			h.clients[client] = struct{}{}

		// CASE 2: A user left
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send) // Tell the client's pump to stop
			}

		// CASE 3: Video packet arrived from C game
		case packet := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- packet:
					// Success: Packet is in their mailbox.
				default:
					// DROP PACKET: The mailbox is full!
					// This user is too slow. We skip them to keep the game fast for others.
				}
			}
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		// Wait for a packet specifically for THIS user
		packet, ok := <-c.send
		if !ok {
			// The Hub closed the channel, time to disconnect
			return
		}

		// This is where the actual network waiting happens.
		// It only blocks THIS client's goroutine.
		err := c.conn.WriteMessage(websocket.BinaryMessage, packet)
		if err != nil {
			return
		}
	}
}
