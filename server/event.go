package server

import (
	"encoding/json"
	"log"
	"slices"

	"github.com/google/uuid"
)

type GameEventType string

const (
	GameEventTypeCreate          = "game_create"
	GameEventTypeStart           = "game_start"
	GameEventTypeEnd             = "game_end"
	GameEventTypeCountdown       = "game_countdown"
	GameEventTypeQuestion        = "game_question"
	GameEventTypePlayerEnter     = "game_player_enter"
	GameEventTypePlayerJoin      = "game_player_join"
	GameEventTypePlayerReady     = "game_player_ready"
	GameEventTypePlayerLeave     = "game_player_leave"
	GameEventTypePlayerCorrect   = "game_player_correct"
	GameEventTypePlayerIncorrect = "game_player_incorrect"
)

type EventPayload interface {
	json.Marshaler
}

type GameEvent struct {
	ID      uuid.UUID     `json:"id"`
	Payload EventPayload  `json:"payload"`
	Type    GameEventType `json:"type"`
}

// Payload to be sent to client when a new game is created
type GameEventCreate struct {
	Name          string `json:"name"`
	QuestionCount int    `json:"question_count"`
}

func (e GameEventCreate) Raw() *json.RawMessage {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(bytes)
	return &raw
}

// Used for all the player actions in a game lobby (Join, Ready, Leave)
type GameEventPlayerLobbyAction struct {
	Player string `json:"player"`
}

func (e GameEventPlayerLobbyAction) Raw() *json.RawMessage {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(bytes)
	return &raw
}

type GameEventPlayerEnter struct {
	Name          string          `json:"name"`
	Players       []string        `json:"players"`
	PlayersReady  map[string]bool `json:"players_ready"`
	QuestionCount int             `json:"question_count"`
}

func (e GameEventPlayerEnter) Raw() *json.RawMessage {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(bytes)
	return &raw
}

func newGameEventCreate(g Game) *GameEvent {
	payload := GameEventCreate{
		Name:          g.Name,
		QuestionCount: g.QuestionCount,
	}
	ge := &GameEvent{
		ID:      g.ID,
		Payload: payload.Raw(),
		Type:    GameEventTypeCreate,
	}

	return ge
}

func newGameEventPlayerEnter(gameID uuid.UUID, player string) *GameEvent {
	// TODO: handle updating game in state once data store is decided
	i := slices.IndexFunc(games, func(g *Game) bool { return g.ID == gameID })
	game := games[i]
	log.Printf("game: %+v", game)
	game.Players = append(game.Players, player)
	game.PlayerCount = game.PlayerCount + 1
	game.PlayersReady[player] = false

	payload := GameEventPlayerEnter{
		Name:          player,
		Players:       game.Players,
		PlayersReady:  game.PlayersReady,
		QuestionCount: game.QuestionCount,
	}

	ge := &GameEvent{
		ID:      gameID,
		Payload: payload.Raw(),
		Type:    GameEventTypePlayerEnter,
	}

	return ge
}

func newGameEventPlayerJoin(gameID uuid.UUID, player string) *GameEvent {
	payload := GameEventPlayerLobbyAction{
		Player: player,
	}

	ge := &GameEvent{
		ID:      gameID,
		Payload: payload.Raw(),
		Type:    GameEventTypePlayerJoin,
	}

	return ge
}

func newGameEventPlayerReady(gameID uuid.UUID, player string) *GameEvent {
	payload := GameEventPlayerLobbyAction{
		Player: player,
	}

	ge := &GameEvent{
		ID:      gameID,
		Payload: payload.Raw(),
		Type:    GameEventTypePlayerReady,
	}

	return ge
}
