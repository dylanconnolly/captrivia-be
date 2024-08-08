package redis

import (
	"fmt"
	"log"
	"strconv"

	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type GameService struct {
	rdb *redis.Client
}

func NewGameService() *GameService {
	client := NewClient()
	return &GameService{
		rdb: client,
	}
}

func redisHashToGame(redisHash map[string]string) (captrivia.RepositoryGame, error) {
	id, err := uuid.Parse(redisHash["id"])
	if err != nil {
		return captrivia.RepositoryGame{}, err
	}

	playerCount, err := strconv.Atoi(redisHash["player_count"])
	if err != nil {
		return captrivia.RepositoryGame{}, err
	}

	questionCount, err := strconv.Atoi(redisHash["question_count"])
	if err != nil {
		return captrivia.RepositoryGame{}, err
	}

	return captrivia.RepositoryGame{
		ID:            id,
		Name:          redisHash["name"],
		PlayerCount:   playerCount,
		QuestionCount: questionCount,
		State:         captrivia.GameState(redisHash["state"]),
	}, nil
}

func (s *GameService) SaveGame(game *captrivia.Game) error {
	key := fmt.Sprintf(gameKey, game.ID)
	repGame := game.ToRepositoryGame()
	return s.rdb.HSet(ctx, key, repGame.ToHash()).Err()
}

func (s *GameService) GetGames() ([]captrivia.RepositoryGame, error) {
	var games []captrivia.RepositoryGame

	iter := s.rdb.Scan(ctx, 0, "game:*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		// strip "game:" from key string value
		id, err := uuid.Parse(key[5:])
		// swallow error and skip key if does not parse since that
		// means it is a key to game:id:players or game:id:players:ready
		if err != nil {
			continue
		}
		gameKey := fmt.Sprintf(gameKey, id)
		gameResp, err := s.rdb.HGetAll(ctx, gameKey).Result()
		if err != nil {
			log.Println("error getting games")
			return nil, err
		}

		game, err := redisHashToGame(gameResp)
		if err != nil {
			return nil, err
		}

		games = append(games, game)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return games, nil
}

func (s *GameService) ExpireGame(gameID uuid.UUID) error {
	key := fmt.Sprintf(gameKey, gameID)
	return s.rdb.Expire(ctx, key, EndedGameTTL).Err()
}
