package client

import (
	"log"

	gws "github.com/gorilla/websocket"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
	"github.com/sushiag/go-webrtc-signaling-server/client/lib/webrtc"
	ws "github.com/sushiag/go-webrtc-signaling-server/client/lib/websocket"
)

func (c *Client) startWSLoops() {
	errCh := make(chan error)

	// Listen Loop
	go func() {
		log.Println("[DEBUG] starting WS listen loop")
		defer func() {
			log.Println("[DEBUG] closing WS listen loop")
			c.Websocket.Conn.Close()
		}()

		for {
			var msg ws.Message
			err := c.Websocket.Conn.ReadJSON(&msg)
			if err != nil {
				log.Printf("[ERROR: %d]: failed to read JSON message from WS conn: %v\n", c.Websocket.UserID, err)
				return
			}

			c.handleMessage(msg)
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
						log.Printf("[CLIENT SIGNALING] Cannot send, connection is closed.")
						continue
					}

					if msg.RoomID == 0 {
						msg.RoomID = c.Websocket.RoomID
					}

					if msg.Sender == 0 {
						msg.Sender = c.Websocket.UserID
					}

					log.Printf("[SEND LOOP: %d] writing to WS conn", c.Websocket.UserID)
					if err := c.Websocket.Conn.WriteJSON(msg); err != nil {
						log.Printf("[CLIENT SIGNALING] Failed to send '%s': %v", msg.Type.String(), err)
					}
				}

			case err := <-errCh:
				{
					if closeErr, ok := err.(*gws.CloseError); ok {
						log.Printf("[CLIENT SIGNALING] WebSocket closed: %s", closeErr.Text)
					} else {
						log.Println("[CLIENT SIGNALING] Read error:", err)
					}
					c.Close()
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
	log.Printf("[HANDLE MSG: %d]: handling %v msg", c.Websocket.UserID, msg.Type)

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

			if !c.Websocket.IsClosed {
				c.Websocket.IsClosed = true
				close(c.Websocket.DoneCh)
				if c.Websocket.Conn != nil {
					_ = c.Websocket.Conn.Close()
					c.Websocket.Conn = nil
				}
				log.Println("[CLIENT SIGNALING] Client disconnected from signaling server.")
			}

			c.startSessionRespCh <- nil
			return
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
			log.Printf("[CLIENT SIGNALING] Unhandled message type: %s", msg.Type)
		}
	}
}
