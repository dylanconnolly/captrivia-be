package server

import (
	"context"
	"fmt"
	"log"

	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/google/uuid"
)

// Hub is the top level struct tracking all active clients.
// It is responsible for
type Hub struct {
	// client fields
	allBroadcast chan []byte      // broadcast messages to all clients
	clients      map[*Client]bool // tracks all active clients
	clientNames  map[string]bool
	disconnect   chan *Client
	hubClients   map[*Client]bool // tracks only clients that are in the hub (not in a game)
	register     chan *Client
	unregister   chan *Client

	// game fields
	GameService captrivia.GameService
	gameHubs    map[uuid.UUID]*GameHub
	gameEvents  chan GameEvent // used to broadcast GameEvents to clients not in games (GameCreate, GameStateChange, GamePlayerCountChange)
}

func NewHub(gs captrivia.GameService) *Hub {
	return &Hub{
		allBroadcast: make(chan []byte, 20), //TODO unbuffered with goroutines?
		clients:      make(map[*Client]bool),
		clientNames:  make(map[string]bool),
		disconnect:   make(chan *Client),
		hubClients:   make(map[*Client]bool),
		register:     make(chan *Client, 10), //TODO unbuffered with goroutines?
		unregister:   make(chan *Client, 10), //TODO unbuffered with goroutines?

		gameEvents:  make(chan GameEvent, 15), //TODO unbuffered with goroutines?
		gameHubs:    make(map[uuid.UUID]*GameHub),
		GameService: gs,
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.register:
			h.hubClients[client] = true
			h.clients[client] = true
			h.clientNames[client.name] = true
		case client := <-h.unregister:
			// Unregister removes client from hubClients so they will not receieve GameEvent updates while in a game
			delete(h.hubClients, client)
		case message := <-h.allBroadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
					log.Printf("all broadcast message: %s", message)
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		// send game state changes to clients which are not actively in a game
		case event := <-h.gameEvents:
			for client := range h.hubClients {
				select {
				case client.send <- event.toBytes():
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		case client := <-h.disconnect:
			log.Printf("CLIENT IN DISCONNECT: %+v", client)
			if client.gameHub != nil {
				client.gameHub.unregister <- client
			}
			delete(h.clients, client)
			delete(h.hubClients, client)
			delete(h.clientNames, client.name)
			if _, ok := <-client.send; ok {
				close(client.send)
			}
			log.Printf("CLIENT AT END OF DISCONNECT: %+v", client)
		case <-ctx.Done():
			log.Println("stopping Hub goroutine")
			return
		}
	}
}

func (h *Hub) NewGameHub(name string, questionCount int) (*GameHub, error) {
	game, err := captrivia.NewGame(name, questionCount)
	if err != nil {
		return nil, fmt.Errorf("error creating game for game hub: %s", err)
	}
	gh := NewGameHub(game, h.GameService, h.gameEvents)
	h.gameHubs[gh.ID] = gh

	ge := newGameEventCreate(game.ID, game.Name, game.QuestionCount)
	h.gameEvents <- ge
	log.Printf("got create command")
	return gh, nil
}

// func (h *Hub) RunGameHub(gameID uuid.UUID) {
// 	if gh := h.GetGameHub(gameID); gh != nil {
// 		ctx := context.Background()
// 		go gh.Run(ctx)
// 	}
// }

func (h *Hub) GetGameHub(gameID uuid.UUID) (*GameHub, error) {
	if gh, ok := h.gameHubs[gameID]; ok {
		return gh, nil
	}
	return nil, fmt.Errorf("no gamehub found for gameID=%s", gameID)
}

func (h *Hub) CloseGameHub(gameID uuid.UUID) {
	if gh, ok := h.gameHubs[gameID]; ok {
		for client := range gh.clients {
			gh.unregister <- client
			h.register <- client
		}
	}
}
