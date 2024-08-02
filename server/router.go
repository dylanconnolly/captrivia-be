package server

import "net/http"

func NewRouter(gameServer *GameServer) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /games", gameServer.Games) // Get existing games
	mux.HandleFunc("GET /connect", gameServer.Connect)

	return mux
}
