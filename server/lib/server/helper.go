package server

import (
	"log"

	smsg "signaling-msgs"
)

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

	conn, ok := wsm.Connections[joiningUserID]
	if !ok {
		log.Printf("[ERROR] No active connection for user %d", joiningUserID)
		return
	}
	room.Users[joiningUserID] = conn
	room.ReadyMap[joiningUserID] = false
	room.JoinOrder = append(room.JoinOrder, joiningUserID)

	var peers []uint64
	for uid := range room.Users {
		if uid != joiningUserID {
			peers = append(peers, uid)
		}
	}

	if conn != nil {
		var clientsInRoom []uint64
		for uid := range room.Users {
			clientsInRoom = append(clientsInRoom, uid)
		}

		_ = wsm.SafeWriteJSON(conn, smsg.MessageAnyPayload{
			MsgType: smsg.RoomJoined,
			Payload: smsg.RoomJoinedPayload{
				RoomID:        roomID,
				ClientsInRoom: clientsInRoom},
		})
	}

	// TODO: notify everyone else in the room that a peer joined
	log.Printf("[WS] User %d joined room %d", joiningUserID, roomID)
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

	// Remove the user from rooms and notify remaining peers
	for roomID, room := range wsm.Rooms {
		if _, inRoom := room.Users[userID]; inRoom {
			delete(room.Users, userID)

			// TODO: Notify remaining peers that this user has left

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
