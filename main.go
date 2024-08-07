package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/dylanconnolly/captrivia-be/redis"
	"github.com/dylanconnolly/captrivia-be/server"
)

var (
	listen string
)

func main() {
	flag.StringVar(&listen, "listen", ":8080", "Listen address")
	flag.Parse()

	ctx, _ := context.WithCancel(context.Background())
	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt)
	// go func() { <-c; cancel() }()

	app := NewApp()

	// if err := app.Run(ctx); err != nil {
	// 	log.Fatal("failed to start app.", err)
	// }
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

func NewApp() *App {
	hub := server.NewHub(redis.NewGameService())
	gameServer := server.NewGameServer(hub)
	httpServer := server.NewHTTPServer(listen, gameServer)

	return &App{
		hub:        hub,
		gameServer: gameServer,
		httpServer: httpServer,
	}
}
