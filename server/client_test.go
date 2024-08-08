package server_test

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/dylanconnolly/captrivia-be/server"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

const (
	playerName    = "test player"
	gameName      = "test game"
	questionCount = 4
)

type MockGameService struct{}

func (s MockGameService) GetGames() ([]captrivia.RepositoryGame, error) {
	game := captrivia.RepositoryGame{
		ID:            uuid.New(),
		Name:          "test game",
		PlayerCount:   1,
		QuestionCount: 3,
		State:         captrivia.GameStateWaiting,
	}

	return []captrivia.RepositoryGame{game}, nil
}

func (s MockGameService) SaveGame(g *captrivia.Game) error {
	return nil
}

func (s MockGameService) ExpireGame(g uuid.UUID) error {
	return nil
}

func buildEvent(resp []byte, v server.EventPayload) server.GameEvent {
	var event server.GameEvent
	event.Payload = v
	json.Unmarshal(resp, &event)
	return event
}

func openWebsocketConn(t *testing.T) (*websocket.Conn, *httptest.Server, server.Client) {
	hub := server.NewHub(MockGameService{}, 1, 1)
	ctx, _ := context.WithCancel(context.Background())
	go hub.Run(ctx)

	client := server.NewClient(playerName, hub)

	s := httptest.NewServer(http.HandlerFunc(client.ServeWebsocket))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")

	header := http.Header{}
	header.Add("Origin", "http://localhost:3000")

	ws, _, err := websocket.DefaultDialer.Dial(u, header)
	if err != nil {
		t.Fatalf("error dialing websocket: %s", err)
	}

	// ignore player_connected message
	ws.ReadMessage()

	return ws, s, *client
}

func Raw(payload any) json.RawMessage {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(bytes)
	return raw
}

func toBytes(command server.PlayerCommand) []byte {
	bytes, _ := json.Marshal(command)
	return bytes
}

func Setup(t *testing.T) (uuid.UUID, *websocket.Conn, *server.Client, context.CancelFunc) {

	hub := server.NewHub(MockGameService{}, 1, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)
	gh, err := hub.NewGameHub(gameName, questionCount)
	if err != nil {
		log.Println(err)
	}

	client := server.NewClient(playerName, hub)

	s := httptest.NewServer(http.HandlerFunc(client.ServeWebsocket))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")

	header := http.Header{}
	header.Add("Origin", "http://localhost:3000")

	ws, _, err := websocket.DefaultDialer.Dial(u, header)
	if err != nil {
		t.Fatalf("error dialing websocket: %s", err)
	}

	// ignore player_connected message
	ws.ReadMessage()

	go gh.Run(ctx)

	return gh.ID, ws, client, cancel
}

func TestServeWebsocket(t *testing.T) {
	hub := server.NewHub(MockGameService{}, 1, 1)
	ctx, _ := context.WithCancel(context.Background())
	go hub.Run(ctx)

	client := server.NewClient(playerName, hub)

	s := httptest.NewServer(http.HandlerFunc(client.ServeWebsocket))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")

	header := http.Header{}
	header.Add("Origin", "http://localhost:3000")

	ws, _, err := websocket.DefaultDialer.Dial(u, header)
	if err != nil {
		t.Fatalf("error dialing websocket: %s", err)
	}
	defer ws.Close()

	// move to test parsing messaging test
	if err := ws.WriteMessage(websocket.TextMessage, []byte("HELLO TEST")); err != nil {
		t.Fatalf("error writing to websocket: %s", err)
	}

	expected := server.PlayerEvent{
		Payload: server.EmptyPayload{}.Raw(),
		Player:  playerName,
		Type:    server.PlayerEventTypeConnect,
	}

	expJSON, _ := json.Marshal(expected)

	_, resp, _ := ws.ReadMessage()

	assert.Equal(t, string(expJSON), string(resp))
	client.Close()
}

func TestPlayerCommandCreate(t *testing.T) {
	gameID, ws, client, cancel := Setup(t)
	// ws, _, _ := Setup(t)

	command := server.PlayerCommand{
		Nonce: "123456",
		Payload: Raw(server.PlayerCommandCreate{
			Name:          gameName,
			QuestionCount: questionCount,
		}),
		Type: server.PlayerCommandTypeCreate,
	}

	b := toBytes(command)

	ws.WriteMessage(websocket.TextMessage, b)

	expected := server.GameEvent{
		ID: gameID,
		Payload: server.GameEventCreate{
			Name:          gameName,
			QuestionCount: questionCount,
		}.Raw(),
		Type: server.GameEventTypeCreate,
	}

	_, r, _ := ws.ReadMessage()

	resp := buildEvent(r, server.GameEventCreate{}.Raw())

	assert.Equal(t, expected.Payload, resp.Payload)
	assert.Equal(t, expected.Type, expected.Type)

	client.Close()
	cancel()
}

func TestPlayerCommandJoin(t *testing.T) {
	gameID, ws, client, cancel := Setup(t)

	command := server.PlayerCommand{
		Nonce: "123456",
		Payload: Raw(server.PlayerLobbyCommand{
			GameID: gameID,
		}),
		Type: server.PlayerCommandTypeJoin,
	}

	b := toBytes(command)

	ws.WriteMessage(websocket.TextMessage, b)

	pmap := make(map[string]bool)

	pmap[playerName] = false

	expected := server.GameEvent{
		ID: gameID,
		Payload: server.GameEventPlayerEnter{
			Name:          gameName,
			Players:       []string{playerName},
			PlayersReady:  pmap,
			QuestionCount: questionCount,
		}.Raw(),
		Type: server.GameEventTypePlayerEnter,
	}

	_, r, _ := ws.ReadMessage()

	resp := buildEvent(r, server.GameEventCreate{}.Raw())

	assert.Equal(t, expected.Payload, resp.Payload)
	assert.Equal(t, expected.Type, expected.Type)

	expected = server.GameEvent{
		ID: gameID,
		Payload: server.GameEventPlayerLobbyAction{
			Player: playerName,
		}.Raw(),
		Type: server.GameEventTypePlayerJoin,
	}
	_, r, _ = ws.ReadMessage()

	log.Println(string(r))

	resp = buildEvent(r, server.GameEventCreate{}.Raw())

	assert.Equal(t, expected.Payload, resp.Payload)
	assert.Equal(t, expected.Type, expected.Type)

	client.Close()
	cancel()
}

func TestPlayerCommandReady(t *testing.T) {
	gameID, ws, client, cancel := Setup(t)

	command := server.PlayerCommand{
		Nonce: "123456",
		Payload: Raw(server.PlayerLobbyCommand{
			GameID: gameID,
		}),
		Type: server.PlayerCommandTypeReady,
	}

	b := toBytes(command)

	ws.WriteMessage(websocket.TextMessage, b)

	expected := server.GameEvent{
		ID: gameID,
		Payload: server.GameEventPlayerLobbyAction{
			Player: playerName,
		}.Raw(),
		Type: server.GameEventTypePlayerReady,
	}
	_, r, _ := ws.ReadMessage()

	log.Println(string(r))

	resp := buildEvent(r, server.GameEventCreate{}.Raw())

	assert.Equal(t, expected.Payload, resp.Payload)
	assert.Equal(t, expected.Type, expected.Type)

	client.Close()
	cancel()
}
