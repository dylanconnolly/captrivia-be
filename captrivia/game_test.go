package captrivia_test

import (
	"encoding/json"
	"testing"

	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/stretchr/testify/assert"
)

const (
	gameName      = "test game"
	questionCount = 5
)

func CreateTestGame() captrivia.Game {
	return *captrivia.NewGame(gameName, questionCount)
}

func TestNewGame(t *testing.T) {
	g := captrivia.NewGame(gameName, questionCount)

	assert.Equal(t, gameName, g.Name)
	assert.Equal(t, questionCount, g.QuestionCount)
	assert.Equal(t, captrivia.GameStateWaiting, g.State)
	assert.Equal(t, 0, g.PlayerCount)
	assert.IsType(t, make(map[string]bool), g.PlayersReady)
	assert.IsType(t, make(map[string]bool), g.PlayersReady)
	assert.IsType(t, make(map[string]bool), g.PlayersReady)
}

func TestMarshalJSON(t *testing.T) {
	g := CreateTestGame()

	bytes, err := json.Marshal(g)

	if !assert.NoError(t, err) {
		t.Error("error marshalling game: ", err)
	}

	var resp map[string]interface{}
	json.Unmarshal(bytes, &resp)
	assert.Contains(t, resp, "id")
	assert.Contains(t, resp, "name")
	assert.Contains(t, resp, "players")
	assert.Contains(t, resp, "players_ready")
	assert.Contains(t, resp, "question_count")
	assert.Contains(t, resp, "state")
	assert.NotContains(t, resp, "Scores")
	assert.NotContains(t, resp, "Questions")
}

func TestPlayerNames(t *testing.T) {
	g := CreateTestGame()
	g.PlayersReady["test1"] = false
	g.PlayersReady["test2"] = false
	expected := []string{"test1", "test2"}
	players := g.PlayerNames()

	assert.Equal(t, expected, players)
}

func TestPlayerScores(t *testing.T) {
	g := CreateTestGame()

	g.AddPlayer("player 1")

	assert.Equal(t, 1, len(g.PlayerScores()))
	assert.Equal(t, "player 1", g.PlayerScores()[0].Name)
	assert.Equal(t, 0, g.PlayerScores()[0].Score)

	g.AddPlayer("player 2")
	assert.Equal(t, 2, len(g.PlayerScores()))
}

func TestAddPlayer(t *testing.T) {
	g := CreateTestGame()

	assert.Empty(t, g.PlayersReady)
	g.AddPlayer("test player")

	assert.NotEmpty(t, g.PlayersReady)
	assert.NotEmpty(t, g.PlayerScores())
	assert.Equal(t, 1, g.PlayerCount)

	g.AddPlayer("test player 2")
	assert.Equal(t, 2, g.PlayerCount)
}

func TestRemovePlayer(t *testing.T) {
	g := CreateTestGame()

	g.AddPlayer("test player")
	g.RemovePlayer("test player")

	assert.Empty(t, g.PlayersReady)

	assert.Equal(t, 0, g.PlayerCount)
	assert.Empty(t, 0, g.PlayerScores)
}

func TestIsLastQuestion(t *testing.T) {
	questions, err := captrivia.LoadQuestions("../questions.json")
	if err != nil {
		t.Error("error loading questions: ", err)
	}
	g := CreateTestGame()

	g.AddQuestions(questions)

	assert.False(t, g.IsLastQuestion())

	g.GoToNextQuestion()

	assert.False(t, g.IsLastQuestion())

	g.GoToNextQuestion()

	assert.True(t, g.IsLastQuestion())
}
