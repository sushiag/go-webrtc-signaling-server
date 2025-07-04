package server

import (
	"encoding/json"
	"log"
	"time"

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
		close(c.Outgoing)
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
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-c.Outgoing:
			{
				if err := c.Conn.WriteJSON(msg); err != nil {
					log.Printf("[WS Server] Write error to %d: %v", c.UserID, err)
					c.Disconnected <- c.UserID
				}
				log.Printf("[DEBUG] sent WS msg to %d", c.UserID)
			}

		// TODO: we can probably ping only when there has been no activitiy for some time
		// instead on a fixed interval
		case <-ticker.C:
			{
				if err := c.Conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					log.Printf("[WS] Ping to user %d failed: %v", c.UserID, err)
					c.Disconnected <- c.UserID
					return
				}
			}
		}
	}
}
