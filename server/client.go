package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	PlayerCommandTypeCreate PlayerCommandType = "create"
	PlayerCommandTypeJoin   PlayerCommandType = "join"
	PlayerCommandTypeReady  PlayerCommandType = "ready"
	PlayerCommandTypeStart  PlayerCommandType = "start"
	PlayerCommandTypeAnswer PlayerCommandType = "answer"
)

var upgrader = websocket.Upgrader{}

type PlayerCommandType string

type WebSocketConn interface {
	ReadMessage() (int, []byte, error)
	WriteMessage(int, []byte) error
	Close() error
}

type PlayerCommand struct {
	Nonce   string            `json:"nonce"`
	Payload json.RawMessage   `json:"payload"`
	Type    PlayerCommandType `json:"type"`
}

type GameLobbyCommand struct {
	Player  string
	Payload PlayerLobbyCommand
	Type    PlayerCommandType
}

type PlayerCommandCreate struct {
	Name          string `json:"name"`
	QuestionCount int    `json:"question_count"`
}

type PlayerLobbyCommand struct {
	GameID uuid.UUID `json:"game_id"`
}

type PlayerCommandAnswer struct {
	GameID     uuid.UUID `json:"game_id"`
	Index      int       `json:"index"`
	QuestionID string    `json:"question_id"`
}

// Client manages the websocket for a user and communicates with the ClientManager
type Client struct {
	name    string
	gameHub *GameHub
	hub     *Hub
	mu      sync.Mutex
	Conn    WebSocketConn
	Send    chan []byte
	closed  bool
}

// Creates a new client but does not attach websocket connection. Running serveWebsocket() upgrades connection and begins
// begins running client
func NewClient(name string, hub *Hub) *Client {
	c := &Client{
		name: name,
		hub:  hub,
		Send: make(chan []byte, 256),
	}

	return c
}

func (c *Client) readMessage() {
	defer c.Close()

	for {
		_, message, err := c.Conn.ReadMessage()
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
	c.mu.Lock()
	defer c.mu.Unlock()
	var cmd PlayerCommand
	err := json.Unmarshal(message, &cmd)
	if err != nil {
		log.Printf("error unmarshalling command: %s. Error: %s", message, err)
		c.Send <- []byte("could not parse command payload")
		// c.mu.Unlock()
		return
	}

	// determine type of incoming message
	switch cmd.Type {
	case PlayerCommandTypeCreate:
		var payload PlayerCommandCreate
		err := json.Unmarshal(cmd.Payload, &payload)
		if err != nil {
			log.Printf("error unmarshalling create game command payload: %s\n Client: %s Command: %s", err, c.name, cmd)
			c.Send <- []byte("could not parse command payload")
			// c.mu.Unlock()
			return
		}
		c.handleCreateGame(payload)

	case PlayerCommandTypeJoin:
		var payload PlayerLobbyCommand
		err := json.Unmarshal(cmd.Payload, &payload)
		if err != nil {
			log.Printf("error unmarshalling join game command payload: %s\n Client: %s Command: %s", err, c.name, cmd)
			c.Send <- []byte("could not parse command payload")
			// c.mu.Unlock()
			return
		}
		c.handleJoinGame(payload)

	case PlayerCommandTypeReady:
		var payload PlayerLobbyCommand
		err := json.Unmarshal(cmd.Payload, &payload)
		if err != nil {
			log.Print(err)
			c.Send <- []byte("error unmarshalling payload for player ready command")
			// c.mu.Unlock()
		}

		c.handlePlayerReady(payload)

	case PlayerCommandTypeStart:
		var payload PlayerLobbyCommand
		err := json.Unmarshal(cmd.Payload, &payload)
		if err != nil {
			log.Printf("error unmarshalling start game command payload: %s\n Client: %+v Command: %s", err, c, cmd)
			c.Send <- []byte("could not parse command payload")
			// c.mu.Unlock()
			return
		}

		c.handleStartGame(payload)

	case PlayerCommandTypeAnswer:
		var payload PlayerCommandAnswer
		err := json.Unmarshal(cmd.Payload, &payload)
		if err != nil {
			log.Printf("error unmarshalling player answer: %s\n Client: %+v Command: %s", err, c, cmd)
			c.Send <- []byte("could not parse command payload")
			// c.mu.Unlock()
			return
		}

		c.handlePlayerAnswer(payload)
	default:
		log.Printf("got unknown command %s", cmd)
	}
}

func (c *Client) handleCreateGame(payload PlayerCommandCreate) {
	// creates GameHub which manages the state and lifecycle of the game
	gameHub, err := c.hub.NewGameHub(payload.Name, payload.QuestionCount)
	if err != nil {
		log.Println(err)
		return
	}

	go gameHub.Run(context.Background())

	gameHub.Register <- c
}

func (c *Client) handleJoinGame(payload PlayerLobbyCommand) {
	gh, err := c.hub.GetGameHub(payload.GameID)
	if err != nil {
		log.Println(err)
		return
	}

	gh.Register <- c
}

func (c *Client) handlePlayerReady(payload PlayerLobbyCommand) {
	gameCommand := GameLobbyCommand{
		Player:  c.name,
		Payload: payload,
		Type:    PlayerCommandTypeReady,
	}

	gh, err := c.hub.GetGameHub(payload.GameID)
	if err != nil {
		log.Println(err)
		return
	}

	// if c.gameHub == nil {
	// 	gh.register <- c
	// }

	gh.Commands <- gameCommand
}

func (c *Client) handleStartGame(payload PlayerLobbyCommand) {
	gameCommand := GameLobbyCommand{
		Player:  c.name,
		Payload: payload,
		Type:    PlayerCommandTypeStart,
	}

	gh, err := c.hub.GetGameHub(payload.GameID)
	if err != nil {
		log.Println(err)
		return
	}

	gh.Commands <- gameCommand
}

func (c *Client) handlePlayerAnswer(payload PlayerCommandAnswer) {
	ga := GameAnswer{
		QuestionID: payload.QuestionID,
		Player:     c.name,
		Index:      payload.Index,
	}

	gh, err := c.hub.GetGameHub(payload.GameID)
	if err != nil {
		log.Println(err)
		return
	}

	gh.Answers <- ga
}

func (c *Client) writeMessage() {
	defer c.Conn.Close()
	for message := range c.Send {
		// if !ok {
		// 	c.conn.WriteMessage(websocket.CloseMessage, []byte{})
		// 	return
		// }
		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("error writing message to websocket: ERROR=%s. MESSAGE=%s, CLIENT=%+v\n", err, message, c)
		}
	}

	c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
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
	c.Conn = conn
	c.hub.register <- c
	pe := newPlayerEventConnect(c.name)
	c.hub.allBroadcast <- pe.toBytes()

	go c.readMessage()
	go c.writeMessage()
}

func (c *Client) Close() {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	log.Printf("player %s disconnect - client connection closed", c.name)
	c.hub.disconnect <- c
	pe := newPlayerEventDisconnect(c.name)
	c.hub.allBroadcast <- pe.toBytes()
	c.Conn.Close()
}
