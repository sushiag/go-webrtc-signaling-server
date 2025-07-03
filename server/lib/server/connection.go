package server

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

func NewConnection(userID uint64, conn *websocket.Conn, inboundMessages chan<- Message, disconnectOut chan<- uint64) *Connection {
	c := &Connection{
		UserID:       userID,
		Conn:         conn,
		Incoming:     make(chan Message),
		Outgoing:     make(chan Message),
		Disconnected: disconnectOut,
	}
	go c.readLoop(inboundMessages)
	go c.writeLoop()
	return c
}

func (c *Connection) readLoop(inboundMessages chan<- Message) {
	defer func() {
		c.Disconnected <- c.UserID
		c.Conn.Close()
		log.Printf("[WS] User %d disconnected (read)", c.UserID)
	}()
	for {
		msgType, data, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("[WS] Read error from user %d: %v", c.UserID, err)
			return
		}

		switch msgType {
		case websocket.BinaryMessage:
			{
				log.Printf("[WS] ignoring binary message from %d", c.UserID)
			}
		case websocket.TextMessage:
			{
				var msg Message
				if err := json.Unmarshal(data, &msg); err != nil {
					log.Printf("[WS] failed to unmarshal WS message from %d: %v", c.UserID, err)
					continue
				}
				msg.Sender = c.UserID
				inboundMessages <- msg
			}
		}
	}
}

func (c *Connection) writeLoop() {
	for msg := range c.Outgoing {
		if err := c.Conn.WriteJSON(msg); err != nil {
			log.Printf("[WS Server] Write error to %d: %v", c.UserID, err)
			c.Disconnected <- c.UserID
			return
		}
	}
}
