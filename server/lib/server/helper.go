package server

import (
	"log"

	"github.com/gorilla/websocket"
)

// TODO: currently unused... is this the same as start session?
func (wsm *WebSocketManager) handleStart(msg Message) {
	roomID := msg.RoomID

	room, exists := wsm.Rooms[roomID]
	if !exists {
		log.Printf("[WS] Room %d does not exist", roomID)
		return
	}

	for uid, peerConn := range room.Users {
		if peerConn != nil {
			log.Printf("[WS] Closing connection to user %d for P2P switch", uid)
			_ = peerConn.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Switching to P2P"))
			_ = peerConn.Conn.Close()
		}
	}

	delete(wsm.Rooms, roomID)
}

func (wsm *WebSocketManager) addUserToRoom(roomID uint64, joiningUserID uint64) {
	log.Printf("[DEBUG] adding user %d to room %d", joiningUserID, roomID)

	room, exists := wsm.Rooms[roomID]
	if !exists {
		log.Printf("[ERROR] user %d tried to join non-existent room %d, skipping join", joiningUserID, roomID)
		return
	}

	if _, alreadyJoined := room.Users[joiningUserID]; alreadyJoined {
		log.Printf("[WS] User %d is already in room %d, skipping join", joiningUserID, roomID)
		return
	}

	room.Users[joiningUserID] = wsm.Connections[joiningUserID]
	var peers []uint64
	for uid := range room.Users {
		if uid != joiningUserID {
			peers = append(peers, uid)
		}
	}

	// Notify the new user the room they joined
	if conn, ok := wsm.Connections[joiningUserID]; ok {
		_ = wsm.SafeWriteJSON(conn, Message{
			Type:   MessageTypeRoomJoined,
			RoomID: roomID,
		})
	}

	// Then notify everyone else of the new peer that joined
	for _, user := range room.Users {
		if user.UserID == joiningUserID {
			continue
		}

		conn, exists := wsm.Connections[user.UserID]
		if !exists {
			log.Printf("[ERROR] unknown user (%d) is in room %d", conn.UserID, room.ID)
		}

		log.Printf("[DEBUG] notifying user %d that %d joined the room", conn.UserID, joiningUserID)

		_ = wsm.SafeWriteJSON(conn, Message{
			Type:   MessageTypePeerJoined,
			UserID: joiningUserID,
		})
	}

	log.Printf("[WS] User %d joined room %d", joiningUserID, roomID)
}

func (wsm *WebSocketManager) forwardOrBuffer(senderID uint64, msg Message) {
	conn, exists := wsm.Connections[msg.Target]
	inSameRoom := wsm.AreInSameRoom(msg.RoomID, []uint64{msg.Sender, msg.Target})

	log.Printf("[WS DEBUG] forwardOrBuffer type=%s from=%d to=%d exists=%v sameRoom=%v",
		msg.Type, senderID, msg.Target, exists, inSameRoom)

	if !exists || !inSameRoom {
		log.Printf("[WS DEBUG] Buffering %s from %d to %d", msg.Type, msg.Sender, msg.Target)
		wsm.candidateBuffer[msg.Target] = append(wsm.candidateBuffer[msg.Target], msg)
		return
	}

	if err := wsm.SafeWriteJSON(conn, msg); err != nil {
		log.Printf("[WS ERROR] Failed to send %s from %d to %d: %v", msg.Type, msg.Sender, msg.Target, err)
		wsm.disconnectChan <- msg.Sender
	} else {
		log.Printf("[WS DEBUG] Sent %s from %d to %d", msg.Type, msg.Sender, msg.Target)
	}
}

func (wsm *WebSocketManager) flushBufferedMessages(userID uint64) {
	buffered, ok := wsm.candidateBuffer[userID]
	if !ok {
		return
	}

	conn, exists := wsm.Connections[userID]
	if !exists {
		return
	}

	var remaining []Message
	for _, msg := range buffered {
		if err := wsm.SafeWriteJSON(conn, msg); err != nil {
			log.Printf("[WS ERROR] Failed to flush buffered message to %d: %v", userID, err)
			remaining = append(remaining, msg)
		}
	}

	if len(remaining) > 0 {
		wsm.candidateBuffer[userID] = remaining
	} else {
		delete(wsm.candidateBuffer, userID)
	}
}

func (wsm *WebSocketManager) handlePeerListRequest(msg Message) {
	roomID := msg.RoomID
	userID := msg.Sender

	room, exists := wsm.Rooms[roomID]
	if !exists {
		log.Printf("[WS] Room %d does not exist for user %d", roomID, userID)
		return
	}

	var peerList []uint64
	for uid := range room.Users {
		if uid != userID {
			peerList = append(peerList, uid)
		}
	}

	if conn, ok := wsm.Connections[userID]; ok {
		_ = wsm.SafeWriteJSON(conn, Message{
			Type:   MessageTypePeerList,
			RoomID: roomID,
			Users:  peerList,
			Sender: userID,
		})
	}

	log.Printf("[WS] Sent peer list to user %d: %v", userID, peerList)
}

func (wsm *WebSocketManager) AreInSameRoom(roomID uint64, userIDs []uint64) bool {
	room, exists := wsm.Rooms[roomID]
	if !exists {
		return false
	}

	for _, id := range userIDs {
		if _, ok := room.Users[id]; !ok {
			return false
		}
	}
	return true
}

func (wsm *WebSocketManager) createRoom(hostID uint64) uint64 {
	roomID := wsm.nextRoomID
	wsm.nextRoomID++

	conn, exists := wsm.Connections[hostID]
	if !exists {
		log.Printf("[WS WARNING] Host %d not connected; cannot add to new room", hostID)
		return roomID
	}

	room := &Room{
		ID:        roomID,
		Users:     map[uint64]*Connection{hostID: conn},
		ReadyMap:  map[uint64]bool{hostID: false},
		JoinOrder: []uint64{hostID},
		HostID:    hostID,
	}
	wsm.Rooms[roomID] = room

	log.Printf("[DEBUG] created room %d with host client %d", roomID, hostID)
	return roomID
}

func (wsm *WebSocketManager) disconnectUser(userID uint64) {
	// Close and remove the user's connection
	if conn, exists := wsm.Connections[userID]; exists {
		conn.Conn.Close()
		delete(wsm.Connections, userID)
	}

	// Remove buffered candidate messages for the user
	delete(wsm.candidateBuffer, userID)

	// Remove the user from rooms and notify remaining peers
	for roomID, room := range wsm.Rooms {
		if _, inRoom := room.Users[userID]; inRoom {
			delete(room.Users, userID)

			// Notify remaining peers that this user has left
			for _, peerConn := range room.Users {
				if peerConn != nil {
					_ = peerConn.Conn.WriteJSON(Message{
						Type:   MessageTypePeerLeft,
						RoomID: roomID,
						Sender: userID,
					})
				}
			}

			log.Printf("[WS] User %d removed from room %d", userID, roomID)

			// Delete the room if empty
			if len(room.Users) == 0 {
				delete(wsm.Rooms, roomID)
				log.Printf("[WS] Room %d deleted because it is empty", roomID)
			}
		}
	}

	// TODO: Release the API key associated with this user ID
	// for apiKey, id := range wsm.apiKeyToUserID {
	// 	if id == userID {
	// 		delete(wsm.apiKeyToUserID, apiKey)
	// 		log.Printf("[WS] API key %s released for user %d", apiKey, userID)
	// 		break
	// 	}
	// }
}
