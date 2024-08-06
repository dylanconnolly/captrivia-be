package server

import (
	"encoding/json"

	"github.com/google/uuid"
)

type PlayerCommandType string

const (
	PlayerCommandTypeCreate PlayerCommandType = "create"
	PlayerCommandTypeJoin   PlayerCommandType = "join"
	PlayerCommandTypeReady  PlayerCommandType = "ready"
	PlayerCommandTypeStart  PlayerCommandType = "start"
	PlayerCommandTypeAnswer PlayerCommandType = "answer"
)

type PlayerCommand struct {
	Nonce   string            `json:"nonce"`
	Payload json.RawMessage   `json:"payload"`
	Type    PlayerCommandType `json:"type"`
}

type GameLobbyCommand struct {
	player  string
	payload PlayerLobbyCommand
	Type    PlayerCommandType
}

type PlayerCommandCreate struct {
	Name          string `json:"name"`
	QuestionCount int    `json:"question_count"`
}

type PlayerLobbyCommand struct {
	GameID uuid.UUID `json:"game_id"`
}

type PlayerCommandAnswer struct {
	GameID     uuid.UUID `json:"game_id"`
	Index      int       `json:"index"`
	QuestionID string    `json:"question_id"`
}
