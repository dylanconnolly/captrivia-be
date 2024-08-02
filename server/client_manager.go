package server

// ClientManager tracks all active websocket clients and broadcasts
// messages to each client
type ClientManager struct {
	broadcast  chan []byte
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		broadcast:  make(chan []byte),
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (cm *ClientManager) Run() {
	for {
		select {
		case client := <-cm.register:
			cm.clients[client] = true
		case client := <-cm.unregister:
			delete(cm.clients, client)
			close(client.send)
		case message := <-cm.broadcast:
			for client := range cm.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(cm.clients, client)
				}
			}
		}
	}
}
