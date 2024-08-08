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
	GameService  captrivia.GameService
	gameHubs     map[uuid.UUID]*GameHub
	hubBroadcast chan GameEvent // used to broadcast GameEvents to clients not in games (GameCreate, GameStateChange, GamePlayerCountChange)
	CountdownSec int
	QuestionSec  int
}

func NewHub(gs captrivia.GameService, countdownSec int, questionSec int) *Hub {
	return &Hub{
		allBroadcast: make(chan []byte, 20),
		clients:      make(map[*Client]bool),
		clientNames:  make(map[string]bool),
		disconnect:   make(chan *Client),
		hubClients:   make(map[*Client]bool),
		register:     make(chan *Client, 10),
		unregister:   make(chan *Client, 10),

		GameService:  gs,
		gameHubs:     make(map[uuid.UUID]*GameHub),
		hubBroadcast: make(chan GameEvent, 15),
		CountdownSec: countdownSec,
		QuestionSec:  questionSec,
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
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		// send game state changes to clients which are not actively in a game
		case event := <-h.hubBroadcast:
			for client := range h.hubClients {
				select {
				case client.send <- event.toBytes():
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		case client := <-h.disconnect:
			if client.gameHub != nil {
				client.gameHub.playerLeave(client)
			}
			delete(h.clients, client)
			delete(h.hubClients, client)
			delete(h.clientNames, client.name)
			close(client.send)
		case <-ctx.Done():
			log.Println("stopping Hub goroutine.")
			return
		}
	}
}

func (h *Hub) NewGameHub(name string, questionCount int) (*GameHub, error) {
	game, err := captrivia.NewGame(name, questionCount)
	if err != nil {
		return nil, fmt.Errorf("error creating game for game hub: %s", err)
	}
	gh := NewGameHub(game, h.GameService, h.hubBroadcast, h.CountdownSec, h.QuestionSec)
	h.gameHubs[gh.ID] = gh

	ge := newGameEventCreate(game.ID, game.Name, game.QuestionCount)
	h.hubBroadcast <- ge
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
