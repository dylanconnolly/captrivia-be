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

func (g *GameHub) Run() {
	for {
		select {
		case client := <-g.register:
			g.clients[client] = true

			g.game.AddPlayer(client.name)

			enterEvent := newGameEventPlayerEnter(client.name, g.game)
			client.send <- enterEvent.toBytes()

			joinEvent := newGameEventPlayerJoin(g.game.ID, client.name)
			g.broadcast <- joinEvent.toBytes()
		case client := <-g.unregister:
			g.game.RemovePlayer(client.name)

			leaveEvent := newGameEventPlayerLeave(g.game.ID, client.name)
			g.broadcast <- leaveEvent.toBytes()

			delete(g.clients, client)
			close(client.send)
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
				go g.StartGame()
			}
			log.Println(event.Type)
			g.broadcast <- event.toBytes()
		}
	}
}

func (g *GameHub) StartGame() {
	g.game.AttachGameEnded(g.gameEnded)
	// var questionTicker *time.Ticker
	countdownEvent := newGameEventCountdown(g.game.ID, g.countdown)
	g.broadcast <- countdownEvent.toBytes()
	countdownTicker := time.NewTicker(countdownTickerDuration)
	questionTicker := time.NewTicker(questionTickerDuration)

	defer questionTicker.Stop()
	defer countdownTicker.Stop()

	for {
		select {
		// countdown has completed, display question
		case <-countdownTicker.C:
			countdownTicker.Stop()
			q := g.game.CurrentQuestion()
			questionEvent := newGameEventQuestion(g.game.ID, q, questionDuration)
			g.broadcast <- questionEvent.toBytes()
			questionTicker = time.NewTicker(questionTickerDuration)
		// time expired before a correct answer was provided
		case <-questionTicker.C:
			questionTicker.Stop()

			g.game.GoToNextQuestion()

			g.broadcast <- countdownEvent.toBytes()
			countdownTicker = time.NewTicker(countdownTickerDuration)
		case ans := <-g.answers:
			var event GameEvent

			correct := g.game.ValidateAnswer(ans.Index)
			if correct {
				questionTicker.Stop()
				g.game.IncrementPlayerScore(ans.Player)
				event = newGameEventPlayerCorrect(g.game.ID, ans.Player, ans.QuestionID)
				g.broadcast <- event.toBytes()

				g.game.GoToNextQuestion()

				g.broadcast <- countdownEvent.toBytes()

				countdownTicker = time.NewTicker(countdownTickerDuration)
			} else {
				event = newGameEventPlayerIncorrect(g.game.ID, ans.Player, ans.QuestionID)
				g.broadcast <- event.toBytes()
			}
		case <-g.gameEnded:
			log.Println("end of game, reached last question")
			gameEndEvent := newGameEventEnd(g.game.ID, g.game.PlayerScores())
			g.broadcast <- gameEndEvent.toBytes()
			return
		}
	}
}
