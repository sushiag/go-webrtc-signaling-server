package server

import (
	"log"

	"github.com/gorilla/websocket"
)

func (wsm *WebSocketManager) handleMessage(msg Message) {
	switch msg.Type {
	case MessageTypeCreateRoom:
		log.Printf("[WS] User %d requested to create a room", msg.Sender)
		roomID := wsm.createRoom(msg.Sender)
		resp := Message{
			Type:   MessageTypeRoomCreated,
			RoomID: roomID,
			Sender: msg.Sender,
		}

		if conn, ok := wsm.Connections[msg.Sender]; ok {
			_ = wsm.SafeWriteJSON(conn, resp)
		}

	case MessageTypeJoinRoom:
		log.Printf("[User %d] requested to join room: %d", msg.Sender, msg.RoomID)
		wsm.addUserToRoom(msg.RoomID, msg.Sender)

	case MessageTypeOffer, MessageTypeAnswer, MessageTypeICECandidate:
		room := wsm.Rooms[msg.RoomID]
		if room == nil {
			// TODO: remove this expensive logging eventually
			rooms := make([]uint64, len(wsm.Rooms))
			idx := 0
			for _, room := range wsm.Rooms {
				rooms[idx] = room.ID
				idx += 1
			}

			log.Printf("[WS WARNING] %s from %d ignored: Room %d does not exist; current rooms: %v", msg.Type, msg.Sender, msg.RoomID, rooms)
			return
		}

		if _, senderOk := room.Users[msg.Sender]; !senderOk {
			// TODO: remove this expensive logging eventually
			usersInRoom := make([]uint64, len(room.Users))
			for i, user := range room.Users {
				usersInRoom[i] = user.UserID
			}

			log.Printf("[WS WARNING] %s from %d ignored: Sender not in room %d; current users in room: %v", msg.Type, msg.Sender, msg.RoomID, usersInRoom)
			return
		}

		if _, targetOk := room.Users[msg.Target]; !targetOk {
			// TODO: remove this expensive logging eventually
			usersInRoom := make([]uint64, len(room.Users))
			for i, user := range room.Users {
				usersInRoom[i] = user.UserID
			}

			log.Printf("[WS WARNING] %s from %d ignored: Target %d not in room %d; current users in room: %v", msg.Type, msg.Sender, msg.Target, msg.RoomID, usersInRoom)
			return
		}

		log.Printf("[WS] Forwarding %s from %d to %d in room %d", msg.Type, msg.Sender, msg.Target, msg.RoomID)
		wsm.forwardOrBuffer(msg.Sender, msg)

	case MessageTypePeerJoined:
		log.Printf("[WS] User %d joined room: %d", msg.Sender, msg.RoomID)

	case MessageTypePeerListReq:
		log.Printf("[WS] User %d requested peer list for room %d", msg.Sender, msg.RoomID)
		wsm.handlePeerListRequest(msg)

	// NOTE: i think this is the same as MessageTypeStartSession
	// case TypeStart:
	// 	log.Printf("[WS] Received 'start' from user %d in room %d", msg.Sender, msg.RoomID)
	// 	wsm.handleStart(msg)

	case MessageTypeDisconnect:
		log.Printf("[WS] Disconnect request from user %d", msg.Sender)
		wsm.disconnectChan <- msg.Sender

	// NOTE: unused
	// case TypeText:
	// 	log.Printf("[WS] Text from %d: %s", msg.Sender, msg.Content)

	case MessageTypeStartSession:
		log.Printf("[WS] Received start-session from peer %d in room %d", msg.Sender, msg.RoomID)
		room, exists := wsm.Rooms[msg.RoomID]
		if !exists {
			log.Printf("[WS] Room %d does not exist", msg.RoomID)
			return
		}

		if room.HostID != msg.Sender {
			log.Printf("[WS] start-session denied: User %d is not host of room %d", msg.Sender, msg.RoomID)
			return
		}

		for uid, peerConn := range room.Users {
			if peerConn != nil {
				log.Printf("[WS] Closing connection for user %d (P2P start)", uid)
				_ = peerConn.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Switching to P2P"))
				_ = peerConn.Conn.Close()
			}
		}

		// Cleanup room after disconnecting everyone
		delete(wsm.Rooms, msg.RoomID)
		log.Printf("[WS] Room %d cleaned up after start-session", msg.RoomID)

	case MessageTypeHostChanged:
		log.Printf("[WS] Host changed notification from user %d in room %d", msg.Sender, msg.RoomID)
		room, exists := wsm.Rooms[msg.RoomID]
		if exists {
			for uid, conn := range room.Users {
				if uid != msg.Sender && conn != nil {
					_ = wsm.SafeWriteJSON(conn, msg)
				}
			}

		}

	case MessageTypeSendMessage:
		log.Printf("[WS] Sending message from user %d to %d: %s", msg.Sender, msg.Target, msg.Content)
		wsm.forwardOrBuffer(msg.Sender, msg)

	default:
		log.Printf("[WS] Unknown message type: %s", msg.Type)
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

	// we tell the user their user ID here
	conn.Outgoing <- Message{
		Type:   MessageTypeSetUserID,
		UserID: conn.UserID,
	}

	// NOTE: idk what this is for but im keeping it as a comment just in case it's
	// important
	// wsm.flushBufferedMessages(userID)
}
