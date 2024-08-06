package redis

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PlayerGames []string

type Player struct {
	Name                string    `json:"player_name"`
	Accuracy            float32   `json:"accuracy"`
	AverageMilliseconds int       `json:"average_milliseconds"`
	CorrectQuestions    string    `json:"correct_questions"`
	TimeMilliseconds    string    `json:"time_milliseconds"`
	TotalQuestions      string    `json:"total_questions"`
	UpdatedAt           time.Time `json:"last_update"`
}

func (db *DB) addPlayerToGame(player string, gameID uuid.UUID) error {
	playerGamesKey := fmt.Sprintf(playerGamesKey, player)

	err := db.client.SAdd(ctx, playerGamesKey, gameID.String()).Err()
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) getPlayerGames(player string) (PlayerGames, error) {
	playerGamesKey := fmt.Sprintf(playerGamesKey, player)

	gameIDs, err := db.client.SMembers(ctx, playerGamesKey).Result()
	if err != nil {
		return nil, err
	}

	return gameIDs, err
}

// func (db *DB) removePlayerFromGame(player string, gameID uuid.UUID) error {
// 	playerGamesKey := fmt.Sprintf(playerGamesKey, player)

// 	err := db.client.SRem(ctx, playerGamesKey, gameID.String()).Err()
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }
