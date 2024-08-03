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
	var cmd PlayerCommand
	err := json.Unmarshal(message, &cmd)
	if err != nil {
		log.Printf("error unmarshalling command: %s. Error: %s", message, err)
	}

	// determine type of incoming message
	switch cmd.Type {
	case PlayerCommandTypeCreate:
		c.handleCreateGame(cmd)
	case PlayerCommandTypeJoin:
		c.handleJoinGame(cmd)
	case PlayerCommandTypeReady:
		c.handlePlayerReady(cmd)

	}
}

func (c *Client) handleCreateGame(cmd PlayerCommand) {
	var payload PlayerCommandCreate

	err := json.Unmarshal(cmd.Payload, &payload)
	if err != nil {
		log.Printf("error unmarshalling create game command payload: %s\n Client: %+v Command: %s", err, c, cmd)
		c.send <- []byte("could not parse command payload")
	}

	game := newGame(payload.Name, payload.QuestionCount)
	games = append(games, &game)

	// broadcast event to all clients
	ge := newGameEventCreate(game)
	msg, err := json.Marshal(ge)
	if err != nil {
		log.Printf("error marshalling create game broadcast message: %s\n Client: %+v, Command: %s, GameEvent: %+v", err, c, cmd, ge)
		c.send <- []byte("there was an error creating game")
		return
	}
	c.manager.broadcast <- msg
}

func (c *Client) handleJoinGame(cmd PlayerCommand) {
	var payload PlayerCommandJoin

	err := json.Unmarshal(cmd.Payload, &payload)
	if err != nil {
		log.Printf("error unmarshalling join game command payload: %s\n Client: %+v Command: %s", err, c, cmd)
		c.send <- []byte("could not parse command payload")
		return
	}

	geEnter := newGameEventPlayerEnter(payload.GameID, c.name)
	msg, err := json.Marshal(geEnter)
	if err != nil {
		log.Printf("error marshalling player enter message: %s\n Client: %+v, Command: %s, GameEvent: %+v", err, c, cmd, geEnter)
		c.send <- []byte("there was an error joining the game")
		return
	}
	c.send <- msg

	// broadcast player join event to everyone in lobby
	geJoin := newGameEventPlayerJoin(payload.GameID, c.name)
	msg, err = json.Marshal(geJoin)
	if err != nil {
		log.Printf("error marshalling join game broadcast: %s\n Client: %+v, Command: %+v, GameEvent: %+v", err, c, cmd, geJoin)
		c.send <- []byte("there was an error joining the game")
		return
	}
	c.manager.broadcast <- msg
}

func (c *Client) handlePlayerReady(cmd PlayerCommand) {
	var payload PlayerCommandReady

	err := json.Unmarshal(cmd.Payload, &payload)
	if err != nil {
		log.Printf("error unmarshalling join game command payload: %s\n Client: %+v Command: %s", err, c, cmd)
		c.send <- []byte("could not parse command payload")
		return
	}

	ge := newGameEventPlayerReady(payload.GameID, c.name)
	msg, err := json.Marshal(ge)
	if err != nil {
		log.Printf("error marshalling player ready broadcast: %s\n Client: %+v Command: %s, GameEvent: %+v", err, c, cmd, ge)
		c.send <- []byte("there was an error marking yourself as ready")
		return
	}
	c.manager.broadcast <- msg
}

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
