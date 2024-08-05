package redis

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	gameStateWaiting   = "waiting"
	gameStateCountdown = "countdown"
	gameStateQuestion  = "question"
	gameStateEnded     = "ended"

	gameKey         string = "game:%s"
	playersKey      string = "game:%s:players"
	playersReadyKey string = "game:%s:players:ready"
	playerGamesKey  string = "players:%s:games"
)

var ctx = context.Background()

type DB struct {
	db *redis.Client
}

type Game struct {
	ID            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	Players       []string        `json:"players"`
	PlayersReady  map[string]bool `json:"players_ready"`
	PlayerCount   int             `json:"player_count"`
	QuestionCount int             `json:"question_count"`
	State         string          `json:"state"`
}

func (g *Game) toRedisHash() map[string]interface{} {
	return map[string]interface{}{
		"id":             g.ID.String(),
		"name":           g.Name,
		"player_count":   g.PlayerCount,
		"question_count": g.QuestionCount,
		"state":          g.State,
	}
}

func newGame(name string, qCount int) Game {
	return Game{
		ID:            uuid.New(),
		Name:          name,
		PlayersReady:  make(map[string]bool),
		PlayerCount:   0,
		QuestionCount: qCount,
		State:         gameStateWaiting,
	}
}

func NewDB() *DB {
	return &DB{
		db: NewClient(),
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

func (db *DB) CreateGame(player string, name string, qCount int) (*Game, error) {
	g := newGame(name, qCount)

	gKey := fmt.Sprintf(gameKey, g.ID)
	pGKey := fmt.Sprintf(playerGamesKey, player)

	err := db.db.HSet(ctx, gKey, g.toRedisHash()).Err()
	if err != nil {
		return nil, err
	}

	err = db.db.SAdd(ctx, pGKey, g.ID).Err()
	if err != nil {
		return nil, err
	}

	log.Printf("create game: %+v", g)

	return &g, nil
}

func (db *DB) GetGame(id uuid.UUID) (*Game, error) {
	gameFields, err := db.getGameHash(id)
	if err != nil {
		return nil, err
	}

	players, err := db.getGamePlayers(id)
	if err != nil {
		return nil, err
	}

	playersReady, err := db.getPlayersReady(id)
	if err != nil {
		return nil, err
	}

	count, err := strconv.Atoi(gameFields["question_count"])
	if err != nil {
		return nil, err
	}

	g := Game{
		ID:            uuid.MustParse(gameFields["id"]),
		Name:          gameFields["name"],
		Players:       players,
		PlayersReady:  playersReady,
		PlayerCount:   len(players),
		QuestionCount: count,
		State:         gameFields["state"],
	}

	return &g, nil
}

func (db *DB) GetAllGames() ([]Game, error) {
	var games []Game

	iter := db.db.Scan(ctx, 0, "game:*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		// strip "game:" from string value
		id, err := uuid.Parse(key[5:])
		if err != nil {
			continue
		}
		game, err := db.GetGame(id)
		if err != nil {
			return nil, err
		}
		games = append(games, *game)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return games, nil
}

func (db *DB) AddPlayerToGame(id uuid.UUID, player string) error {
	pKey := fmt.Sprintf(playersKey, id)
	pReadyKey := fmt.Sprintf(playersReadyKey, id)

	err := db.db.SAdd(ctx, pKey, player).Err()
	if err != nil {
		return err
	}

	err = db.db.HSet(ctx, pReadyKey, player, false).Err()
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) PlayerReady(id uuid.UUID, player string) error {
	key := fmt.Sprintf(playersReadyKey, id)

	err := db.db.HSet(ctx, key, player, true).Err()
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) getGameHash(id uuid.UUID) (map[string]string, error) {
	key := fmt.Sprintf("game:%s", id)

	gameFields, err := db.db.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	return gameFields, nil
}

func (db *DB) getGamePlayers(id uuid.UUID) ([]string, error) {
	key := fmt.Sprintf("game:%s:players", id)

	players, err := db.db.SMembers(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	return players, nil
}

func (db *DB) getPlayersReady(id uuid.UUID) (map[string]bool, error) {
	key := fmt.Sprintf("game:%s:players:ready", id)
	ready := make(map[string]bool)

	playersReady, err := db.db.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	for p, r := range playersReady {
		readyBool, err := strconv.ParseBool(r)
		if err != nil {
			ready[p] = false
		}
		ready[p] = readyBool
	}

	return ready, nil
}
