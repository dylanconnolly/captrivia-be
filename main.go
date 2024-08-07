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

	log.Println("starting server")
	log.Println("listening on ", app.httpServer.Addr)

	err := app.httpServer.ListenAndServe()
	if err != nil {
		log.Fatalf("failed to listen: %s", err)
	}

	// <-ctx.Done()
	// app.Close()
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

// func (a *App) Close() error {
// 	if a.hub != nil {
// 		a.hub.Close()
// 	}
// 	return nil
// }

// func (a *App) Run(ctx context.Context) error {
// 	go a.hub.Run(ctx)

// 	err := a.httpServer.ListenAndServe()
// 	if err != nil {
// 		log.Fatal("HTTPServer failed to listen")
// 		return err
// 	}

// 	return nil
// }
