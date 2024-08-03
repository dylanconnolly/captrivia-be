package server

import (
	"encoding/json"
	"log"
	"slices"

	"github.com/google/uuid"
)

type GameEventType string

const (
	GameEventTypeCreate      = "game_create"
	GameEventTypePlayerJoin  = "game_player_join"
	GameEventTypePlayerEnter = "game_player_enter"
)

type EventPayload interface {
	json.Marshaler
}

type GameEvent struct {
	ID      uuid.UUID     `json:"id"`
	Payload EventPayload  `json:"payload"`
	Type    GameEventType `json:"type"`
}

// Payload to be sent to client when a new game is created
type GameEventCreate struct {
	Name          string `json:"name"`
	QuestionCount int    `json:"question_count"`
}

func (e GameEventCreate) Raw() *json.RawMessage {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(bytes)
	return &raw
}

type GameEventPlayerJoin struct {
	Player string `json:"player"`
}

func (e GameEventPlayerJoin) Raw() *json.RawMessage {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(bytes)
	return &raw
}

type GameEventPlayerEnter struct {
	Name          string          `json:"name"`
	Players       []string        `json:"players"`
	PlayersReady  map[string]bool `json:"players_ready"`
	QuestionCount int             `json:"question_count"`
}

func (e GameEventPlayerEnter) Raw() *json.RawMessage {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(bytes)
	return &raw
}

func newGameEventCreate(g Game) *GameEvent {
	payload := GameEventCreate{
		g.Name,
		g.QuestionCount,
	}
	ge := &GameEvent{
		g.ID,
		payload.Raw(),
		GameEventTypeCreate,
	}

	return ge
}

func newGameEventPlayerEnter(gameID uuid.UUID, player string) *GameEvent {
	i := slices.IndexFunc(games, func(g *Game) bool { return g.ID == gameID })
	game := games[i]
	log.Printf("game: %+v", game)
	game.Players = append(game.Players, player)
	game.PlayerCount = game.PlayerCount + 1
	game.PlayersReady[player] = false
	payload := GameEventPlayerEnter{
		player,
		game.Players,
		game.PlayersReady,
		game.QuestionCount,
	}

	ge := &GameEvent{
		gameID,
		payload.Raw(),
		GameEventTypePlayerEnter,
	}

	return ge
}

func newGameEventPlayerJoin(gameID uuid.UUID, player string) *GameEvent {
	payload := GameEventPlayerJoin{
		player,
	}

	ge := &GameEvent{
		gameID,
		payload.Raw(),
		GameEventTypePlayerJoin,
	}

	return ge
}
