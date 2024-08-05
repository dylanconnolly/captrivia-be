package server

import (
	"github.com/dylanconnolly/captrivia-be/redis"
	"github.com/google/uuid"
)

var (
	games []*Game = []*Game{{uuid.New(), "Game 1", []string{"Player 1", "Player 2"}, make(map[string]bool), 3, 5, "countdown"}, {uuid.New(), "John's Game", []string{}, make(map[string]bool), 3, 5, "waiting"}}
)

const (
	gameStateWaiting   = "waiting"
	gameStateCountdown = "countdown"
	gameStateQuestion  = "question"
	gameStateEnded     = "ended"
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

func newGame(name string, qCount int) Game {
	return Game{
		ID:            uuid.New(),
		Name:          name,
		PlayersReady:  make(map[string]bool),
		PlayerCount:   0,
		QuestionCount: qCount,
		State:         gameStateWaiting,
	}
}

type HttpGameResp struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	PlayerCount   int       `json:"player_count"`
	QuestionCount int       `json:"question_count"`
	State         string    `json:"state"`
}

func GameToHTTPResp(g redis.Game) HttpGameResp {
	return HttpGameResp{
		g.ID,
		g.Name,
		g.PlayerCount,
		g.QuestionCount,
		g.State,
	}
}
