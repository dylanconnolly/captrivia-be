package server

import (
	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/google/uuid"
)

type Game struct {
	ID            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	Players       []string        `json:"players"`
	PlayersReady  map[string]bool `json:"players_ready"`
	QuestionCount int             `json:"question_count"`
	State         string          `json:"state"`
}

type HttpGameResp struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	PlayerCount   int       `json:"player_count"`
	QuestionCount int       `json:"question_count"`
	State         string    `json:"state"`
}

func GameToHTTPResp(g captrivia.Game) HttpGameResp {
	return HttpGameResp{
		g.ID,
		g.Name,
		g.PlayerCount,
		g.QuestionCount,
		g.State,
	}
}
