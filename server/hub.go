package server

import (
	"context"
	"fmt"

	"github.com/dylanconnolly/captrivia-be/captrivia"
)

// Hub tracks all active websocket clients and broadcasts
// messages to each client.
type Hub struct {
	// allClients stores all active clients
	allClients  map[*Client]bool
	GameService captrivia.GameService
	broadcast   chan []byte
	// clients stores only clients that are not actively in a game/lobby
	clients     map[*Client]bool
	clientNames map[string]bool
	disconnect  chan *Client
	gameEvents  chan GameEvent
	register    chan *Client
	unregister  chan *Client
}

func NewHub(gs captrivia.GameService) *Hub {
	return &Hub{
		allClients:  make(map[*Client]bool),
		GameService: gs,
		broadcast:   make(chan []byte),
		clients:     make(map[*Client]bool),
		disconnect:  make(chan *Client),
		clientNames: make(map[string]bool),
		gameEvents:  make(chan GameEvent),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
	}
}

func (h *Hub) Run(ctx context.Context) {
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
		case client := <-h.disconnect:
			delete(h.allClients, client)
			delete(h.clients, client)
			delete(h.clientNames, client.name)
			pe := newPlayerEventDisconnect(client.name)
			h.broadcast <- pe.toBytes()
			client.Close()
		case <-ctx.Done():
			fmt.Println("stopping Hub goroutine")
			return
		}
	}
}

// func (h *Hub) Close() {
// 	for client := range h.allClients {
// 		close(client.send)
// 	}
// 	clear(h.allClients)
// 	clear(h.clients)
// 	clear(h.clientNames)

// 	close(h.broadcast)
// 	// if _, ok := <-h.close; ok {
// 	close(h.close)
// 	// }
// 	// if _, ok := <-h.gameEvents; ok {
// 	close(h.gameEvents)
// 	// }
// 	// if _, ok := <-h.register; ok {
// 	close(h.register)
// 	// }
// 	// if _, ok := <-h.unregister; ok {
// 	close(h.unregister)
// 	// }
// }
