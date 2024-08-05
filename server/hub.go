package server

import "github.com/dylanconnolly/captrivia-be/redis"

// ClientManager tracks all active websocket clients and broadcasts
// messages to each client.
type Hub struct {
	db          *redis.DB
	broadcast   chan []byte
	clients     map[*Client]bool
	clientNames map[string]bool
	register    chan *Client
	unregister  chan *Client
}

func NewHub(db *redis.DB) *Hub {
	return &Hub{
		db:          db,
		broadcast:   make(chan []byte),
		clients:     make(map[*Client]bool),
		clientNames: make(map[string]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.clientNames[client.name] = true
		case client := <-h.unregister:
			delete(h.clients, client)
			delete(h.clientNames, client.name)
			h.db.RemovePlayerFromCreatedGames(client.name)
			close(client.send)
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
