package main

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

const (
	viewerTickerInterval = 2 * time.Second
	voteTickerInterval   = 500 * time.Millisecond
	commandBufferSize    = 256
	gameCmdsBufferSize   = 10
)

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	sendVideo chan []byte
	sendText  chan string
}

type clientCommand struct { // mainly to add 1 user = 1 vote support
	client *Client
	cmd    string
}

type Hub struct {
	clients       map[*Client]struct{} // surprised that Go doesn't have a set
	broadcast     chan []byte          // msgs to send from gameserver
	broadcastText chan string          // for events to all clients
	register      chan *Client
	unregister    chan *Client

	commands chan clientCommand // cmds aggregated from clients
	gameCmds chan string        // the cmds that won sent with tcp
}

func NewHub() *Hub {
	return &Hub{
		clients:       make(map[*Client]struct{}),
		broadcast:     make(chan []byte),
		broadcastText: make(chan string),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		commands:      make(chan clientCommand, commandBufferSize),
		gameCmds:      make(chan string, gameCmdsBufferSize),
	}
}

func (h *Hub) Run() {

	viewerTicker := time.NewTicker(viewerTickerInterval)
	defer viewerTicker.Stop()

	voteTicker := time.NewTicker(voteTickerInterval)
	defer voteTicker.Stop()
	tally := NewVoteTally()

	for {
		select {
		// new user joined
		case client := <-h.register:
			h.clients[client] = struct{}{}

		// user left
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.sendVideo) // Tell the client's pump to stop
				close(client.sendText)
			}

		case textMsg := <-h.broadcastText:
			for client := range h.clients {
				select {
				case client.sendText <- textMsg:
				default:
				}
			}
		// new frame from game
		case packet := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.sendVideo <- packet:
					// Success: Packet is in their mailbox.
				default:
					// DROP PACKET: The mailbox is full!
					// This user is too slow. We skip them to keep the game fast for others.
				}
			}
		case cmdClient := <-h.commands:
			tally.Record(cmdClient.client, cmdClient.cmd)

		case <-viewerTicker.C:
			viewerMsg := fmt.Sprintf(`{"type":"viewers", "count":%d}`, len(h.clients))
			for client := range h.clients {
				select {
				case client.sendText <- viewerMsg:
				default:
				}
			}
		case <-voteTicker.C:
			if tally.HasVotes() {
				winner, maxVotes := tally.Winner()
				tally.Reset()
				announcement := fmt.Sprintf(`{"type":"vote", "command":"%s", "votes":%d}`, winner, maxVotes)

				for client := range h.clients {
					select {
					case client.sendText <- announcement:
					default:
					}
				}
				select {
				case h.gameCmds <- winner:
				default:
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
