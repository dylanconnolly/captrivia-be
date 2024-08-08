package server

import (
	"context"
	"log"
	"time"

	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/dylanconnolly/captrivia-be/redis"
	"github.com/google/uuid"
)

var (
	GameHubs                              = make(map[uuid.UUID]*GameHub)
	questionTickerDuration  time.Duration = (time.Duration(questionDuration) * time.Second)
	countdownTickerDuration time.Duration = (time.Duration(questionCountdown) * time.Second)
)

const (
	questionCountdown int = 5
	questionDuration  int = 20
)

type GameHub struct {
	ID        uuid.UUID
	broadcast chan []byte
	clients   map[*Client]bool
	commands  chan GameLobbyCommand
	countdown int
	// disconnects      chan *Client
	game             *captrivia.Game
	gameService      captrivia.GameService
	gameEnded        <-chan bool
	hubBroadcast     chan<- GameEvent // send only channel to push GameEvents to Hub
	register         chan *Client
	unregister       chan *Client
	questionDuration int
	answers          chan GameAnswer
}

type GameAnswer struct {
	QuestionID string
	Player     string
	Index      int
}

func NewGameHub(g *captrivia.Game, gameService captrivia.GameService, hubBroadcast chan<- GameEvent) *GameHub {
	return &GameHub{
		ID:        g.ID,
		answers:   make(chan GameAnswer),
		broadcast: make(chan []byte, 256), //TODO unbuffered with goroutines?
		clients:   make(map[*Client]bool),
		commands:  make(chan GameLobbyCommand),
		countdown: questionCountdown,
		// disconnects:      make(chan *Client),
		game:             g,
		gameService:      gameService,
		gameEnded:        g.GameEndedChan(),
		hubBroadcast:     hubBroadcast,
		register:         make(chan *Client, 5), //TODO unbuffered with goroutines?
		questionDuration: questionDuration,
		unregister:       make(chan *Client, 5), //TODO unbuffered with goroutines?
	}
}

// Run() handles client connections and message directives such
// as register, unregister, command, broadcast
func (g *GameHub) Run(ctx context.Context) {
	done := make(chan bool, 1)
	for {
		select {
		case client := <-g.register:
			g.clients[client] = true
			client.gameHub = g
			go g.playerJoin(client)
		case client := <-g.unregister:
			go g.playerLeave(client)
		case message := <-g.broadcast:
			// broadcasts message to all clients that are part of the GameHub
			for client := range g.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(g.clients, client)
				}
			}

		case command := <-g.commands:
			// commands channel listens for lobby commands (Ready, Start, Leave) issued by player clients
			var event GameEvent
			switch command.Type {
			case PlayerCommandTypeReady:
				event = newGameEventPlayerReady(command.payload.GameID, command.player)
				g.game.PlayerReady(command.player)
				go g.gameService.SaveGame(g.game)
			case PlayerCommandTypeStart:
				event = newGameEventStart(command.payload.GameID)

				go g.runGame(done)
			}
			g.broadcast <- event.toBytes()

		case <-done:
			log.Printf("Setting game TTL in Redis to %d minutes", redis.EndedGameTTL)
			err := g.gameService.ExpireGame(g.ID)

			if err != nil {
				log.Printf("error expiring game %s. GameID=%s", err, g.game.ID)
			}
			log.Printf("GameHub game (%s) completed.", g.game.ID)

			// re-register clients with Hub to recieve game creation/state updates and remove from GameHub clients
			for client := range g.clients {
				client.hub.register <- client
				delete(g.clients, client)
			}
			return
		}
	}
}

// helper function to add player to Game and generate PlayerEnter + PlayerJoin
// GameEvents to be broadcast to the game lobby
func (g *GameHub) playerJoin(client *Client) {
	g.game.AddPlayer(client.name)
	g.gameService.SaveGame(g.game)

	enterEvent := newGameEventPlayerEnter(client.name, g.game)
	client.send <- enterEvent.toBytes()

	// unregister player from hub broadcasts
	client.hub.unregister <- client

	joinEvent := newGameEventPlayerJoin(g.game.ID, client.name)
	g.broadcast <- joinEvent.toBytes()

	playerCountEvent := newGameEventPlayerCount(g.game.ID, g.game.PlayerCount)
	g.hubBroadcast <- playerCountEvent
}

// Helper function to remove a player from GameHub + Game, and re-register
// the client to the Hub
func (g *GameHub) playerLeave(client *Client) {
	delete(g.clients, client)
	g.game.RemovePlayer(client.name)
	g.gameService.SaveGame(g.game)

	playerCountEvent := newGameEventPlayerCount(g.game.ID, g.game.PlayerCount)
	g.hubBroadcast <- playerCountEvent

	leaveEvent := newGameEventPlayerLeave(g.game.ID, client.name)
	g.broadcast <- leaveEvent.toBytes()
}

// Runs the main trivia game loop. Listens for answers from client and handles
// the tickers used for countdowns and question durations
func (g *GameHub) runGame(done chan<- bool) {
	countdownEvent := newGameEventCountdown(g.game.ID, g.countdown)
	g.broadcast <- countdownEvent.toBytes()
	countdownTicker := time.NewTicker(countdownTickerDuration)
	questionTicker := time.NewTicker(questionTickerDuration)

	g.ChangeGameState(captrivia.GameStateCountdown)

	defer questionTicker.Stop()
	defer countdownTicker.Stop()

	for {
		select {
		case <-countdownTicker.C: // countdown has completed, display question
			countdownTicker.Stop()
			g.handleDisplayQuestion()
			questionTicker = time.NewTicker(questionTickerDuration)

		case <-questionTicker.C: // time expired before a correct answer was provided
			questionTicker.Stop()
			g.handleQuestionTimeExpired()
			g.broadcast <- countdownEvent.toBytes()
			countdownTicker = time.NewTicker(countdownTickerDuration)

		case ans := <-g.answers: // player has answered the question
			correct := g.game.ValidateAnswer(ans.Index)

			if correct {
				questionTicker.Stop()

				g.game.IncrementPlayerScore(ans.Player)
				event := newGameEventPlayerCorrect(g.game.ID, ans.Player, ans.QuestionID)
				g.broadcast <- event.toBytes()

				g.game.GoToNextQuestion()

				g.broadcast <- countdownEvent.toBytes()
				countdownTicker = time.NewTicker(countdownTickerDuration)

				g.ChangeGameState(captrivia.GameStateCountdown)
			} else {
				event := newGameEventPlayerIncorrect(g.game.ID, ans.Player, ans.QuestionID)
				g.broadcast <- event.toBytes()
			}

		case <-g.gameEnded:
			gameEndEvent := newGameEventEnd(g.game.ID, g.game.PlayerScores())
			g.broadcast <- gameEndEvent.toBytes()
			g.ChangeGameState(captrivia.GameStateEnded)
			done <- true
			return
		}
	}
}

func (g *GameHub) ChangeGameState(state captrivia.GameState) {
	g.game.State = state
	g.gameService.SaveGame(g.game)
	g.hubBroadcast <- newGameEventStateChange(g.game.ID, g.game.State)
}

// helper function used to get current game question, create GameEvent to display
// question to users, and emit game state change to Hub
func (g *GameHub) handleDisplayQuestion() {
	q := g.game.CurrentQuestion()
	questionEvent := newGameEventQuestion(g.game.ID, q, questionDuration)
	g.broadcast <- questionEvent.toBytes()

	g.ChangeGameState(captrivia.GameStateQuestion)
}

// helper function used when a question has reached its duration and the correct
// answer was not provided.
func (g *GameHub) handleQuestionTimeExpired() {
	g.game.GoToNextQuestion()
	g.ChangeGameState(captrivia.GameStateCountdown)
}
