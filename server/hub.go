package server

import (
	"fmt"

	"github.com/dylanconnolly/captrivia-be/redis"
)

// Hub tracks all active websocket clients and broadcasts
// messages to each client.
type Hub struct {
	// allClients stores all active clients
	allClients map[*Client]bool
	db         *redis.DB
	broadcast  chan []byte
	// clients stores only clients that are not actively in a game/lobby
	clients     map[*Client]bool
	clientNames map[string]bool
	close       chan *Client
	gameEvents  chan GameEvent
	register    chan *Client
	unregister  chan *Client
}

func NewHub(db *redis.DB) *Hub {
	return &Hub{
		allClients:  make(map[*Client]bool),
		db:          db,
		broadcast:   make(chan []byte),
		clients:     make(map[*Client]bool),
		close:       make(chan *Client),
		clientNames: make(map[string]bool),
		gameEvents:  make(chan GameEvent),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.allClients[client] = true
			h.clients[client] = true
			h.clientNames[client.name] = true
		// Unregister removes client from Hub but keeps them in allClients to receive broadcast messages
		case client := <-h.unregister:
			delete(h.clients, client)
		case message := <-h.broadcast:
			for client := range h.allClients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		// send game state changes to clients which are not actively in a game
		case event := <-h.gameEvents:
			for client := range h.clients {
				select {
				case client.send <- event.toBytes():
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			fmt.Println(event)
		case client := <-h.close:
			delete(h.allClients, client)
			delete(h.clients, client)
			delete(h.clientNames, client.name)
			close(client.send)
		}
	}
}
