package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/dylanconnolly/captrivia-be/redis"
	"github.com/google/uuid"
)

type GameServer struct {
	hub *Hub
}

func NewGameServer(hub *Hub) *GameServer {
	return &GameServer{hub: hub}
}

type HttpGameResp struct {
	ID            uuid.UUID           `json:"id"`
	Name          string              `json:"name"`
	PlayerCount   int                 `json:"player_count"`
	QuestionCount int                 `json:"question_count"`
	State         captrivia.GameState `json:"state"`
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

// Games writes the existing games to the response.
func (g *GameServer) Games(w http.ResponseWriter, r *http.Request) {
	// TODO: Fix this data so it is not hardcoded, and is the right shape
	// that the frontend expects
	var httpGames []HttpGameResp

	games, err := g.hub.db.GetAllGames()
	if err != nil {
		log.Println(err)
		writeJSON(w, http.StatusInternalServerError, httpGames)
		return
	}
	for _, g := range games {
		httpGames = append(httpGames, GameToHTTPResp(g))
	}
	if len(games) < 1 {
		writeJSON(w, http.StatusNoContent, httpGames)
	}
	writeJSON(w, http.StatusOK, httpGames)
}

func (g *GameServer) Connect(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// fail if user already exists with name
	if _, ok := g.hub.clientNames[name]; ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	c := newClient(name, g.hub, redis.NewClient())
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
