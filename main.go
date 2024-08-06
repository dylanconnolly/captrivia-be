package main

import (
	"flag"
	"fmt"

	"github.com/dylanconnolly/captrivia-be/redis"
	"github.com/dylanconnolly/captrivia-be/server"
)

var (
	listen string
)

func main() {
	flag.StringVar(&listen, "listen", "localhost:8080", "Listen address")
	flag.Parse()

	db := redis.NewDB()
	db.LoadQuestionsFromFile("questions.json")
	hub := server.NewHub(db)
	go hub.Run()

	gameServer := server.NewGameServer(hub)
	httpServer := server.NewHTTPServer(listen, gameServer)
	redis.NewClient()

	err := httpServer.ListenAndServe()
	if err != nil {
		fmt.Println("listen failed", err) // todo, better logging and error handling
	}
}
