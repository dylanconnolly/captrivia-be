package captrivia

import (
	"github.com/google/uuid"
)

const (
	GameStateWaiting   = "waiting"
	GameStateCountdown = "countdown"
	GameStateQuestion  = "question"
	GameStateEnded     = "ended"
)

type Game struct {
	ID            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	Players       []string        `json:"players"`
	PlayersReady  map[string]bool `json:"players_ready"`
	PlayerCount   int             `json:"player_count"`
	QuestionCount int             `json:"question_count"`
	State         string          `json:"state"`
}

func NewGame(id uuid.UUID, name string, players []string, pReady map[string]bool, qCount int, state string) *Game {
	return &Game{
		ID:            id,
		Name:          name,
		Players:       players,
		PlayersReady:  pReady,
		PlayerCount:   len(players),
		QuestionCount: qCount,
		State:         state,
	}
}
