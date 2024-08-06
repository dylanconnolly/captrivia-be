package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{}

// Client manages the websocket for a user and communicates with the ClientManager
type Client struct {
	name    string
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
	close   chan bool
	redis   *redis.Client
	gameHub *GameHub
}

// Creates a new client but does not attach websocket connection. Running serveWebsocket upgrades connection and begins
// begins running client
func newClient(name string, hub *Hub, rdb *redis.Client) *Client {
	c := &Client{
		name:  name,
		hub:   hub,
		send:  make(chan []byte, 256),
		close: make(chan bool),
		redis: rdb,
	}

	return c
}

func (c *Client) readMessage() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
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
	}

	// determine type of incoming message
	switch cmd.Type {
	case PlayerCommandTypeCreate:
		c.handleCreateGame(cmd)
	case PlayerCommandTypeJoin:
		c.handleJoinGame(cmd)
	case PlayerCommandTypeReady:
		c.handlePlayerReady(cmd)
	case PlayerCommandTypeStart:
		c.handleStartGame(cmd)
	case PlayerCommandTypeAnswer:
		c.handlePlayerAnswer(cmd)
	}
}

// func (c *Client) subscribe(gameID uuid.UUID) {
// 	pubsub := c.redis.Subscribe(context.Background(), gameID.String())
// 	c.sub = pubsub
// 	defer c.sub.Close()

// 	log.Printf("subscribing to redis pubsub %s", gameID)
// 	cancel := make(chan bool)
// 	ch := c.sub.Channel()

// 	for {
// 		select {
// 		case message := <-ch:
// 			fmt.Println("got channel message", message.Payload)
// 			c.send <- []byte(message.Payload)
// 		case <-cancel:
// 			log.Println("cancelling subscription")
// 			c.unsubscribe()
// 			return
// 		}
// 	}
// }

// func (c *Client) unsubscribe() {
// 	log.Println("cancelling subscription", c.sub.String())
// 	c.sub.Unsubscribe(context.Background())
// }

func (c *Client) handleCreateGame(cmd PlayerCommand) {
	var payload PlayerCommandCreate

	err := json.Unmarshal(cmd.Payload, &payload)
	if err != nil {
		log.Printf("error unmarshalling create game command payload: %s\n Client: %+v Command: %s", err, c, cmd)
		c.send <- []byte("could not parse command payload")
	}

	redisGame, err := c.hub.db.CreateGame(c.name, payload.Name, payload.QuestionCount)
	if err != nil {
		log.Printf("error writing to redis: %s", err)
		return
	}

	ge := newGameEventCreate(*redisGame)

	msg, err := json.Marshal(ge)
	if err != nil {
		log.Printf("error marshalling create game broadcast message: %s\n Client: %+v, Command: %s, GameEvent: %+v", err, c, cmd, ge)
		c.send <- []byte("there was an error creating game")
		return
	}
	c.hub.broadcast <- msg

	// add player that created the game to the game
	err = c.hub.db.AddPlayerToGame(redisGame.ID, c.name)
	if err != nil {
		return
	}
	captriviaGame, _ := c.hub.db.GetGame(redisGame.ID)

	// create gamehub to manage game state
	gh := NewGameHub(captriviaGame)
	go gh.Run()
	gameHubs[redisGame.ID] = gh

	enter := newGameEventPlayerEnter(c.name, captriviaGame)

	// assign client to gamehub
	c.gameHub = gh
	c.gameHub.register <- c

	msg, err = json.Marshal(enter)
	if err != nil {
		log.Printf("error marshalling create game broadcast message: %s\n Client: %+v, Command: %s, GameEvent: %+v", err, c, cmd, ge)
		c.send <- []byte("there was an error creating game")
		return
	}
	c.send <- msg
}

func (c *Client) handleJoinGame(cmd PlayerCommand) {
	var payload PlayerCommandJoin

	err := json.Unmarshal(cmd.Payload, &payload)
	if err != nil {
		log.Printf("error unmarshalling join game command payload: %s\n Client: %+v Command: %s", err, c, cmd)
		c.send <- []byte("could not parse command payload")
		return
	}

	err = c.hub.db.AddPlayerToGame(payload.GameID, c.name)
	if err != nil {
		log.Printf("error adding player to game: %s", err)
	}
	game, err := c.hub.db.GetGame(payload.GameID)
	log.Printf("game: %+v", game)
	if err != nil {
		log.Printf("error getting game: %s", err)
		return
	}

	geEnter := newGameEventPlayerEnter(c.name, game)

	// register gamehub
	c.gameHub = gameHubs[game.ID]
	c.gameHub.register <- c

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

	c.gameHub.broadcast <- msg
}

func (c *Client) handlePlayerReady(cmd PlayerCommand) {
	var payload PlayerCommandReady

	err := json.Unmarshal(cmd.Payload, &payload)
	if err != nil {
		log.Printf("error unmarshalling join game command payload: %s\n Client: %+v Command: %s", err, c, cmd)
		c.send <- []byte("could not parse command payload")
		return
	}

	err = c.hub.db.PlayerReady(payload.GameID, c.name)
	if err != nil {
		log.Printf("error readying player: %s", err)
	}

	ge := newGameEventPlayerReady(payload.GameID, c.name)
	msg, err := json.Marshal(ge)
	if err != nil {
		log.Printf("error marshalling player ready broadcast: %s\n Client: %+v Command: %s, GameEvent: %+v", err, c, cmd, ge)
		c.send <- []byte("there was an error marking yourself as ready")
		return
	}
	c.gameHub.broadcast <- msg
}

func (c *Client) handleStartGame(cmd PlayerCommand) {
	var payload PlayerCommandStart
	err := json.Unmarshal(cmd.Payload, &payload)
	if err != nil {
		log.Printf("error unmarshalling start game command payload: %s\n Client: %+v Command: %s", err, c, cmd)
		c.send <- []byte("could not parse command payload")
		return
	}

	// send GameEventStart
	ge := newGameEventStart(c.gameHub.id)
	msg, err := json.Marshal(ge)
	if err != nil {
		// log.Printf("error marshalling game start broadcast: %s\n Client: %+v Command: %s, GameEvent: %+v", err, c, cmd, ge)
		c.gameHub.broadcast <- []byte("there was an error starting game")
		return
	}
	c.gameHub.broadcast <- msg
	go c.gameHub.StartGame()
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

	c.gameHub.answers <- ga
	// correctIndex, err := c.hub.db.GetQuestionCorrectIndex(payload.QuestionID)
	// if err != nil {
	// 	log.Printf("error getting question's correct answer: %s", err)
	// }

	// var ge GameEvent
	// if payload.Index == correctIndex {
	// 	ge = newGameEventPlayerCorrect(payload.GameID, c.name, payload.QuestionID)
	// } else {
	// 	ge = newGameEventPlayerIncorrect(payload.GameID, c.name, payload.QuestionID)
	// }

	// msg, _ := json.Marshal(ge)
	// c.gameHub.broadcast <- msg
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
				log.Printf("error writing message to websocket. Client: %+v, message: %s", c, message)
			}
		}
	}
}

// upgrades connection to websocket on client and registers client with client Hub
func (c *Client) serveWebsocket(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return r.Header.Get("Origin") == "http://localhost:3000"
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error upgrading connection: %s\n", err)
	}
	c.conn = conn
	c.hub.register <- c

	go c.readMessage()
	go c.writeMessage()
}
