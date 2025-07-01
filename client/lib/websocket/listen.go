package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
)

func (c *Client) Start() {
	readCh := make(chan []byte)
	errCh := make(chan error)

	// Listen Loop
	go func() {
		for {
			c.Conn.SetReadDeadline(time.Now().Add(time.Second * 30))
			msgType, data, err := c.Conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			if msgType != websocket.TextMessage {
				log.Printf("[CLIENT SIGNALING] Ignoring non-text message (type=%d)", msgType)
				continue
			}
			readCh <- data
		}
	}()

	// Send Loop
	go func() {
		for {
			select {
			case msg := <-c.sendQueue:
				if c.Conn == nil || c.isClosed {
					log.Printf("[CLIENT SIGNALING] Cannot send, connection is closed.")
					continue
				}
				if msg.RoomID == 0 {
					msg.RoomID = c.RoomID
				}
				if msg.Sender == 0 {
					msg.Sender = c.UserID
				}
				if err := c.Conn.WriteJSON(msg); err != nil {
					log.Printf("[CLIENT SIGNALING] Failed to send '%s': %v", msg.Type.String(), err)
				}
			case data := <-readCh:
				var msg Message
				if err := json.Unmarshal(data, &msg); err != nil {
					log.Println("[CLIENT SIGNALING] Unmarshal error:", err)
					log.Println("[CLIENT SIGNALING] Raw data:", string(data))
					continue
				}

				c.handleMessage(msg)

			case err := <-errCh:
				if closeErr, ok := err.(*websocket.CloseError); ok {
					log.Printf("[CLIENT SIGNALING] WebSocket closed: %s", closeErr.Text)
				} else {
					log.Println("[CLIENT SIGNALING] Read error:", err)
				}
				c.Close()
				return

			case <-c.doneCh:
				return
			}
		}
	}()
}

func (c *Client) handleMessage(msg Message) {
	switch msg.Type {
	case common.MessageTypeRoomCreated:
		fmt.Printf("Room created: %d\n", msg.RoomID)
		c.RoomID = msg.RoomID

	case common.MessageTypeRoomJoined:
		c.RoomID = msg.RoomID
		c.RequestPeerList()

	case common.MessageTypeStartSession:
		if c.OnMessage != nil {
			c.OnMessage(msg)
		}
		c.CloseSignaling()
		return

	case common.MessageTypeSendMessage:
		if msg.Text != "" {
			log.Printf("Text message from %d: %s", msg.Sender, msg.Text)
		}
		if msg.Payload.Data != nil {
			log.Printf("Binary message from %d (%s): %d bytes", msg.Sender, msg.Payload.DataType, len(msg.Payload.Data))
		}
		if err := c.SendDataToPeer(msg.Target, []byte(msg.Text)); err != nil {
			log.Printf("[CLIENT SIGNALING] Failed to send data to peer: %v", err)
		}

	case common.MessageTypePeerJoined,
		common.MessageTypePeerList,
		common.MessageTypeOffer,
		common.MessageTypeAnswer,
		common.MessageTypeICECandidate,
		common.MessageTypeHostChanged:

	default:
		log.Printf("[CLIENT SIGNALING] Unhandled message type: %s", msg.Type)
	}

	if c.OnMessage != nil {
		c.OnMessage(msg)
	}
}
