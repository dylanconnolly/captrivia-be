package player

import "encoding/json"

const (
	PlayerCommandTypeCreate = "create"
	PlayerCommandTypeJoin   = "join"
	PlayerCommandTypeReady  = "ready"
	PlayerCommandTypeStart  = "start"
	PlayerCommandTypeAnswer = "answer"
)

type PlayerCommand struct {
	Nonce   string          `json:"nonce"`
	Payload json.RawMessage `json:"payload"`
	Type    string          `json:"type"`
}

type CreatePayload struct {
	Name          string `json:"name"`
	QuestionCount int    `json:"question_count"`
}
