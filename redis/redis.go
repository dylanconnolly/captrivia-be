package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	gameEndExpiry       time.Duration = (time.Duration(5) * time.Minute)
	gameKey             string        = "game:%s"
	gamePlayersKey      string        = "game:%s:players"
	gamePlayersReadyKey string        = "game:%s:players:ready"
	playerGamesKey      string        = "player:%s:games"
	gameQuestionsKey    string        = "game:%s:questions"
	questionKey         string        = "question:%s"
	questionOptionsKey  string        = "question:%s:options"
	allQuestionsKey     string        = "questions"
)

var ctx = context.Background()

type Question struct {
}

func NewClient() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		// Addr:     "redis:6379" // for docker container
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	return rdb
}

// func (db *DB) AddPlayerToGame(id uuid.UUID, player string) error {
// 	pKey := fmt.Sprintf(gamePlayersKey, id)
// 	pReadyKey := fmt.Sprintf(gamePlayersReadyKey, id)

// 	err := db.client.SAdd(ctx, pKey, player).Err()
// 	if err != nil {
// 		return err
// 	}

// 	err = db.client.HSet(ctx, pReadyKey, player, false).Err()
// 	if err != nil {
// 		return err
// 	}

// 	err = db.addPlayerToGame(player, id)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (db *DB) PlayerReady(id uuid.UUID, player string) error {
// 	key := fmt.Sprintf(gamePlayersReadyKey, id)

// 	err := db.client.HSet(ctx, key, player, true).Err()
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (db *DB) RemovePlayerFromGame(id uuid.UUID, player string) error {
// 	gamePlayersKey := fmt.Sprintf(gamePlayersKey, id)
// 	gamePlayersReadyKey := fmt.Sprintf(gamePlayersReadyKey, player)

// 	err := db.client.SRem(ctx, gamePlayersKey, player).Err()
// 	if err != nil {
// 		return err
// 	}

// 	err = db.client.HDel(ctx, gamePlayersReadyKey, player).Err()
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (db *DB) RemovePlayerFromCreatedGames(player string) error {
// 	gameIDs, err := db.getPlayerGames(player)
// 	if err != nil {
// 		return err
// 	}
// 	for _, g := range gameIDs {
// 		id, _ := uuid.Parse(g)
// 		err = db.RemovePlayerFromGame(id, player)
// 		db.client.SRem(ctx, playerGamesKey, id)
// 		// need to expire the game and remove it
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (db *DB) LoadQuestionsFromFile(filename string) error {
// 	questions, err := captrivia.LoadQuestions(filename)
// 	if err != nil {
// 		return err
// 	}

// 	for _, q := range questions {
// 		questionKey := fmt.Sprintf(questionKey, q.ID)
// 		err := db.client.HSet(ctx, questionKey, map[string]interface{}{
// 			"id":            q.ID,
// 			"question_text": q.QuestionText,
// 			"correct_index": q.CorrectIndex,
// 		}).Err()
// 		if err != nil {
// 			return err
// 		}

// 		// set to track all question IDs and randomly choose questions for games
// 		db.client.SAdd(ctx, allQuestionsKey, q.ID)

// 		optionsKey := fmt.Sprintf(questionOptionsKey, q.ID)

// 		for _, option := range q.Options {
// 			err := db.client.RPush(ctx, optionsKey, option).Err()
// 			if err != nil {
// 				return err
// 			}
// 			// lazily trimming options list to be size of JSON question options after each startup
// 			db.client.LTrim(ctx, optionsKey, 0, int64(len(q.Options)-1))
// 		}
// 	}
// 	return nil
// }
