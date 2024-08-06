package server

import (
	"encoding/json"

	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/dylanconnolly/captrivia-be/redis"
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

type GameEventCountdown struct {
	Seconds int `json:"seconds"`
}

func (e GameEventCountdown) Raw() *json.RawMessage {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(bytes)
	return &raw
}

type GameEventQuestion struct {
	ID       string   `json:"id"`
	Options  []string `json:"options"`
	Question string   `json:"question"`
	Seconds  int      `json:"seconds"`
}

func (e GameEventQuestion) Raw() *json.RawMessage {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(bytes)
	return &raw
}

type GameEventPlayerAnswer struct {
	QuestionID string `json:"id"`
	Player     string `json:"player"`
}

func (e GameEventPlayerAnswer) Raw() *json.RawMessage {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(bytes)
	return &raw
}

type EmptyGameEvent struct{}

func (e EmptyGameEvent) Raw() *json.RawMessage {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(bytes)
	return &raw
}

func newGameEventCreate(g redis.Game) *GameEvent {
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

func newGameEvent(id uuid.UUID, payload EventPayload, eventType GameEventType) GameEvent {
	return GameEvent{
		ID:      id,
		Payload: payload,
		Type:    eventType,
	}
}

func newGameEventPlayerEnter(player string, game *captrivia.Game) GameEvent {
	payload := GameEventPlayerEnter{
		Name:          player,
		Players:       game.Players,
		PlayersReady:  game.PlayersReady,
		QuestionCount: game.QuestionCount,
	}

	ge := newGameEvent(game.ID, payload.Raw(), GameEventTypePlayerEnter)

	return ge
}

func newGameEventPlayerJoin(id uuid.UUID, player string) GameEvent {
	payload := GameEventPlayerLobbyAction{
		Player: player,
	}

	ge := newGameEvent(id, payload.Raw(), GameEventTypePlayerJoin)

	return ge
}

func newGameEventPlayerReady(id uuid.UUID, player string) GameEvent {
	payload := GameEventPlayerLobbyAction{
		Player: player,
	}

	ge := newGameEvent(id, payload.Raw(), GameEventTypePlayerReady)

	return ge
}

func newGameEventStart(id uuid.UUID) GameEvent {
	payload := EmptyGameEvent{}

	ge := newGameEvent(id, payload.Raw(), GameEventTypeStart)

	return ge
}

func newGameEventCountdown(id uuid.UUID, duration int) GameEvent {
	payload := GameEventCountdown{
		Seconds: duration,
	}

	ge := newGameEvent(id, payload.Raw(), GameEventTypeCountdown)

	return ge
}

func newGameEventQuestion(id uuid.UUID, question *captrivia.Question, duration int) GameEvent {
	payload := GameEventQuestion{
		ID:       question.ID,
		Options:  question.Options,
		Question: question.QuestionText,
		Seconds:  duration,
	}

	ge := newGameEvent(id, payload.Raw(), GameEventTypeQuestion)

	return ge
}

func newGameEventPlayerCorrect(id uuid.UUID, player string, questionID string) GameEvent {
	payload := GameEventPlayerAnswer{
		QuestionID: questionID,
		Player:     player,
	}

	ge := newGameEvent(id, payload.Raw(), GameEventTypePlayerCorrect)

	return ge
}

func newGameEventPlayerIncorrect(id uuid.UUID, player string, questionID string) GameEvent {
	payload := GameEventPlayerAnswer{
		QuestionID: questionID,
		Player:     player,
	}

	ge := newGameEvent(id, payload.Raw(), GameEventTypePlayerIncorrect)

	return ge
}
