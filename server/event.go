package server

import (
	"encoding/json"
	"log"

	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/google/uuid"
)

type GameEventType string
type PlayerEventType string

const (
	// event types broadcasted to all clients
	PlayerEventTypeConnect    PlayerEventType = "player_connect"
	PlayerEventTypeDisconnect PlayerEventType = "player_disconnect"

	// event types broadcasted to anyone not in a game
	GameEventTypeCreate      GameEventType = "game_create"
	GameEventTypeStateChange GameEventType = "game_state_change"
	GameEventTypePlayerCount GameEventType = "game_player_count"
	GameEventTypeDestroy     GameEventType = "game_destroy"

	// event types broadcasted to active game participants
	GameEventTypeStart           GameEventType = "game_start"
	GameEventTypeEnd             GameEventType = "game_end"
	GameEventTypeCountdown       GameEventType = "game_countdown"
	GameEventTypeQuestion        GameEventType = "game_question"
	GameEventTypePlayerEnter     GameEventType = "game_player_enter"
	GameEventTypePlayerJoin      GameEventType = "game_player_join"
	GameEventTypePlayerReady     GameEventType = "game_player_ready"
	GameEventTypePlayerLeave     GameEventType = "game_player_leave"
	GameEventTypePlayerCorrect   GameEventType = "game_player_correct"
	GameEventTypePlayerIncorrect GameEventType = "game_player_incorrect"
)

type EventPayload interface {
	json.Marshaler
}

// type GameEventEmitter interface {
// 	Emit(chan)
// }

type GameEvent struct {
	ID      uuid.UUID     `json:"id"`
	Payload EventPayload  `json:"payload"`
	Type    GameEventType `json:"type"`
}

func (e GameEvent) toBytes() []byte {
	bytes, err := json.Marshal(e)
	if err != nil {
		log.Printf("error marshalling %s for gameID=%s. Err: %s", e.Type, e.ID, err)
		return []byte("error marshalling GameEvent response")
	}
	return bytes
}

type PlayerEvent struct {
	Payload EventPayload    `json:"payload"`
	Player  string          `json:"player"`
	Type    PlayerEventType `json:"type"`
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

type GameEventPlayerCount struct {
	PlayerCount int `json:"player_count"`
}

func (e GameEventPlayerCount) Raw() *json.RawMessage {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(bytes)
	return &raw
}

type GameEventStateChange struct {
	State captrivia.GameState `json:"state"`
}

func (e GameEventStateChange) Raw() *json.RawMessage {
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

type GameEventEnd struct {
	Scores []captrivia.PlayerScore `json:"scores"`
}

func (e GameEventEnd) Raw() *json.RawMessage {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(bytes)
	return &raw
}

type EmptyPayload struct{}

func (e EmptyPayload) Raw() *json.RawMessage {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(bytes)
	return &raw
}

func newGameEventCreate(gameID uuid.UUID, gameName string, gameQCount int) GameEvent {
	payload := GameEventCreate{
		Name:          gameName,
		QuestionCount: gameQCount,
	}
	ge := GameEvent{
		ID:      gameID,
		Payload: payload.Raw(),
		Type:    GameEventTypeCreate,
	}

	return ge
}

func newGameEvent(gameID uuid.UUID, payload EventPayload, eventType GameEventType) GameEvent {
	return GameEvent{
		ID:      gameID,
		Payload: payload,
		Type:    eventType,
	}
}

func newGameEventPlayerEnter(player string, game *captrivia.Game) GameEvent {
	payload := GameEventPlayerEnter{
		Name:          player,
		Players:       game.PlayerNames(),
		PlayersReady:  game.PlayersReady,
		QuestionCount: game.QuestionCount,
	}

	ge := newGameEvent(game.ID, payload.Raw(), GameEventTypePlayerEnter)

	return ge
}

func newGameEventPlayerJoin(gameID uuid.UUID, player string) GameEvent {
	payload := GameEventPlayerLobbyAction{
		Player: player,
	}

	ge := newGameEvent(gameID, payload.Raw(), GameEventTypePlayerJoin)

	return ge
}

func newGameEventPlayerReady(gameID uuid.UUID, player string) GameEvent {
	payload := GameEventPlayerLobbyAction{
		Player: player,
	}

	ge := newGameEvent(gameID, payload.Raw(), GameEventTypePlayerReady)

	return ge
}

func newGameEventPlayerLeave(gameID uuid.UUID, player string) GameEvent {
	payload := GameEventPlayerLobbyAction{
		Player: player,
	}

	ge := newGameEvent(gameID, payload.Raw(), GameEventTypePlayerLeave)

	return ge
}

func newGameEventPlayerCount(gameID uuid.UUID, count int) GameEvent {
	payload := GameEventPlayerCount{
		PlayerCount: count,
	}

	ge := newGameEvent(gameID, payload.Raw(), GameEventTypePlayerCount)

	return ge
}

func newGameEventStart(gameID uuid.UUID) GameEvent {
	payload := EmptyPayload{}

	ge := newGameEvent(gameID, payload.Raw(), GameEventTypeStart)

	return ge
}

func newGameEventStateChange(gameID uuid.UUID, state captrivia.GameState) GameEvent {
	payload := GameEventStateChange{
		State: state,
	}

	ge := newGameEvent(gameID, payload.Raw(), GameEventTypeStateChange)

	return ge
}

func newGameEventCountdown(gameID uuid.UUID, duration int) GameEvent {
	payload := GameEventCountdown{
		Seconds: duration,
	}

	ge := newGameEvent(gameID, payload.Raw(), GameEventTypeCountdown)

	return ge
}

func newGameEventQuestion(gameID uuid.UUID, question captrivia.Question, duration int) GameEvent {
	payload := GameEventQuestion{
		ID:       question.ID,
		Options:  question.Options,
		Question: question.QuestionText,
		Seconds:  duration,
	}

	ge := newGameEvent(gameID, payload.Raw(), GameEventTypeQuestion)

	return ge
}

func newGameEventPlayerCorrect(gameID uuid.UUID, player string, questionID string) GameEvent {
	payload := GameEventPlayerAnswer{
		QuestionID: questionID,
		Player:     player,
	}

	ge := newGameEvent(gameID, payload.Raw(), GameEventTypePlayerCorrect)

	return ge
}

func newGameEventPlayerIncorrect(gameID uuid.UUID, player string, questionID string) GameEvent {
	payload := GameEventPlayerAnswer{
		QuestionID: questionID,
		Player:     player,
	}

	ge := newGameEvent(gameID, payload.Raw(), GameEventTypePlayerIncorrect)

	return ge
}

func newGameEventEnd(gameID uuid.UUID, scores []captrivia.PlayerScore) GameEvent {
	payload := GameEventEnd{
		Scores: scores,
	}

	ge := newGameEvent(gameID, payload.Raw(), GameEventTypeEnd)

	return ge
}

func newPlayerEvent(player string, payload EventPayload, eventType PlayerEventType) PlayerEvent {
	return PlayerEvent{
		Payload: payload,
		Player:  player,
		Type:    eventType,
	}
}

func newPlayerEventConnect(player string) PlayerEvent {
	payload := EmptyPayload{}

	pe := newPlayerEvent(player, payload.Raw(), PlayerEventTypeConnect)

	return pe
}

func newPlayerEventDisconnect(player string) PlayerEvent {
	payload := EmptyPayload{}

	pe := newPlayerEvent(player, payload.Raw(), PlayerEventTypeDisconnect)

	return pe
}
