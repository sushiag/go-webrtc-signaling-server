package server

import (
	"encoding/json"
	"log"

	smsg "signaling-msgs"
)

// TODO: set the FROM correctly before handling the message
func (wsm *WebSocketManager) handleMessage(msg *smsg.MessageRawJSONPayload) {
	switch msg.MsgType {
	case smsg.CreateRoom:
		log.Printf("[WS] User %d requested to create a room", msg.From)
		roomID := wsm.createRoom(msg.From)
		resp := smsg.MessageAnyPayload{
			MsgType: smsg.RoomCreated,
			Payload: smsg.RoomCreatedPayload{RoomID: roomID},
		}

		if conn, ok := wsm.Connections[msg.From]; ok {
			_ = wsm.SafeWriteJSON(conn, resp)
		}

	case smsg.JoinRoom:
		var payload smsg.JoinRoomPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("[ERROR] failed to unmarshal join room payload from: %d", msg.From)
			break
		}

		log.Printf("[User %d] requested to join room: %d", msg.From, payload.RoomID)
		wsm.addUserToRoom(payload.RoomID, msg.From)

	case smsg.SDP, smsg.ICECandidate:
		conn, exists := wsm.Connections[msg.To]
		if !exists {
			log.Printf("[WARN] client %d tried to send a %s message to an unknown client: %d", msg.From, msg.MsgType.AsString(), msg.To)
			return
		}

		// TODO(rmarinn): i removed the check if the users are in the same room... need to re-implement it

		log.Printf("[WS] Forwarding %s from %d to %d", msg.MsgType.AsString(), msg.From, msg.To)
		// kinda wasteful casting but whatever
		wsm.SafeWriteJSON(conn, smsg.MessageAnyPayload{
			MsgType: msg.MsgType,
			From:    msg.From,
			To:      msg.To,
			Payload: msg.Payload,
		})

	case smsg.LeaveRoom:
		log.Printf("[WS] Disconnect request from user %d", msg.From)
		wsm.disconnectChan <- msg.From

		// NOTE: unused
		// case TypeText:
		// 	log.Printf("[WS] Text from %d: %s", msg.Sender, msg.Content)

		// TODO: we might need to re-implement some of these
		// case MessageTypeStartSession:
		// 	log.Printf("[WS] Received start-session from peer %d in room %d", msg.Sender, msg.RoomID)
		// 	room, exists := wsm.Rooms[msg.RoomID]
		// 	if !exists {
		// 		log.Printf("[WS] Room %d does not exist", msg.RoomID)
		// 		return
		// 	}
		//
		// 	if room.HostID != msg.Sender {
		// 		log.Printf("[WS] start-session denied: User %d is not host of room %d", msg.Sender, msg.RoomID)
		// 		return
		// 	}
		//
		// 	for uid, peerConn := range room.Users {
		// 		if peerConn != nil {
		// 			log.Printf("[WS] Closing connection for user %d (P2P start)", uid)
		// 			_ = peerConn.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Switching to P2P"))
		// 			_ = peerConn.Conn.Close()
		// 		}
		// 	}
		//
		// 	// Cleanup room after disconnecting everyone
		// 	delete(wsm.Rooms, msg.RoomID)
		// 	log.Printf("[WS] Room %d cleaned up after start-session", msg.RoomID)
		//
		// case MessageTypeHostChanged:
		// 	log.Printf("[WS] Host changed notification from user %d in room %d", msg.Sender, msg.RoomID)
		// 	room, exists := wsm.Rooms[msg.RoomID]
		// 	if exists {
		// 		for uid, conn := range room.Users {
		// 			if uid != msg.Sender && conn != nil {
		// 				_ = wsm.SafeWriteJSON(conn, msg)
		// 			}
		// 		}
		//
		// 	}
		//
		// case MessageTypeSendMessage:
		// 	log.Printf("[WS] Sending message from user %d to %d: %s", msg.Sender, msg.Target, msg.Content)
		// 	wsm.forwardOrBuffer(msg.Sender, msg)
		//
		// default:
		// 	log.Printf("[WS] Unknown message type: %s", msg.Type)
		// }
	}
}

func (wsm *WebSocketManager) handleNewConnection(conn *Connection) {
	log.Printf("[WS] User %d connected", conn.UserID)
	conn.Disconnected = wsm.disconnectChan

	conn.Conn.SetPongHandler(func(string) error {
		return nil
	})

	go conn.readLoop(wsm.messageChan)
	go conn.writeLoop()
	log.Printf("[WS] WS read and write loop for user %d started", conn.UserID)

	wsm.Connections[conn.UserID] = conn

	// NOTE: idk what this is for but im keeping it as a comment just in case it's
	// important
	// wsm.flushBufferedMessages(userID)
}
