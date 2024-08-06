package captrivia

import (
	"encoding/json"

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
	PlayersReady  map[string]bool `json:"players_ready"`
	PlayerCount   int             `json:"player_count"`
	QuestionCount int             `json:"question_count"`
	State         string          `json:"state"`

	questions            []Question
	currentQuestionIndex int
	scores               map[string]int
}

type PlayerScore struct {
	Player string `json:"player"`
	Score  int    `json:"score"`
}

func (g Game) MarshalJSON() ([]byte, error) {
	type Alias Game
	return json.Marshal(&struct {
		Players []string `json:"players"`
		Alias
	}{
		Players: g.PlayerNames(),
		Alias:   (Alias)(g),
	})
}

func (g Game) PlayerNames() []string {
	names := make([]string, 0, len(g.PlayersReady))
	for name := range g.PlayersReady {
		names = append(names, name)
	}

	return names
}

func NewGame(name string, qCount int) *Game {
	return &Game{
		ID:            uuid.New(),
		Name:          name,
		PlayersReady:  make(map[string]bool),
		QuestionCount: qCount,
		State:         GameStateWaiting,
		scores:        make(map[string]int),
	}
}

func (g *Game) AddPlayer(player string) {
	g.PlayersReady[player] = false
	g.scores[player] = 0
	g.PlayerCount++
}

func (g *Game) RemovePlayer(player string) {
	delete(g.PlayersReady, player)
	delete(g.scores, player)
	g.PlayerCount--
}

func (g *Game) AddQuestion(q Question) {
	g.questions = append(g.questions, q)
}

func (g *Game) PlayerScores() []PlayerScore {
	var playerScores []PlayerScore
	for player, score := range g.scores {
		s := PlayerScore{
			Player: player,
			Score:  score,
		}
		playerScores = append(playerScores, s)
	}

	return playerScores
}

func (g *Game) StartGame() {
	g.State = GameStateCountdown
}
