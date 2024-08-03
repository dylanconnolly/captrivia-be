package server

import (
	"encoding/json"
	"net/http"
	"time"
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
	var httpGames []HttpGameResp
	for _, g := range games {
		httpGames = append(httpGames, g.httpResp())
	}
	writeJSON(w, http.StatusOK, httpGames)
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
