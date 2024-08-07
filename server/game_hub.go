package server

import (
	"log"
	"time"

	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/google/uuid"
)

var (
	gameHubs                              = make(map[uuid.UUID]*GameHub)
	questionTickerDuration  time.Duration = (time.Duration(questionDuration) * time.Second)
	countdownTickerDuration time.Duration = (time.Duration(questionCountdown) * time.Second)
)

const (
	questionCountdown int = 5
	questionDuration  int = 20
)

type GameHub struct {
	id               uuid.UUID
	broadcast        chan []byte
	clients          map[*Client]bool
	commands         chan GameLobbyCommand
	countdown        int
	game             *captrivia.Game
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

func NewGameHub(g *captrivia.Game) *GameHub {
	return &GameHub{
		id:               g.ID,
		answers:          make(chan GameAnswer),
		broadcast:        make(chan []byte, 1),
		clients:          make(map[*Client]bool),
		commands:         make(chan GameLobbyCommand),
		countdown:        questionCountdown,
		game:             g,
		gameEnded:        make(chan bool, 1),
		register:         make(chan *Client),
		questionDuration: questionDuration,
		unregister:       make(chan *Client),
	}
}

func (g *GameHub) Run(emitToHub chan<- GameEvent) {
	done := make(chan bool, 1)
	for {
		select {
		case client := <-g.register:
			g.clients[client] = true

			g.playerJoin(client, emitToHub)
		case client := <-g.unregister:
			delete(g.clients, client)

			g.playerLeave(client, emitToHub)
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
		// commands channel listens for lobby commands (Ready, Start, Leave) issued by any player clients
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
		case <-done:
			log.Println("stopping GameHub Run() routine")
			return
		}
	}
}

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
			} else {
				event := newGameEventPlayerIncorrect(g.game.ID, ans.Player, ans.QuestionID)
				g.broadcast <- event.toBytes()
			}
		case <-g.gameEnded:
			log.Println("end of game, reached last question")
			gameEndEvent := newGameEventEnd(g.game.ID, g.game.PlayerScores())
			g.broadcast <- gameEndEvent.toBytes()
			g.game.State = captrivia.GameStateEnded
			emitToHub <- newGameEventStateChange(g.game.ID, g.game.State)
			done <- true
			return
		}
	}
}

func (g *GameHub) playerJoin(client *Client, emit chan<- GameEvent) {
	g.game.AddPlayer(client.name)

	enterEvent := newGameEventPlayerEnter(client.name, g.game)
	client.send <- enterEvent.toBytes()

	joinEvent := newGameEventPlayerJoin(g.game.ID, client.name)
	g.broadcast <- joinEvent.toBytes()

	playerCountEvent := newGameEventPlayerCount(g.game.ID, g.game.PlayerCount)
	emit <- playerCountEvent
}

func (g *GameHub) playerLeave(client *Client, emit chan<- GameEvent) {
	g.game.RemovePlayer(client.name)

	leaveEvent := newGameEventPlayerLeave(g.game.ID, client.name)
	g.broadcast <- leaveEvent.toBytes()

	// re-register client to Hub to recieve updates on games
	client.hub.register <- client

	playerCountEvent := newGameEventPlayerCount(g.game.ID, g.game.PlayerCount)
	emit <- playerCountEvent
}

// helper function used to get current game question, create GameEvent to display
// question to users, and emit game state change to Hub
func (g *GameHub) handleDisplayQuestion(emit chan<- GameEvent) {
	q := g.game.CurrentQuestion()
	questionEvent := newGameEventQuestion(g.game.ID, q, questionDuration)
	g.broadcast <- questionEvent.toBytes()

	g.game.State = captrivia.GameStateQuestion
	emit <- newGameEventStateChange(g.game.ID, g.game.State)
}

// helper function used when a question has reached its duration and the correct
// answer was not provided.
func (g *GameHub) handleQuestionTimeExpired(emit chan<- GameEvent) {
	g.game.GoToNextQuestion()
	g.game.State = captrivia.GameStateCountdown
	emit <- newGameEventStateChange(g.game.ID, g.game.State)
}

func (g *GameHub) handleCorrectAnswer(emit chan<- GameEvent, ans GameAnswer, countdownEvent GameEvent) {
	g.game.IncrementPlayerScore(ans.Player)
	event := newGameEventPlayerCorrect(g.game.ID, ans.Player, ans.QuestionID)
	g.broadcast <- event.toBytes()

	g.game.GoToNextQuestion()

	g.broadcast <- countdownEvent.toBytes()
	g.game.State = captrivia.GameStateCountdown
	emit <- newGameEventStateChange(g.game.ID, g.game.State)
}
