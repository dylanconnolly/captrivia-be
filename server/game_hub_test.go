package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/dylanconnolly/captrivia-be/server"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// MockWebSocketConn is a mock implementation of a WebSocket connection
type MockWebSocketConn struct{}

func (m *MockWebSocketConn) ReadMessage() (int, []byte, error) {
	return websocket.TextMessage, []byte(""), nil
}

func (m *MockWebSocketConn) WriteMessage(messageType int, data []byte) error {
	return nil
}

func (m *MockWebSocketConn) Close() error {
	return nil
}

func TestGameHubRegisterClient(t *testing.T) {
	// Mock or create the dependencies
	gameID := uuid.New()
	game := &captrivia.Game{
		ID:            gameID,
		Name:          "test game",
		PlayersReady:  make(map[string]bool),
		QuestionCount: 3,
		State:         captrivia.GameStateWaiting,
		Scores:        make(map[string]int),
	}
	gameService := &MockGameService{}
	hubBroadcast := make(chan server.GameEvent, 10)
	countdownSec := 5
	questionSec := 10

	hub := server.NewHub(gameService, 5, 5)
	gameHub := server.NewGameHub(game, gameService, hubBroadcast, countdownSec, questionSec)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go gameHub.Run(ctx)

	client := server.NewClient("test_client", hub)
	client.Conn = &MockWebSocketConn{}

	gameHub.Register <- client

	go func() {
		for i := 0; i < 100; i++ {
			gameHub.Commands <- server.GameLobbyCommand{
				Type:    server.PlayerCommandTypeReady,
				Player:  "test_client",
				Payload: server.PlayerLobbyCommand{GameID: gameID},
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	time.Sleep(1 * time.Second)
	assert.Contains(t, gameHub.Clients, client)
	assert.Equal(t, 1, len(gameHub.Clients))
}

func TestGameHubBroadcast(t *testing.T) {
	// Mock or create the dependencies
	gameID := uuid.New()
	game := &captrivia.Game{
		ID:            gameID,
		Name:          "test game",
		PlayersReady:  make(map[string]bool),
		QuestionCount: 3,
		State:         captrivia.GameStateWaiting,
		Scores:        make(map[string]int),
	}
	gameService := &MockGameService{}
	hubBroadcast := make(chan server.GameEvent, 10)
	countdownSec := 5
	questionSec := 10

	hub := server.NewHub(gameService, 5, 5)
	gameHub := server.NewGameHub(game, gameService, hubBroadcast, countdownSec, questionSec)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go gameHub.Run(ctx)

	client := server.NewClient("test_client", hub)
	client.Conn = &MockWebSocketConn{}

	client2 := server.NewClient("test_client 2", hub)
	client.Conn = &MockWebSocketConn{}

	client3 := server.NewClient("test_client 3", hub)
	client.Conn = &MockWebSocketConn{}

	gameHub.Register <- client
	gameHub.Register <- client2

	go func() {
		for i := 0; i < 100; i++ {
			gameHub.Broadcast <- []byte("test message")
			time.Sleep(10 * time.Millisecond)
		}
	}()

	time.Sleep(2 * time.Second)

	// should have 100+ due to other messages from client joins/enter
	assert.GreaterOrEqual(t, len(client.Send), 100)
	assert.GreaterOrEqual(t, len(client.Send), 100)
	assert.Equal(t, 0, len(client3.Send))
}

func TestGameHubUnregisterFailure(t *testing.T) {
	// Mock or create the dependencies
	gameID := uuid.New()
	game := &captrivia.Game{
		ID:            gameID,
		Name:          "test game",
		PlayersReady:  make(map[string]bool),
		QuestionCount: 3,
		State:         captrivia.GameStateWaiting,
		Scores:        make(map[string]int),
	}
	gameService := &MockGameService{}
	hubBroadcast := make(chan server.GameEvent, 10)
	countdownSec := 5
	questionSec := 10

	hub := server.NewHub(gameService, 5, 5)
	gameHub := server.NewGameHub(game, gameService, hubBroadcast, countdownSec, questionSec)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go gameHub.Run(ctx)

	client := server.NewClient("test_client", hub)
	client.Conn = &MockWebSocketConn{}

	gameHub.Unregister <- client

	time.Sleep(1 * time.Second)
	assert.Equal(t, 0, game.PlayerCount)
}
