package websocket

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

func (c *Client) listen() {
	for {
		select {
		case <-c.doneCh:
			return
		default:
			msgType, data, err := c.Conn.ReadMessage()
			if err != nil {
				if closeErr, ok := err.(*websocket.CloseError); ok {
					log.Printf("[CLIENT SIGNALING] WebSocket closed: %s", closeErr.Text)
				} else {
					log.Println("[CLIENT SIGNALING] Read error:", err)
				}
				c.Close()
				return
			}

			if msgType != websocket.TextMessage {
				log.Printf("[CLIENT SIGNALING] Ignoring non-text message (type=%d)", msgType)
				continue
			}

			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Println("[CLIENT SIGNALING] Unmarshal error:", err)
				continue
			}

			c.handleMessage(msg)
		}

	}
}
func (c *Client) handleMessage(msg Message) {
	switch msg.Type {
	case MessageTypeRoomCreated:
		fmt.Printf("Room created: %d\n", msg.RoomID)
		c.RoomID = msg.RoomID
		// c.RequestPeerList()
	case MessageTypeRoomJoined:
		c.RoomID = msg.RoomID
		c.RequestPeerList()
	case MessageTypeStartSession:
		if c.onMessage != nil {
			c.onMessage(msg)
		}
		c.CloseSignaling()
		return
	case MessageTypeSendMessage:
		if msg.Text != "" {
			log.Printf("Text message from %d: %s", msg.Sender, msg.Text)
		}
		if msg.Payload.Data != nil {
			log.Printf("Binary message from %d (%s): %d bytes", msg.Sender, msg.Payload.DataType, len(msg.Payload.Data))
		}
		if err := c.SendDataToPeer(msg.Target, []byte(msg.Text)); err != nil {
			log.Printf("[CLIENT SIGNALING] Failed to send data to peer: %v", err)
		}
	}
	if c.onMessage != nil {
		c.onMessage(msg)
	}
}
