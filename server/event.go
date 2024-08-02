package server

import (
	"encoding/json"

	"github.com/google/uuid"
)

const (
	GameEventTypeCreate      = "game_create"
	GameEventTypePlayerJoin  = "game_player_join"
	GameEventTypePlayerEnter = "game_player_enter"
)

type GameEvent struct {
	ID      uuid.UUID        `json:"id"`
	Payload *json.RawMessage `json:"payload"`
	Type    string           `json:"type"`
}

type GameEventPlayerJoin struct {
	Player string `json:"player"`
}

type GameEventPlayerEnter struct {
	Name          string          `json:"name"`
	Players       []string        `json:"players"`
	PlayersReady  map[string]bool `json:"players_ready"`
	QuestionCount int             `json:"question_count"`
}

func structToEventPayload(s any) (*json.RawMessage, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	raw := json.RawMessage(b)
	return &raw, nil
}
