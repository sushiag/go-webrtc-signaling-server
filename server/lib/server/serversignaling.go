package server

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

func (wsm *WebSocketManager) readMessages(userID uint64, conn *websocket.Conn) {
	defer func() {
		wsm.disconnectChan <- userID
		conn.Close()
		log.Printf("[WS] User %d disconnected", userID)
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error for user %d: %v", userID, err)
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("[WS] Invalid message from %d: %v", userID, err)
			continue
		}

		msg.Sender = userID
		wsm.messageChan <- msg
	}
}

func (wsm *WebSocketManager) handleMessage(msg Message) {
	switch msg.Type {
	case TypeCreateRoom:
		log.Printf("[WS] User %d requested to create a room", msg.Sender)
		roomID := wsm.CreateRoom(msg.Sender)
		resp := Message{
			Type:   TypeRoomCreated,
			RoomID: roomID,
			Sender: msg.Sender,
		}
		if conn, ok := wsm.Connections[msg.Sender]; ok {
			_ = wsm.SafeWriteJSON(conn, resp)
		}

	case TypeJoin:
		log.Printf("[User %d] requested to join room: %d", msg.Sender, msg.RoomID)
		wsm.AddUserToRoom(msg.RoomID, msg.Sender)

	case TypeOffer, TypeAnswer, TypeICE:
		room := wsm.Rooms[msg.RoomID]
		if room == nil {
			log.Printf("[WS WARNING] %s from %d ignored: Room %d does not exist", msg.Type, msg.Sender, msg.RoomID)
			return
		}
		if _, senderOk := room.Users[msg.Sender]; !senderOk {
			log.Printf("[WS WARNING] %s from %d ignored: Sender not in room %d", msg.Type, msg.Sender, msg.RoomID)
			return
		}
		if _, targetOk := room.Users[msg.Target]; !targetOk {
			log.Printf("[WS WARNING] %s from %d ignored: Target %d not in room %d", msg.Type, msg.Sender, msg.Target, msg.RoomID)
			return
		}

		log.Printf("[WS] Forwarding %s from %d to %d in room %d", msg.Type, msg.Sender, msg.Target, msg.RoomID)
		wsm.forwardOrBuffer(msg.Sender, msg)

	case TypePeerJoined:
		log.Printf("[WS] User %d joined room: %d", msg.Sender, msg.RoomID)

	case TypePeerListRequest:
		log.Printf("[WS] User %d requested peer list for room %d", msg.Sender, msg.RoomID)
		wsm.handlePeerListRequest(msg)

	case TypeStart:
		log.Printf("[WS] Received 'start' from user %d in room %d", msg.Sender, msg.RoomID)
		wsm.handleStart(msg)

	case TypeDisconnect:
		log.Printf("[WS] Disconnect request from user %d", msg.Sender)
		wsm.disconnectChan <- msg.Sender

	case TypeText:
		log.Printf("[WS] Text from %d: %s", msg.Sender, msg.Content)

	case TypeStartP2P:
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
				_ = peerConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Switching to P2P"))
				_ = peerConn.Close()
			}
		}

		// Cleanup room after disconnecting everyone
		delete(wsm.Rooms, msg.RoomID)
		log.Printf("[WS] Room %d cleaned up after start-session", msg.RoomID)

	case "host-changed":
		log.Printf("[WS] Host changed notification from user %d in room %d", msg.Sender, msg.RoomID)
		room, exists := wsm.Rooms[msg.RoomID]
		if exists {
			for uid, conn := range room.Users {
				if uid != msg.Sender && conn != nil {
					_ = wsm.SafeWriteJSON(conn, msg)
				}
			}
		}

	case "send-message":
		log.Printf("[WS] Sending message from user %d to %d: %s", msg.Sender, msg.Target, msg.Content)
		wsm.forwardOrBuffer(msg.Sender, msg)

	default:
		log.Printf("[WS] Unknown message type: %s", msg.Type)
	}
}
