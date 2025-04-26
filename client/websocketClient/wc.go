package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type WebSocketClient struct {
	Conn     *websocket.Conn
	ClientID string
}

func (wsc *WebSocketClient) Send(msg Message) {
	wsc.Conn.WriteJSON(msg)
}

func connect(apiKey, serverURL string) (*WebSocketClient, error) {
	headers := http.Header{}
	headers.Set("X-Api-Key", apiKey)

	conn, _, err := websocket.DefaultDialer.Dial(serverURL, headers)
	if err != nil {
		return nil, err
	}

	client := &WebSocketClient{Conn: conn}

	go func() {
		for {
			var msg Message
			if err := conn.ReadJSON(&msg); err != nil {
				log.Println("WebSocket read error:", err)
				break
			}
			handleSignal(msg, client)
		}
	}()

	return client, nil
}
