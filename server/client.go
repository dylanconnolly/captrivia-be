package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Client struct {
	name     string
	manager  *ClientManager
	conn     *websocket.Conn
	messages chan string
}

// upgrades HTTP protocol to websocket and creates a client to manage the connection and messages
func newClient(name string, manager *ClientManager, w http.ResponseWriter, r *http.Request) (*Client, error) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return r.Header.Get("Origin") == "http://localhost:3000"
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error upgrading connection: %s\n", err)
		return nil, err
	}

	c := &Client{
		conn:     conn,
		name:     name,
		manager:  manager,
		messages: make(chan string),
	}

	return c, nil
}

func (c *Client) readMessages() {
	c.conn.ReadMessage()
	for {
		t, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("error reading message: %s\n", err)
			break
		}
		var data map[string]interface{}
		err = json.Unmarshal(message, &data)
		if err != nil {
			log.Printf("error unmarshalling: %s", err)
		}
		log.Printf("message: %+v, message_type: %d\n", data, t)
	}
}

// handle websocket requests from frontend
func (c *Client) handleWebsockets() {
	c.manager.register <- c

	go c.readMessages()
}
