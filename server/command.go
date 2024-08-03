package server

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type PlayerCommandType string
type PlayerEventType string

const (
	PlayerCommandTypeCreate   PlayerCommandType = "create"
	PlayerCommandTypeJoin     PlayerCommandType = "join"
	PlayerCommandTypeReady    PlayerCommandType = "ready"
	PlayerCommandTypeStart    PlayerCommandType = "start"
	PlayerCommandTypeAnswer   PlayerCommandType = "answer"
	PlayerEventTypeConnect    PlayerEventType   = "player_connect"
	PlayerEventTypeDisconnect PlayerEventType   = "player_disconnect"
)

type PlayerCommand struct {
	Nonce   string            `json:"nonce"`
	Payload json.RawMessage   `json:"payload"`
	Type    PlayerCommandType `json:"type"`
}

type PlayerCommandCreate struct {
	Name          string `json:"name"`
	QuestionCount int    `json:"question_count"`
}

type PlayerCommandJoin struct {
	GameID uuid.UUID `json:"game_id"`
}

func (c PlayerCommand) handleCreateGameCommand() ([]byte, error) {
	var payload PlayerCommandCreate
	err := json.Unmarshal(c.Payload, &payload)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling create command payload: %+v. Error: %s", payload, err)
	}
	game := newGame(payload.Name, payload.QuestionCount)
	games = append(games, &game)

	// broadcast event
	ge := newGameEventCreate(game)
	resp, err := json.Marshal(ge)
	if err != nil {
		return nil, fmt.Errorf("error marshalling create game response: %s\n Command: %+v, Event: %+v", err, c, ge)
	}
	return resp, nil
}

func (c PlayerCommand) handleJoinGameCommand(playerName string) ([]byte, error) {
	var payload PlayerCommandJoin
	err := json.Unmarshal(c.Payload, &payload)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling create command payload: %+v. Error: %s", payload, err)
	}

	ge := newGameEventPlayerEnter(payload.GameID, playerName)

	resp, err := json.Marshal(ge)
	if err != nil {
		return nil, fmt.Errorf("error marshalling join game response: %s\n Command: %+v, Event: %+v", err, c, ge)
	}
	return resp, nil
}
