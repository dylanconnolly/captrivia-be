package server

import (
	"context"
	"log"
	"time"

	"github.com/dylanconnolly/captrivia-be/captrivia"
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
	ID               uuid.UUID
	broadcast        chan []byte
	clients          map[*Client]bool
	commands         chan GameLobbyCommand
	countdown        int
	game             *captrivia.Game
	gameService      captrivia.GameService
	gameEnded        chan bool
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

func NewGameHub(g *captrivia.Game, gameService captrivia.GameService) *GameHub {
	return &GameHub{
		ID:               g.ID,
		answers:          make(chan GameAnswer),
		broadcast:        make(chan []byte, 1),
		clients:          make(map[*Client]bool),
		commands:         make(chan GameLobbyCommand),
		countdown:        questionCountdown,
		game:             g,
		gameService:      gameService,
		gameEnded:        make(chan bool, 1),
		register:         make(chan *Client),
		questionDuration: questionDuration,
		unregister:       make(chan *Client),
	}
}

// Run() handles client connections and message directives such
// as register, unregister, command, broadcast
func (g *GameHub) Run(ctx context.Context, emitToHub chan<- GameEvent) {
	done := make(chan bool, 1)
	for {
		select {
		case client := <-g.register:
			g.RegisterClient(client)

			g.playerJoin(client, emitToHub)

			g.gameService.SaveGame(g.game)
		case client := <-g.unregister:
			delete(g.clients, client)

			g.playerLeave(client, emitToHub)

			g.gameService.SaveGame(g.game)
		// broadcasts message to all clients that are part of the GameHub
		case message := <-g.broadcast:
			for client := range g.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(g.clients, client)
				}
			}
		// commands channel listens for lobby commands (Ready, Start, Leave) issued by player clients
		case command := <-g.commands:
			var event GameEvent
			switch command.Type {
			case PlayerCommandTypeReady:
				event = newGameEventPlayerReady(command.payload.GameID, command.player)
			case PlayerCommandTypeStart:
				event = newGameEventStart(command.payload.GameID)

				go g.runGame(done, emitToHub)
			}
			g.broadcast <- event.toBytes()
		case <-ctx.Done():
			g.gameEnded <- true
			log.Println("stopping GameHub Run()")
			return
		case <-done:
			log.Println("stopping GameHub Run() routine")
			return
		}
	}
}

// helper function to add player to Game and generate PlayerEnter + PlayerJoin
// GameEvents to be broadcast to the game lobby
func (g *GameHub) playerJoin(client *Client, emit chan<- GameEvent) {
	g.game.AddPlayer(client.name)

	enterEvent := newGameEventPlayerEnter(client.name, g.game)
	client.send <- enterEvent.toBytes()

	joinEvent := newGameEventPlayerJoin(g.game.ID, client.name)
	g.broadcast <- joinEvent.toBytes()

	playerCountEvent := newGameEventPlayerCount(g.game.ID, g.game.PlayerCount)
	emit <- playerCountEvent
}

// Helper function to remove a player from GameHub + Game, and re-register
// the client to the Hub
func (g *GameHub) playerLeave(client *Client, emit chan<- GameEvent) {
	g.game.RemovePlayer(client.name)

	leaveEvent := newGameEventPlayerLeave(g.game.ID, client.name)
	g.broadcast <- leaveEvent.toBytes()

	// re-register client to Hub to recieve updates on games
	client.hub.register <- client

	playerCountEvent := newGameEventPlayerCount(g.game.ID, g.game.PlayerCount)
	emit <- playerCountEvent
}

// Runs the main trivia game loop. Listens for answers from client and handles
// the tickers used for countdowns and question durations
func (g *GameHub) runGame(done chan<- bool, emitToHub chan<- GameEvent) {
	g.game.AttachGameEnded(g.gameEnded)

	countdownEvent := newGameEventCountdown(g.game.ID, g.countdown)
	g.broadcast <- countdownEvent.toBytes()
	countdownTicker := time.NewTicker(countdownTickerDuration)
	questionTicker := time.NewTicker(questionTickerDuration)

	g.game.State = captrivia.GameStateCountdown
	emitToHub <- newGameEventStateChange(g.game.ID, g.game.State)

	defer questionTicker.Stop()
	defer countdownTicker.Stop()

	for {
		select {
		// countdown has completed, display question
		case <-countdownTicker.C:
			countdownTicker.Stop()
			g.handleDisplayQuestion(emitToHub)
			questionTicker = time.NewTicker(questionTickerDuration)
		// time expired before a correct answer was provided
		case <-questionTicker.C:
			questionTicker.Stop()
			g.handleQuestionTimeExpired(emitToHub)
			g.broadcast <- countdownEvent.toBytes()
			countdownTicker = time.NewTicker(countdownTickerDuration)
		// player has answered the question
		case ans := <-g.answers:
			correct := g.game.ValidateAnswer(ans.Index)

			if correct {
				questionTicker.Stop()

				g.game.IncrementPlayerScore(ans.Player)
				event := newGameEventPlayerCorrect(g.game.ID, ans.Player, ans.QuestionID)
				g.broadcast <- event.toBytes()

				g.game.GoToNextQuestion()

				g.broadcast <- countdownEvent.toBytes()
				countdownTicker = time.NewTicker(countdownTickerDuration)

				g.game.State = captrivia.GameStateCountdown
				emitToHub <- newGameEventStateChange(g.game.ID, g.game.State)
				g.gameService.SaveGame(g.game)
			} else {
				event := newGameEventPlayerIncorrect(g.game.ID, ans.Player, ans.QuestionID)
				g.broadcast <- event.toBytes()
			}
		case <-g.gameEnded:
			log.Println("end of game, no more questions")
			gameEndEvent := newGameEventEnd(g.game.ID, g.game.PlayerScores())
			g.broadcast <- gameEndEvent.toBytes()
			g.game.State = captrivia.GameStateEnded
			emitToHub <- newGameEventStateChange(g.game.ID, g.game.State)
			g.gameService.SaveGame(g.game)
			done <- true
			return
		}
	}
}

// helper function used to get current game question, create GameEvent to display
// question to users, and emit game state change to Hub
func (g *GameHub) handleDisplayQuestion(emit chan<- GameEvent) {
	q := g.game.CurrentQuestion()
	questionEvent := newGameEventQuestion(g.game.ID, q, questionDuration)
	g.broadcast <- questionEvent.toBytes()

	g.game.State = captrivia.GameStateQuestion
	emit <- newGameEventStateChange(g.game.ID, g.game.State)
	g.gameService.SaveGame(g.game)
}

// helper function used when a question has reached its duration and the correct
// answer was not provided.
func (g *GameHub) handleQuestionTimeExpired(emit chan<- GameEvent) {
	g.game.GoToNextQuestion()
	g.game.State = captrivia.GameStateCountdown
	emit <- newGameEventStateChange(g.game.ID, g.game.State)
	g.gameService.SaveGame(g.game)
}

func (g *GameHub) RegisterClient(c *Client) {
	g.clients[c] = true
}
