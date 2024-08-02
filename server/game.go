package server

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

const (
	gameStateWaiting   = "waiting"
	gameStateCountdown = "countdown"
	gameStateQuestion  = "question"
	gameStateEnded     = "ended"
)

type Game struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	PlayerCount   int       `json:"player_count"`
	QuestionCount int       `json:"question_count"`
	State         string    `json:"state"`
}

type GameCreatePayload struct {
	Name          string `json:"name"`
	QuestionCount int    `json:"question_count"`
}

func createGame(payload json.RawMessage) (*Game, error) {
	var p GameCreatePayload
	err := json.Unmarshal(payload, &p)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling create command payload: %+v. Error: %s", payload, err)
	}

	return &Game{
		ID:            uuid.New(),
		Name:          p.Name,
		PlayerCount:   0,
		QuestionCount: p.QuestionCount,
		State:         gameStateWaiting,
	}, nil
}
