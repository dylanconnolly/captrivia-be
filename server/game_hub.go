package server

import (
	"log"

	"github.com/dylanconnolly/captrivia-be/captrivia"
	"github.com/google/uuid"
)

var gameHubs = make(map[uuid.UUID]*GameHub)

const (
	gameCountdown    int = 10
	questionDuration int = 20
)

type GameHub struct {
	id               uuid.UUID
	broadcast        chan []byte
	clients          map[*Client]bool
	commands         chan GameLobbyCommand
	countdown        int
	register         chan *Client
	unregister       chan *Client
	questionDuration int
	answers          chan GameAnswer
	game             *captrivia.Game
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
		countdown:        gameCountdown,
		game:             g,
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
		case message := <-g.broadcast:
			for client := range g.clients {
				select {
				case client.send <- message:
					log.Println("got message broadcast")
				default:
					close(client.send)
					delete(g.clients, client)
				}
			}
		case command := <-g.commands:
			log.Println("got a game command")

			log.Printf("payload: %+v", command.payload)

			switch command.Type {
			case PlayerCommandTypeReady:
				readyEvent := newGameEventPlayerReady(command.payload.GameID, command.player)
				g.broadcast <- readyEvent.toBytes()
			}
		}
	}
}

// type GameHub struct {
// 	id                   uuid.UUID
// 	broadcast            chan []byte
// 	clients              map[*Client]bool
// 	countdown            int
// 	db                   *redis.DB
// 	register             chan *Client
// 	unregister           chan *Client
// 	questionDuration     int
// 	state                string
// 	Mutex                sync.Mutex
// 	answers              chan GameAnswer
// 	game                 *Game
// 	Answered             bool
// 	Questions            []captrivia.Question
// 	CurrentQuestionIndex int
// 	Scores               map[string]int
// }

// func NewGameHub(g *captrivia.Game) *GameHub {
// 	gh := &GameHub{
// 		id:               g.ID,
// 		broadcast:        make(chan []byte),
// 		clients:          make(map[*Client]bool),
// 		countdown:        gameCountdown,
// 		db:               redis.NewDB(),
// 		register:         make(chan *Client),
// 		unregister:       make(chan *Client),
// 		questionDuration: questionDuration,
// 		state:            captrivia.GameStateWaiting,
// 		answers:          make(chan GameAnswer),
// 		Answered:         false,
// 		Scores:           make(map[string]int),
// 	}

// 	return gh
// }

// func (g *GameHub) LoadQuestions() {
// 	questions, err := g.db.GetGameQuestions(g.id)
// 	if err != nil {
// 		log.Printf("error getting questions for game: %s", err)
// 	}
// 	log.Printf("questions: %+v", questions)
// 	g.Questions = questions
// }

// func (g *GameHub) GeneratePlayerScores() []PlayerScore {
// 	var scores []PlayerScore
// 	for client := range g.clients {
// 		var ps PlayerScore
// 		if score, ok := g.Scores[client.name]; ok {
// 			ps.Name = client.name
// 			ps.Score = score
// 		} else {
// 			ps.Name = client.name
// 			score = 0
// 		}

// 		scores = append(scores, ps)
// 	}

// 	return scores
// }

// type PlayerScore struct {
// 	Name  string `json:"name"`
// 	Score int    `json:"score"`
// }

// type GameAnswer struct {
// 	QuestionID string
// 	Player     string
// 	Index      int
// }

// func (g *GameHub) Run() {
// 	for {
// 		select {
// 		case client := <-g.register:
// 			g.clients[client] = true
// 		case client := <-g.unregister:
// 			delete(g.clients, client)
// 			// g.db.RemovePlayerFromCreatedGames(client.name)
// 			close(client.send)
// 		case message := <-g.broadcast:
// 			for client := range g.clients {
// 				select {
// 				case client.send <- message:
// 				default:
// 					close(client.send)
// 					delete(g.clients, client)
// 				}
// 			}
// 		}
// 	}
// }

// func (g *GameHub) startCountdown() {
// 	ge := newGameEventCountdown(g.id, g.countdown)
// 	msg, err := json.Marshal(ge)
// 	if err != nil {
// 		log.Println("error starting game")
// 	}
// 	g.broadcast <- msg
// }

// func (g *GameHub) displayQuestion() {
// 	g.Mutex.Lock()
// 	defer g.Mutex.Unlock()
// 	g.Answered = false
// 	if g.CurrentQuestionIndex > (len(g.Questions) - 1) {
// 		ge := newGameEventEnd(g.id, g.GeneratePlayerScores())
// 		msg, _ := json.Marshal(ge)
// 		g.broadcast <- msg
// 		return
// 	}
// 	question := g.Questions[g.CurrentQuestionIndex]
// 	ge := newGameEventQuestion(g.id, &question, g.questionDuration)
// 	msg, _ := json.Marshal(ge)
// 	g.broadcast <- msg
// }

// func (g *GameHub) StartGame() {
// 	g.LoadQuestions()
// 	if len(g.Questions) < 1 {
// 		log.Println("failed to load questions into gamehub")
// 		return
// 	}
// 	g.Answered = false
// 	g.startCountdown()
// 	questionTicker := time.NewTicker(time.Duration(g.questionDuration) * time.Second)
// 	countdownTicker := time.NewTicker(time.Duration(g.countdown) * time.Second)
// 	defer questionTicker.Stop()
// 	defer countdownTicker.Stop()

// 	for {
// 		select {
// 		case ans := <-g.answers:
// 			g.Mutex.Lock()
// 			if g.Answered {
// 				g.Mutex.Unlock()
// 				continue
// 			}

// 			var ge GameEvent
// 			if ans.Index == g.Questions[g.CurrentQuestionIndex].CorrectIndex {
// 				ge = newGameEventPlayerCorrect(g.id, ans.Player, ans.QuestionID)
// 				g.Answered = true
// 				questionTicker.Stop()
// 				msg, _ := json.Marshal(ge)
// 				g.broadcast <- msg
// 				g.CurrentQuestionIndex += 1
// 				g.Mutex.Unlock()
// 				g.startCountdown()
// 				countdownTicker = time.NewTicker(time.Duration(g.countdown) * time.Second)
// 			} else {
// 				ge = newGameEventPlayerIncorrect(g.id, ans.Player, ans.QuestionID)
// 				msg, _ := json.Marshal(ge)
// 				g.broadcast <- msg
// 				g.Mutex.Unlock()
// 			}

// 		case <-countdownTicker.C:
// 			countdownTicker.Stop()
// 			g.displayQuestion()
// 			questionTicker = time.NewTicker(time.Duration(g.questionDuration) * time.Second)
// 		case <-questionTicker.C:
// 			g.Mutex.Lock()
// 			questionTicker.Stop()
// 			g.CurrentQuestionIndex += 1
// 			g.Mutex.Unlock()
// 			g.startCountdown()
// 			countdownTicker = time.NewTicker(time.Duration(g.countdown) * time.Second)
// 		}
// 	}
// }
