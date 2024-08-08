package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/dylanconnolly/captrivia-be/redis"
	"github.com/dylanconnolly/captrivia-be/server"
)

var (
	listen string
)

func main() {
	cfg := NewConfig()

	flag.StringVar(&listen, "listen", ":8080", "Listen address")
	flag.Parse()

	ctx, _ := context.WithCancel(context.Background())

	app := NewApp(cfg)

	go app.hub.Run(ctx)

	log.Println("listening on ", app.httpServer.Addr)
	err := app.httpServer.ListenAndServe()
	if err != nil {
		log.Fatalf("failed to listen: %s", err)
	}
}

type App struct {
	hub        *server.Hub
	gameServer *server.GameServer
	httpServer *http.Server
}

func NewApp(cfg Config) *App {
	hub := server.NewHub(redis.NewGameService(cfg.RedisAddr, cfg.RedisTTL), cfg.CountdownDuration, cfg.QuestionDuration)
	gameServer := server.NewGameServer(hub)
	httpServer := server.NewHTTPServer(listen, gameServer)

	return &App{
		hub:        hub,
		gameServer: gameServer,
		httpServer: httpServer,
	}
}

type Config struct {
	RedisAddr         string
	RedisTTL          int
	CountdownDuration int
	QuestionDuration  int
}

func NewConfig() Config {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	ttl := os.Getenv("REDIS_TTL_SEC")
	if ttl == "" {
		ttl = "300"
	}
	cd := os.Getenv("COUNTDOWN_DURATION_SEC")
	if cd == "" {
		cd = "5"
	}
	qd := os.Getenv("QUESTION_DURATION_SEC")
	if qd == "" {
		qd = "5"
	}
	questions_path := os.Getenv("QUESTIONS_FILE_PATH")
	if questions_path == "" {
		log.Fatal("QUESTIONS_FILE_PATH env variable not found. Please provide full path to questions.json")
	}

	ttlInt, err := strconv.Atoi(ttl)
	if err != nil {
		log.Fatal("error converting env variable REDIS_TTL to integer ", err)
	}
	cdInt, err := strconv.Atoi(cd)
	if err != nil {
		log.Fatal("error converting env variable COUNTDOWN_DURATION_SEC to integer ", err)
	}
	qdInt, err := strconv.Atoi(qd)
	if err != nil {
		log.Fatal("error converting env variable QUESTION_DURATION_SEC to integer ", err)
	}

	cfg := Config{
		RedisAddr:         addr,
		RedisTTL:          ttlInt,
		CountdownDuration: cdInt,
		QuestionDuration:  qdInt,
	}

	return cfg
}
