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

func TestGameHub_Concurrency(t *testing.T) {
	// Mock or create the dependencies
	gameID := uuid.New()
	game := &captrivia.Game{
		ID:            uuid.New(),
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
	client.Conn = &MockWebSocketConn{} // Mock WebSocket connection

	// Register the client
	gameHub.Register <- client

	// Simulate concurrent client messages
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

	go func() {
		for i := 0; i < 100; i++ {
			gameHub.Broadcast <- []byte("test message")
			time.Sleep(15 * time.Millisecond)
		}
	}()

	go func() {
		for i := 0; i < 100; i++ {
			gameHub.Unregister <- client
			time.Sleep(10 * time.Millisecond)
			gameHub.Register <- client
			time.Sleep(10 * time.Millisecond)
		}
	}()

	time.Sleep(5 * time.Second)

	// Verify the state
	time.Sleep(2 * time.Second)
	assert.Contains(t, gameHub.Clients, client)
}
