package client

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	gws "github.com/gorilla/websocket"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
	"github.com/sushiag/go-webrtc-signaling-server/client/lib/webrtc"
	ws "github.com/sushiag/go-webrtc-signaling-server/client/lib/websocket"
)

func (c *Client) startWSLoops() {
	errCh := make(chan error)

	// Listen Loop
	go func() {
		log.Printf("[DEBUG: %d] starting WS listen loop", c.Websocket.UserID)
		defer func() {
			log.Printf("[DEBUG: %d] closing WS listen loop", c.Websocket.UserID)
			c.Websocket.Conn.Close()
		}()

		for {
			msgType, data, err := c.Websocket.Conn.ReadMessage()
			if err != nil {
				log.Printf("[ERROR: %d]: failed to read WS message from server: %v", c.Websocket.UserID, err)
			}

			switch msgType {
			case websocket.BinaryMessage:
				{
					log.Printf("[WARN: %d]: ignored binary message", c.Websocket.UserID)
				}
			case websocket.TextMessage:
				{
					var jsonMsg ws.Message
					err := json.Unmarshal(data, &jsonMsg)
					if err != nil {
						log.Printf("[ERROR: %d]: failed to unmarshal JSON WS message from server: %v", c.Websocket.UserID, err)
						continue
					}

					msgTypeStr, err := jsonMsg.Type.ToString()
					if err != nil {
						log.Panicf("[ERROR: %d] got unknown message type from server: %v", c.Websocket.UserID, err)
						continue
					}

					log.Printf("[INFO: %d]: handling text message from server with type: %s", c.Websocket.UserID, msgTypeStr)
					c.handleMessage(jsonMsg)
				}
			}
		}

	}()

	// Send Loop
	go func() {
		log.Println("[DEBUG] starting WS send loop")
		defer func() {
			log.Println("[DEBUG] closing WS send loop")
		}()

		for {
			select {
			case msg := <-c.Websocket.SendWSMsgCh:
				{
					if c.Websocket.Conn == nil || c.Websocket.IsClosed {
						log.Printf("[CLIENT SIGNALING] cannot WS message send, connection is closed")
						break
					}

					// kinda hacky but we must set these correctly before sending
					msg.RoomID = c.Websocket.RoomID
					msg.Sender = c.Websocket.UserID

					// NOTE: debug logging, should be removed later
					msgTypeStr, err := msg.Type.ToString()
					if err != nil {
						log.Panicf("tried to send an unknown message type to the server: %v", err)
					}
					log.Printf("[SEND LOOP: %d] writing message type `%s` to WS conn", c.Websocket.UserID, msgTypeStr)

					log.Printf("[SEND LOOP: %d] MSG ROOM ID %d", c.Websocket.UserID, msg.RoomID)

					b, err := json.Marshal(msg)
					if err != nil {
						log.Printf("[SEND LOOP: %d] failed to marshal WS message: %v", c.Websocket.UserID, err)
					} else {
						log.Printf("[SEND LOOP: %d] outgoing JSON: %s", c.Websocket.UserID, b)
					}

					if err := c.Websocket.Conn.WriteJSON(msg); err != nil {
						log.Printf("[CLIENT SIGNALING] Failed to send message type '%d': %v", msg.Type, err)
					}
				}

			case err := <-errCh:
				{
					if closeErr, ok := err.(*gws.CloseError); ok {
						log.Printf("[SIGNALING: %d] web socket closed: %s", c.Websocket.UserID, closeErr.Text)
					} else {
						log.Printf("[SIGNALING: %d] read error: %s", c.Websocket.UserID, err)
					}
					return
				}

			case <-c.Websocket.DoneCh:
				{
					return
				}
			}
		}
	}()
}

func (c *Client) handleMessage(msg ws.Message) {
	switch msg.Type {
	case common.MessageTypeSetUserID:
		{
			oldID := c.Websocket.UserID
			c.Websocket.UserID = msg.UserID
			c.PeerManager.UserID = msg.UserID
			log.Printf("user id set from %d to %d", oldID, msg.UserID)
		}

	case common.MessageTypeRoomCreated:
		{
			log.Printf("[DEBUG: %d] created room %d\n", c.Websocket.UserID, msg.RoomID)
			c.Websocket.RoomID = msg.RoomID
			c.createRoomRespCh <- nil
		}

	case common.MessageTypeRoomJoined:
		{
			log.Printf("[DEBUG: %d] joined room %d\n", c.Websocket.UserID, msg.RoomID)
			c.Websocket.RoomID = msg.RoomID
			c.joinRoomRespCh <- nil
		}

	case common.MessageTypeStartSession:
		{
			// TODO: maybe we don't really need to convert types here
			signalingMsg := webrtc.SignalingMessage{
				Type:      msg.Type,
				Sender:    msg.Sender,
				Target:    msg.Target,
				SDP:       msg.SDP,
				Candidate: msg.Candidate,
				Text:      msg.Text,
				Users:     msg.Users,
			}
			c.PeerManager.HandleIncomingMessage(signalingMsg, c.Websocket.SendWSMsgCh)

			// TODO: this is a bit messy and needs some re-organization
			// ... why are we closing the WS connection?
			if c.Websocket.IsClosed {
				// if there isn't an existing connection, starting a session is fine
				c.startSessionRespCh <- nil
			} else {
				// otherwise, we close the current connection
				c.Websocket.IsClosed = true
				close(c.Websocket.DoneCh)
				if c.Websocket.Conn != nil {
					_ = c.Websocket.Conn.Close()
					c.Websocket.Conn = nil
				}
				log.Println("[CLIENT SIGNALING] Client disconnected from signaling server.")
				c.startSessionRespCh <- fmt.Errorf("already has an existing connection")
			}

			c.startSessionRespCh <- nil
		}

	case common.MessageTypeSendMessage:
		{
			if msg.Text != "" {
				log.Printf("Text message from %d: %s", msg.Sender, msg.Text)
			}
			if msg.Payload.Data != nil {
				log.Printf("Binary message from %d (%s): %d bytes", msg.Sender, msg.Payload.DataType, len(msg.Payload.Data))
			}

			c.Websocket.SendWSMsgCh <- msg
		}

	case common.MessageTypePeerJoined:
		{
			log.Printf("[DEBUG: %d] got joined room message for: %d", c.Websocket.UserID, msg.UserID)

			// TODO: maybe we don't really need to convert types here
			signalingMsg := webrtc.SignalingMessage{
				Type:   msg.Type,
				UserID: msg.UserID,
			}
			c.PeerManager.HandleIncomingMessage(signalingMsg, c.Websocket.SendWSMsgCh)

			// TODO: this is a workound for the bug that the sever isn't sending
			// a `common.MessageTypeRoomJoined` message when the client joins a room
			if c.joinRoomRespCh != nil {
				c.joinRoomRespCh <- nil
			}
		}

	case
		common.MessageTypePeerLeft,
		common.MessageTypeOffer,
		common.MessageTypeAnswer,
		common.MessageTypeICECandidate,
		common.MessageTypeHostChanged:
		{
			// TODO: maybe we don't really need to convert types here
			signalingMsg := webrtc.SignalingMessage{
				Type:      msg.Type,
				Sender:    msg.Sender,
				Target:    msg.Target,
				SDP:       msg.SDP,
				Candidate: msg.Candidate,
				Text:      msg.Text,
				Users:     msg.Users,
				UserID:    msg.UserID,
			}
			c.PeerManager.HandleIncomingMessage(signalingMsg, c.Websocket.SendWSMsgCh)
		}

	default:
		{
			msgTypeStr, _ := msg.Type.ToString()
			log.Printf("[CLIENT SIGNALING] Unhandled message type: %s", msgTypeStr)
		}
	}
}
