package redis

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	gameStateWaiting   = "waiting"
	gameStateCountdown = "countdown"
	gameStateQuestion  = "question"
	gameStateEnded     = "ended"

	gameKey             string = "game:%s"
	gamePlayersKey      string = "game:%s:players"
	gamePlayersReadyKey string = "game:%s:players:ready"
	playerGamesKey      string = "player:%s:games"
)

var ctx = context.Background()

type DB struct {
	client *redis.Client
}

func NewDB() *DB {
	return &DB{
		client: NewClient(),
	}
}

func NewClient() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	return rdb
}

func (db *DB) AddPlayerToGame(id uuid.UUID, player string) error {
	pKey := fmt.Sprintf(gamePlayersKey, id)
	pReadyKey := fmt.Sprintf(gamePlayersReadyKey, id)

	err := db.client.SAdd(ctx, pKey, player).Err()
	if err != nil {
		return err
	}

	err = db.client.HSet(ctx, pReadyKey, player, false).Err()
	if err != nil {
		return err
	}

	err = db.addPlayerGame(player, id)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) PlayerReady(id uuid.UUID, player string) error {
	key := fmt.Sprintf(gamePlayersReadyKey, id)

	err := db.client.HSet(ctx, key, player, true).Err()
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) RemovePlayerFromGame(id uuid.UUID, player string) error {
	gamePlayersKey := fmt.Sprintf(gamePlayersKey, id)
	gamePlayersReadyKey := fmt.Sprintf(gamePlayersReadyKey, player)

	err := db.client.SRem(ctx, gamePlayersKey, player).Err()
	if err != nil {
		return err
	}

	err = db.client.HDel(ctx, gamePlayersReadyKey, player).Err()
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) RemovePlayerFromCreatedGames(player string) error {
	gameIDs, err := db.getPlayerGames(player)
	if err != nil {
		return err
	}
	for _, g := range gameIDs {
		id, _ := uuid.Parse(g)
		err = db.RemovePlayerFromGame(id, player)
		db.client.SRem(ctx, playerGamesKey, id)
		// need to expire the game and remove it
		if err != nil {
			return err
		}
	}
	return nil
}
