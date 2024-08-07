package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

// Client manages the websocket for a user and communicates with the ClientManager
type Client struct {
	name  string
	hub   *Hub
	conn  *websocket.Conn
	send  chan []byte
	close chan bool
}

// Creates a new client but does not attach websocket connection. Running serveWebsocket() upgrades connection and begins
// begins running client
func NewClient(name string, hub *Hub) *Client {
	c := &Client{
		name:  name,
		hub:   hub,
		send:  make(chan []byte, 256),
		close: make(chan bool),
	}

	return c
}

func (c *Client) readMessage() {
	defer c.conn.Close()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error reading message: %s\n", err)
			}
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
		c.send <- []byte("could not parse command payload")
		return
	}

	// determine type of incoming message
	switch cmd.Type {
	case PlayerCommandTypeCreate:
		var payload PlayerCommandCreate

		err := json.Unmarshal(cmd.Payload, &payload)
		if err != nil {
			log.Printf("error unmarshalling create game command payload: %s\n Client: %s Command: %s", err, c.name, cmd)
			c.send <- []byte("could not parse command payload")
		}

		c.handleCreateGame(payload)
	case PlayerCommandTypeJoin:
		var payload PlayerLobbyCommand

		err := json.Unmarshal(cmd.Payload, &payload)
		if err != nil {
			log.Printf("error unmarshalling join game command payload: %s\n Client: %s Command: %s", err, c.name, cmd)
			c.send <- []byte("could not parse command payload")
			return
		}
		c.handleJoinGame(payload)
	case PlayerCommandTypeReady:
		c.handlePlayerReady(cmd)
	case PlayerCommandTypeStart:
		c.handleStartGame(cmd)
	case PlayerCommandTypeAnswer:
		c.handlePlayerAnswer(cmd)
	}
}

func (c *Client) handleCreateGame(payload PlayerCommandCreate) {
	game := captrivia.NewGame(payload.Name, payload.QuestionCount)
	gh := NewGameHub(game, c.hub.GameService)

	ge := newGameEventCreate(game.ID, game.Name, game.QuestionCount)
	// dont want to broadcast to all
	c.hub.broadcast <- ge.toBytes()

	// create GameHub to manage game and client
	gh.register <- c
	GameHubs[gh.ID] = gh
}

func (c *Client) handleJoinGame(payload PlayerLobbyCommand) {
	log.Println("handling player join")
	if gameHub, ok := GameHubs[payload.GameID]; ok {
		gameHub.register <- c
	} else {
		c.send <- []byte("could not connect to game")
	}
}

func (c *Client) handlePlayerReady(cmd PlayerCommand) {
	var payload PlayerLobbyCommand

	err := json.Unmarshal(cmd.Payload, &payload)
	if err != nil {
		log.Print(err)
		c.send <- []byte("error unmarshalling payload for player ready command")
	}

	gameCommand := GameLobbyCommand{
		player:  c.name,
		payload: payload,
		Type:    PlayerCommandTypeReady,
	}

	if gameHub, ok := GameHubs[payload.GameID]; ok {
		gameHub.commands <- gameCommand
	} else {
		c.send <- []byte("error occured marking player as ready")
	}
}

func (c *Client) handleStartGame(cmd PlayerCommand) {
	var payload PlayerLobbyCommand

	err := json.Unmarshal(cmd.Payload, &payload)
	if err != nil {
		log.Printf("error unmarshalling start game command payload: %s\n Client: %+v Command: %s", err, c, cmd)
		c.send <- []byte("could not parse command payload")
		return
	}

	gameCommand := GameLobbyCommand{
		player:  c.name,
		payload: payload,
		Type:    PlayerCommandTypeStart,
	}

	if gameHub, ok := GameHubs[payload.GameID]; ok {
		gameHub.commands <- gameCommand
	}
}

func (c *Client) handlePlayerAnswer(cmd PlayerCommand) {
	var payload PlayerCommandAnswer
	err := json.Unmarshal(cmd.Payload, &payload)
	if err != nil {
		log.Printf("error unmarshalling player answer: %s\n Client: %+v Command: %s", err, c, cmd)
		c.send <- []byte("could not parse command payload")
		return
	}

	ga := GameAnswer{
		QuestionID: payload.QuestionID,
		Player:     c.name,
		Index:      payload.Index,
	}

	if gameHub, ok := GameHubs[payload.GameID]; ok {
		gameHub.answers <- ga
	}
}

func (c *Client) writeMessage() {
	defer c.conn.Close()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			err := c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("error writing message to websocket: ERROR=%s. MESSAGE=%s, CLIENT=%+v\n", err, message, c)
			}
		}
	}
}

// upgrades connection to websocket on client and registers client with client Hub
func (c *Client) ServeWebsocket(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return r.Header.Get("Origin") == "http://localhost:3000"
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error upgrading connection: %s\n", err)
		return
	}
	log.Printf("client connected: %s", c.name)
	c.conn = conn
	c.hub.register <- c
	pe := newPlayerEventConnect(c.name)
	c.hub.broadcast <- pe.toBytes()

	go c.readMessage()
	go c.writeMessage()
}

func (c *Client) Close() {
	log.Println("client connection closed")
	c.hub.disconnect <- c
	c.conn.Close()
}
