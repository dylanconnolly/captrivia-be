package server

type ClientManager struct {
	broadcast  chan []byte
	clients    map[string]bool
	register   chan *Client
	unregister chan *Client
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		broadcast:  make(chan []byte),
		clients:    make(map[string]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (cm *ClientManager) Run() {
	for {
		select {
		case client := <-cm.register:
			cm.clients[client.name] = true
		case client := <-cm.unregister:
			delete(cm.clients, client.name)
			close(client.messages)
		}
	}
}
