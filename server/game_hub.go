package server

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/google/uuid"
)

type GameHub struct {
	ID           uuid.UUID
	Answers      chan GameAnswer
	Broadcast    chan []byte
	Clients      map[*Client]bool
	Commands     chan GameLobbyCommand
	countdownSec int
	game         *captrivia.Game
	gameService  captrivia.GameService
	gameEnded    <-chan bool
	hubBroadcast chan<- GameEvent // send only channel to push GameEvents to Hub
	mu           sync.Mutex
	Register     chan *Client
	Unregister   chan *Client
	questionSec  int
}

type GameAnswer struct {
	QuestionID string
	Player     string
	Index      int
}

func NewGameHub(g *captrivia.Game, gameService captrivia.GameService, hubBroadcast chan<- GameEvent, countdownSec int, questionSec int) *GameHub {
	return &GameHub{
		ID:           g.ID,
		Answers:      make(chan GameAnswer),
		Broadcast:    make(chan []byte, 256),
		Clients:      make(map[*Client]bool),
		Commands:     make(chan GameLobbyCommand),
		countdownSec: countdownSec,
		game:         g,
		gameService:  gameService,
		gameEnded:    g.GameEndedChan(),
		hubBroadcast: hubBroadcast,
		Register:     make(chan *Client, 5),
		questionSec:  questionSec,
		Unregister:   make(chan *Client, 5),
	}
}

// Run() handles client connections and message directives such
// as register, unregister, command, broadcast
func (g *GameHub) Run(ctx context.Context) {
	done := make(chan bool, 1)
	for {
		select {
		case client := <-g.Register:
			g.mu.Lock()
			g.Clients[client] = true
			g.mu.Unlock()
			client.gameHub = g
			go g.playerJoin(client)
		case client := <-g.Unregister:
			go g.playerLeave(client)
		case message := <-g.Broadcast:
			// broadcasts message to all clients that are part of the GameHub
			g.mu.Lock()
			for client := range g.Clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(g.Clients, client)
				}
			}
			g.mu.Unlock()

		case command := <-g.Commands:
			// commands channel listens for lobby commands (Ready, Start, Leave) issued by player clients
			var event GameEvent
			switch command.Type {
			case PlayerCommandTypeReady:
				event = newGameEventPlayerReady(command.Payload.GameID, command.Player)
				g.game.PlayerReady(command.Player)
				go g.gameService.SaveGame(g.game)
			case PlayerCommandTypeStart:
				event = newGameEventStart(command.Payload.GameID)

				go g.RunGame(done)
			}
			g.Broadcast <- event.toBytes()

		case <-done:
			err := g.gameService.ExpireGame(g.ID)

			if err != nil {
				log.Printf("error setting game to expire %s . GameID=%s", err, g.game.ID)
			}

			// re-register clients with Hub to recieve game creation/state updates and remove from GameHub clients
			g.mu.Lock()
			for client := range g.Clients {
				client.hub.register <- client
				delete(g.Clients, client)
			}
			g.mu.Unlock()
			return
		}
	}
}

// helper function to add player to Game and generate PlayerEnter + PlayerJoin
// GameEvents to be broadcast to the game lobby
func (g *GameHub) playerJoin(client *Client) {
	g.mu.Lock()
	g.game.AddPlayer(client.name)
	g.gameService.SaveGame(g.game)

	enterEvent := newGameEventPlayerEnter(client.name, g.game)
	client.send <- enterEvent.toBytes()

	// unregister player from hub broadcasts
	client.hub.unregister <- client

	joinEvent := newGameEventPlayerJoin(g.game.ID, client.name)
	g.Broadcast <- joinEvent.toBytes()

	playerCountEvent := newGameEventPlayerCount(g.game.ID, g.game.PlayerCount)
	g.hubBroadcast <- playerCountEvent
	g.mu.Unlock()
}

// Helper function to remove a player from GameHub + Game, and re-register
// the client to the Hub
func (g *GameHub) playerLeave(client *Client) {
	g.mu.Lock()
	delete(g.Clients, client)
	g.game.RemovePlayer(client.name)
	g.gameService.SaveGame(g.game)

	playerCountEvent := newGameEventPlayerCount(g.game.ID, g.game.PlayerCount)
	g.hubBroadcast <- playerCountEvent

	leaveEvent := newGameEventPlayerLeave(g.game.ID, client.name)
	g.Broadcast <- leaveEvent.toBytes()
	g.mu.Unlock()
}

// Runs the main trivia game loop. Listens for answers from client and handles
// the tickers used for countdowns and question durations
func (g *GameHub) RunGame(done chan<- bool) {
	countdownEvent := newGameEventCountdown(g.game.ID, g.countdownSec)
	g.Broadcast <- countdownEvent.toBytes()
	countdownDuration := time.Duration(g.countdownSec) * time.Second
	questionDuration := time.Duration(g.questionSec) * time.Second

	countdownTicker := time.NewTicker(countdownDuration)
	questionTicker := time.NewTicker(questionDuration)

	g.ChangeGameState(captrivia.GameStateCountdown)

	defer questionTicker.Stop()
	defer countdownTicker.Stop()

	for {
		select {
		case <-countdownTicker.C: // countdown has completed, display question
			countdownTicker.Stop()
			g.handleDisplayQuestion()
			questionTicker = time.NewTicker(questionDuration)

		case <-questionTicker.C: // time expired before a correct answer was provided
			questionTicker.Stop()
			g.handleQuestionTimeExpired()
			g.Broadcast <- countdownEvent.toBytes()
			countdownTicker = time.NewTicker(countdownDuration)

		case ans := <-g.Answers: // player has answered the question
			correct := g.game.ValidateAnswer(ans.Index)

			if correct {
				questionTicker.Stop()

				g.game.IncrementPlayerScore(ans.Player)
				event := newGameEventPlayerCorrect(g.game.ID, ans.Player, ans.QuestionID)
				g.Broadcast <- event.toBytes()

				g.game.GoToNextQuestion()

				g.Broadcast <- countdownEvent.toBytes()
				countdownTicker = time.NewTicker(countdownDuration)

				g.ChangeGameState(captrivia.GameStateCountdown)
			} else {
				event := newGameEventPlayerIncorrect(g.game.ID, ans.Player, ans.QuestionID)
				g.Broadcast <- event.toBytes()
			}

		case <-g.gameEnded:
			gameEndEvent := newGameEventEnd(g.game.ID, g.game.PlayerScores())
			g.Broadcast <- gameEndEvent.toBytes()
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
	questionEvent := newGameEventQuestion(g.game.ID, q, g.questionSec)
	g.Broadcast <- questionEvent.toBytes()

	g.ChangeGameState(captrivia.GameStateQuestion)
}

// helper function used when a question has reached its duration and the correct
// answer was not provided.
func (g *GameHub) handleQuestionTimeExpired() {
	g.game.GoToNextQuestion()
	g.ChangeGameState(captrivia.GameStateCountdown)
}
