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
			// var msg ws.Message
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
					log.Printf("[INFO: %d]: handling text message from server with type: %d", c.Websocket.UserID, jsonMsg.Type)
					c.handleMessage(jsonMsg)
				}
			}

			// err := c.Websocket.Conn.ReadJSON(&msg)
			// if err != nil {
			// 	log.Printf("[ERROR: %d]: failed to read JSON message from WS conn: %v", c.Websocket.UserID, err)
			// 	continue
			// }
			//
			// c.handleMessage(msg)
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

					if msg.RoomID == 0 {
						msg.RoomID = c.Websocket.RoomID
					}

					if msg.Sender == 0 {
						msg.Sender = c.Websocket.UserID
					}

					// NOTE: debug logging, should be removed later
					log.Printf("[SEND LOOP: %d] writing message type to WS conn for %d, %v", c.Websocket.UserID, msg.Type, msg.Target)
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
	case common.MessageTypeRoomCreated:
		{
			log.Printf("Room created: %d\n", msg.RoomID)
			c.Websocket.RoomID = msg.RoomID
			c.createRoomRespCh <- nil
		}

	case common.MessageTypeRoomJoined:
		{
			c.Websocket.RoomID = msg.RoomID

			// request peer list
			c.Websocket.SendWSMsgCh <- ws.Message{Type: common.MessageTypePeerListReq}

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

	case common.MessageTypePeerList:
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

			// TODO: this is a workound for the bug that the sever isn't sending
			// a `common.MessageTypeRoomJoined` message when the client joins a room
			if c.joinRoomRespCh != nil {
				c.joinRoomRespCh <- nil
			}
		}

	case common.MessageTypePeerJoined,
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
			}
			c.PeerManager.HandleIncomingMessage(signalingMsg, c.Websocket.SendWSMsgCh)
		}

	default:
		{
			log.Printf("[CLIENT SIGNALING] Unhandled message type `%d`: %v", msg.Type, msg)
		}
	}
}
