package websocket

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func (c *Client) Init() error {
	headers := http.Header{}
	headers.Set("X-Api-Key", c.ApiKey) // set the auth header

	wsEndpoint := fmt.Sprintf("ws://%s/ws", c.ServerURL)
	conn, _, err := websocket.DefaultDialer.Dial(wsEndpoint, headers)
	if err != nil {
		return fmt.Errorf("[CLIENT SIGNALING] websocket connection failed: %v", err)
	}

	log.Println("[CLIENT SIGNALING] Connected to:", c.ServerURL)
	c.Conn = conn
	return nil
}
