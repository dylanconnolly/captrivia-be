package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/dylanconnolly/captrivia-be/game"
	"github.com/dylanconnolly/captrivia-be/player"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

type Client struct {
	name     string
	manager  *ClientManager
	conn     *websocket.Conn
	messages chan []byte
}

// upgrades HTTP protocol to websocket and creates a client to manage the connection and messages
func newClient(name string, manager *ClientManager, w http.ResponseWriter, r *http.Request) (*Client, error) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return r.Header.Get("Origin") == "http://localhost:3000"
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error upgrading connection: %s\n", err)
		return nil, err
	}

	c := &Client{
		conn:     conn,
		name:     name,
		manager:  manager,
		messages: make(chan []byte),
	}

	return c, nil
}

func (c *Client) readMessages() {
	c.conn.ReadMessage()
	for {
		t, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("error reading message: %s\n", err)
			break
		}
		var data player.PlayerCommand
		err = json.Unmarshal(message, &data)
		if err != nil {
			log.Printf("error unmarshalling: %s", err)
		}
		var payload player.CreatePayload
		if data.Type == player.PlayerCommandTypeCreate {
			_ = json.Unmarshal(data.Payload, &payload)
		}
		log.Printf("message: %+v, payload: %+v, message_type: %d\n", data, payload, t)
		game_str := struct {
			ID            uuid.UUID `json:"id"`
			Name          string    `json:"name"`
			PlayerCount   int       `json:"player_count"`
			QuestionCount int       `json:"question_count"`
			State         string    `json:"state"`
		}{uuid.New(), payload.Name, 0, payload.QuestionCount, "waiting"}
		games = append(games, game_str)

		log.Println("writing message back to client")

		ge := game.GameEvent{
			ID:      game_str.ID,
			Payload: &data.Payload,
			Type:    "game_create",
		}
		resp, _ := json.Marshal(ge)
		c.conn.WriteMessage(websocket.TextMessage, resp)

		log.Printf("games: %v", games)
	}
}

// func (c *Client) sendMessages() {

// }

// handle websocket requests from frontend
func (c *Client) handleWebsockets() {
	c.manager.register <- c

	go c.readMessages()
}
