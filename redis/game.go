package redis

import (
	"fmt"
	"log"
	"strconv"

	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/google/uuid"
)

type GamePlayers []string

type GamePlayersReady map[string]bool

type Game struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	QuestionCount int       `json:"question_count"`
	State         string    `json:"state"`
}

type GameQuestion struct {
	ID           string `json:"id"`
	Question     string `json:"question"`
	CorrectIndex int    `json:"correct_index"`
}

func newGame(name string, qCount int) Game {
	return Game{
		ID:            uuid.New(),
		Name:          name,
		QuestionCount: qCount,
		State:         gameStateWaiting,
	}
}

func (g *Game) toRedisHash() map[string]interface{} {
	return map[string]interface{}{
		"id":             g.ID.String(),
		"name":           g.Name,
		"question_count": g.QuestionCount,
		"state":          g.State,
	}
}

func (db *DB) CreateGame(player string, name string, qCount int) (*Game, error) {
	g := newGame(name, qCount)

	gKey := fmt.Sprintf(gameKey, g.ID)
	pGKey := fmt.Sprintf(playerGamesKey, player)

	err := db.client.HSet(ctx, gKey, g.toRedisHash()).Err()
	if err != nil {
		return nil, err
	}

	err = db.client.SAdd(ctx, pGKey, g.ID.String()).Err()
	if err != nil {
		return nil, err
	}

	// db.generateGameQuestions(g.ID, qCount)

	log.Printf("create game: %+v", g)

	return &g, nil
}

func (db *DB) GetGame(id uuid.UUID) (*captrivia.Game, error) {
	redisGame, err := db.getGameHashSet(id)
	if err != nil {
		return nil, err
	}

	players, err := db.getGamePlayersSet(id)
	if err != nil {
		return nil, err
	}

	playersReady, err := db.getGamePlayersReadyHashSet(id)
	if err != nil {
		return nil, err
	}

	count, err := strconv.Atoi(redisGame["question_count"])
	if err != nil {
		return nil, err
	}

	gID, err := uuid.Parse(redisGame["id"])
	if err != nil {
		return nil, err
	}

	g := captrivia.NewGame(
		gID,
		redisGame["name"],
		players,
		playersReady,
		count,
		redisGame["state"],
	)

	return g, nil
}

func (db *DB) GetAllGames() ([]captrivia.Game, error) {
	var games []captrivia.Game

	iter := db.client.Scan(ctx, 0, "game:*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		// strip "game:" from key string value
		id, err := uuid.Parse(key[5:])
		// swaller error and skip key if does not parse since that
		// means it is a key to game:id:players or game:id:players:ready
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

func (db *DB) StartGame(id uuid.UUID) error {
	return nil
}

func (db *DB) getGameHashSet(id uuid.UUID) (map[string]string, error) {
	key := fmt.Sprintf("game:%s", id)

	gameFields, err := db.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	return gameFields, nil
}

func (db *DB) getGamePlayersSet(id uuid.UUID) (GamePlayers, error) {
	key := fmt.Sprintf("game:%s:players", id)

	players, err := db.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	return players, nil
}

func (db *DB) getGamePlayersReadyHashSet(id uuid.UUID) (GamePlayersReady, error) {
	key := fmt.Sprintf("game:%s:players:ready", id)
	ready := make(GamePlayersReady)

	playersReady, err := db.client.HGetAll(ctx, key).Result()
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
