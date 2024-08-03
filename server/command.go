package server

import (
	"encoding/json"

	"github.com/google/uuid"
)

type PlayerCommandType string
type PlayerEventType string

const (
	PlayerCommandTypeCreate   PlayerCommandType = "create"
	PlayerCommandTypeJoin     PlayerCommandType = "join"
	PlayerCommandTypeReady    PlayerCommandType = "ready"
	PlayerCommandTypeStart    PlayerCommandType = "start"
	PlayerCommandTypeAnswer   PlayerCommandType = "answer"
	PlayerEventTypeConnect    PlayerEventType   = "player_connect"
	PlayerEventTypeDisconnect PlayerEventType   = "player_disconnect"
)

type PlayerCommand struct {
	Nonce   string            `json:"nonce"`
	Payload json.RawMessage   `json:"payload"`
	Type    PlayerCommandType `json:"type"`
}

type PlayerCommandCreate struct {
	Name          string `json:"name"`
	QuestionCount int    `json:"question_count"`
}

type PlayerCommandJoin struct {
	GameID uuid.UUID `json:"game_id"`
}

type PlayerCommandReady struct {
	GameID uuid.UUID `json:"game_id"`
}
