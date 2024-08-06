package server

import (
	"sync"

	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/google/uuid"
)

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

type Game struct {
	id                   uuid.UUID
	name                 string
	State                string
	Mutex                sync.Mutex
	Answered             bool
	Questions            []captrivia.Question
	CurrentQuestionIndex int
	Scores               map[string]int
	PlayerCount          int
	Players              []string
}

// func startGame(rdb *redis.Client) {
// 	gameID := "game123"
// 	gameKey := fmt.Sprintf("game:%s", gameID)
// 	questionsKey := fmt.Sprintf("%s:questions", gameKey)

// 	questions, err := getQuestionsForGame(rdb, questionsKey)
// 	if err != nil {
// 		log.Fatal("Error getting questions:", err)
// 	}

// 	ticker := time.NewTicker(10 * time.Second)
// 	defer ticker.Stop()

// 	for range ticker.C {
// 		question := getNextQuestion(questions)
// 		if question == nil {
// 			break
// 		}
// 		broadcast <- *question
// 	}
// }
