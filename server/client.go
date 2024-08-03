package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

// Client manages the websocket for a user and communicates with the ClientManager
type Client struct {
	name    string
	manager *ClientManager
	conn    *websocket.Conn
	send    chan []byte
}

// Creates a new client but does not attach websocket connection. Running serveWebsocket upgrades connection and begins
// begins running client
func newClient(name string, manager *ClientManager) *Client {
	c := &Client{
		name:    name,
		manager: manager,
		send:    make(chan []byte, 256),
	}

	return c
}

func (c *Client) readMessage() {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("error reading message: %s\n", err)
			break
		}

		c.handleRead(message)
	}
}

func (c *Client) handleRead(message []byte) {
	var command PlayerCommand
	err := json.Unmarshal(message, &command)
	if err != nil {
		log.Printf("error unmarshalling command: %s. Error: %s", message, err)
	}

	// determine type of incoming message
	switch command.Type {
	case PlayerCommandTypeCreate:
		resp, err := command.handleCreateGameCommand()
		if err != nil {
			return
		}
		c.manager.broadcast <- resp
	case PlayerCommandTypeJoin:
		c.handleJoinGame(command)
		// c.manager.broadcast <- resp

	}
}

func (c *Client) handleJoinGame(cmd PlayerCommand) {
	var payload PlayerCommandJoin

	err := json.Unmarshal(cmd.Payload, &payload)
	if err != nil {
		log.Printf("error unmarshalling join game command payload: %s. Command: %s", err, cmd)
		c.send <- []byte("could not parse command payload")
	}

	geEnter := newGameEventPlayerEnter(payload.GameID, c.name)
	msg, err := json.Marshal(geEnter)
	if err != nil {
		log.Printf("error marshalling player enter message: %s\n Command: %+v, Event: %+v", err, c, geEnter)
		c.send <- []byte("there was an error joining the game")
		return
	}
	c.send <- msg

	// broadcast player join event to everyone in lobby
	geJoin := newGameEventPlayerJoin(payload.GameID, c.name)
	msg, err = json.Marshal(geJoin)
	if err != nil {
		log.Printf("error marshalling join game broadcast: %s\n Command: %+v, Event: %+v", err, c, geJoin)
		c.send <- []byte("there was an error joining the game")
		return
	}
	c.manager.broadcast <- msg
}

// func (c *Client) handlePlayerJoin(command PlayerCommand) {
// 	var cmd PlayerCommandJoin
// 	json.Unmarshal(command.Payload, &cmd)
// 	ready := make(map[string]bool)
// 	ready["Player 1"] = false
// 	ready["Player 2"] = true
// 	payload := GameEventPlayerEnter{
// 		Name:          c.name,
// 		Players:       []string{"Player 1", "Player 2"},
// 		PlayersReady:  ready,
// 		QuestionCount: 5,
// 	}

// 	p, _ := structToEventPayload(payload)

// 	ge := &GameEvent{
// 		ID:      cmd.GameID,
// 		Payload: p,
// 		Type:    GameEventTypePlayerEnter,
// 	}

// 	resp, err := json.Marshal(ge)
// 	if err != nil {
// 		log.Printf("error marshalling create game response: %s\n Command: %+v, Event: %+v", err, c, ge)
// 	}
// 	c.send <- resp
// }

func (c *Client) writeMessage() {
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			err := c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("error writing message to websocket. Client: %+v, message: %s", c, message)
			}
		}
	}
}

// upgrades connection to websocket on client and registers client with ClientManager
func (c *Client) serveWebsocket(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return r.Header.Get("Origin") == "http://localhost:3000"
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error upgrading connection: %s\n", err)
	}
	c.conn = conn
	c.manager.register <- c

	go c.readMessage()
	go c.writeMessage()
}
