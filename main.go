package main

import (
	"flag"
	"fmt"

	"github.com/dylanconnolly/captrivia-be/server"
)

var (
	listen string
)

func main() {
	flag.StringVar(&listen, "listen", "localhost:8080", "Listen address")
	flag.Parse()
	cm := server.NewClientManager()
	go cm.Run()

	gameServer := server.NewGameServer(cm)
	httpServer := server.NewHTTPServer(listen, gameServer)

	err := httpServer.ListenAndServe()
	if err != nil {
		fmt.Println("listen failed", err) // todo, better logging and error handling
	}
}
