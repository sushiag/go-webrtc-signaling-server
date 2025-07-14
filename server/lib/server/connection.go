package server

import (
	"log"
	"time"

	"github.com/gorilla/websocket"

	smsg "signaling-msgs"
)

func NewConnection(userID uint64, conn *websocket.Conn, inboundMessages chan<- *smsg.MessageRawJSONPayload, disconnectOut chan<- uint64) *Connection {
	c := &Connection{
		UserID:       userID,
		Conn:         conn,
		Incoming:     make(chan Message),
		Outgoing:     make(chan smsg.MessageAnyPayload),
		Disconnected: disconnectOut,
	}
	go c.readLoop(inboundMessages)
	go c.writeLoop()
	return c
}

func (c *Connection) readLoop(inboundMessages chan<- *smsg.MessageRawJSONPayload) {
	defer func() {
		c.Disconnected <- c.UserID
		c.Conn.Close()
		close(c.Outgoing)
		log.Printf("[WS] User %d disconnected (read)", c.UserID)
	}()

	for {
		msg := &smsg.MessageRawJSONPayload{}
		if err := c.Conn.ReadJSON(&msg); err != nil {
			log.Printf("[WS] failed to read WS message from %d: %v", c.UserID, err)
			continue
		}
		log.Printf("[WS] got WS message from %d", c.UserID)

		msg.From = c.UserID
		inboundMessages <- msg
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
				log.Printf("[DEBUG] sent '%s' msg to %d", msg.MsgType.AsString(), c.UserID)
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
