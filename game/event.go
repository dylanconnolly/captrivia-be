package game

import (
	"encoding/json"

	"github.com/google/uuid"
)

type GameEvent struct {
	ID      uuid.UUID        `json:"id"`
	Payload *json.RawMessage `json:"payload"`
	Type    string           `json:"type"`
}

type GameEventCreate struct {
	Name           string `json:"name"`
	Question_count int    `json:"question_count"`
}
