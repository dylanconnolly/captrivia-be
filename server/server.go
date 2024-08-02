package server

import (
	"net/http"

	"github.com/rs/cors"
)

func NewHTTPServer(addr string, gameServer *GameServer) *http.Server {
	mux := NewRouter(gameServer)

	httpStack := cors.AllowAll().Handler(mux)

	return &http.Server{
		Addr:    addr,
		Handler: httpStack,
	}
}
