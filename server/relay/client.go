package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gorilla/websocket"
)

const maxMessageSize = 32

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	sendVideo chan []byte
	sendText  chan string
}

func (c *Client) readPump() {
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

func (c *Client) writePump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		select {
		// Wait for a packet specifically for THIS user
		case packet, ok := <-c.sendVideo:
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
		case text, ok := <-c.sendText:
			if !ok {
				return
			}
			err := c.conn.WriteMessage(websocket.TextMessage, []byte(text))
			if err != nil {
				return
			}
		}
	}
}
