package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

var (
	games = []struct {
		ID            uuid.UUID `json:"id"`
		Name          string    `json:"name"`
		PlayerCount   int       `json:"player_count"`
		QuestionCount int       `json:"question_count"`
		State         string    `json:"state"`
	}{
		{uuid.New(), "Game 1", 3, 5, "countdown"},
		{uuid.New(), "John's Game", 1, 3, "waiting"},
		{uuid.New(), "Unnamed Game", 0, 6, "ended"},
	}
)

type GameServer struct {
	cm *ClientManager
}

func NewGameServer(cm *ClientManager) *GameServer {
	return &GameServer{cm: cm}
}

// Games writes the existing games to the response.
func (g *GameServer) Games(w http.ResponseWriter, r *http.Request) {
	// TODO: Fix this data so it is not hardcoded, and is the right shape
	// that the frontend expects
	writeJSON(w, http.StatusOK, games)
}

func (g *GameServer) Connect(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		return
	}

	c := newClient(name, g.cm)
	c.serveWebsocket(w, r)
}

func (g *GameServer) Leaderboard(w http.ResponseWriter, r *http.Request) {
	t, err := time.Parse(time.RFC3339, "2022-01-01T12:00:00Z")
	if err != nil {
		return
	}
	leaderboard := []struct {
		PlayerName          string    `json:"player_name"`
		Accuracy            float32   `json:"accuracy"`
		AverageMilliseconds int       `json:"average_milliseconds"`
		CorrectQuestions    string    `json:"correct_questions"`
		TimeMilliseconds    string    `json:"time_milliseconds"`
		TotalQuestions      string    `json:"total_questions"`
		LastUpdate          time.Time `json:"last_update"`
	}{
		{"John Doe", 0.75, 1500, "10", "5000", "15", t},
	}

	writeJSON(w, http.StatusOK, leaderboard)
}

func writeJSON(w http.ResponseWriter, statusCode int, obj any) error {
	b, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	// TODO: Handle this error
	_, err = w.Write(b)
	return err
}
