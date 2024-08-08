package captrivia

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
)

type GameState string

const (
	GameStateWaiting   GameState = "waiting"
	GameStateCountdown GameState = "countdown"
	GameStateQuestion  GameState = "question"
	GameStateEnded     GameState = "ended"
)

type Game struct {
	ID            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	PlayersReady  map[string]bool `json:"players_ready"`
	PlayerCount   int             `json:"player_count"`
	QuestionCount int             `json:"question_count"`
	State         GameState       `json:"state"`

	currentQuestionIndex int
	questions            []Question
	scores               map[string]int
	gameEnded            chan bool
	// mu                   sync.Mutex
}

type PlayerScore struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type GameService interface {
	SaveGame(g *Game) error
	GetGames() ([]RepositoryGame, error)
	ExpireGame(id uuid.UUID) error
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

type RepositoryGame struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	PlayerCount   int       `json:"player_count"`
	QuestionCount int       `json:"question_count"`
	State         GameState `json:"state"`
}

func (g Game) ToRepositoryGame() RepositoryGame {
	return RepositoryGame{
		ID:            g.ID,
		Name:          g.Name,
		PlayerCount:   g.PlayerCount,
		QuestionCount: g.QuestionCount,
		State:         g.State,
	}
}

func (g RepositoryGame) ToHash() map[string]string {
	return map[string]string{
		"id":             g.ID.String(),
		"name":           g.Name,
		"player_count":   strconv.Itoa(g.PlayerCount),
		"question_count": strconv.Itoa(g.QuestionCount),
		"state":          string(g.State),
	}
}

func newGame(name string, qCount int) *Game {
	return &Game{
		ID:            uuid.New(),
		Name:          name,
		PlayersReady:  make(map[string]bool),
		QuestionCount: qCount,
		State:         GameStateWaiting,
		scores:        make(map[string]int),
		gameEnded:     make(chan bool, 1),
	}
}

func NewGame(name string, qCount int) (*Game, error) {
	game := newGame(name, qCount)

	file, err := filepath.Abs("questions.json")
	if err != nil {
		return nil, fmt.Errorf("error forming filepath for questions: %s", err)
	}
	questions, err := LoadQuestions(file)
	if err != nil {
		return nil, fmt.Errorf("error loading questions: %s", err)
	}

	shuffled := ShuffleQuestions(questions, qCount)
	game.questions = shuffled

	return game, nil
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

func (g *Game) PlayerReady(player string) {
	g.PlayersReady[player] = true
}

func (g *Game) AddQuestions(questions []Question) {
	g.questions = append(g.questions, questions...)
}

func (g *Game) PlayerScores() []PlayerScore {
	var playerScores []PlayerScore
	for player, score := range g.scores {
		s := PlayerScore{
			Name:  player,
			Score: score,
		}
		playerScores = append(playerScores, s)
	}

	return playerScores
}

func (g *Game) CurrentIndex() int {
	return g.currentQuestionIndex
}

func (g *Game) CurrentQuestion() Question {
	return g.questions[g.currentQuestionIndex]
}

func (g *Game) GoToNextQuestion() {
	if g.IsLastQuestion() {
		g.gameEnded <- true
		return
	}
	g.currentQuestionIndex++
}

func (g *Game) IsLastQuestion() bool {
	return g.currentQuestionIndex >= (len(g.questions) - 1)
}

func (g *Game) StartGame() {
	g.State = GameStateCountdown
}

func (g *Game) ValidateAnswer(index int) bool {
	return index == g.questions[g.currentQuestionIndex].CorrectIndex
}

func (g *Game) IncrementPlayerScore(player string) {
	g.scores[player] += 1
}

func (g *Game) GameEndedChan() chan bool {
	return g.gameEnded
}

func (g *Game) EndGame() {
	clear(g.PlayersReady)
	g.PlayerCount = 0
}
