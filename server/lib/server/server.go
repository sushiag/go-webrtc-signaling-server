package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// message type for the readmessages
const (
	TypeCreateRoom      = "create-room"
	TypeJoin            = "join-room"
	TypeOffer           = "offer"
	TypeAnswer          = "answer"
	TypeICE             = "ice-candidate"
	TypeDisconnect      = "disconnect"
	TypeText            = "text"
	TypePeerJoined      = "room-joined"
	TypeRoomCreated     = "room-created"
	TypePeerList        = "peer-list"
	TypePeerReady       = "peer-ready"
	TypeStart           = "start"
	TypeStartP2P        = "start-session"
	TypePeerListRequest = "peer-list-request"
)

type Message struct {
	Type      string   `json:"type"`
	APIKey    string   `json:"apikey,omitempty"`
	Content   string   `json:"content,omitempty"`
	RoomID    uint64   `json:"roomid,omitempty"`
	Sender    uint64   `json:"from,omitempty"`
	Target    uint64   `json:"to,omitempty"`
	SDP       string   `json:"sdp,omitempty"`
	Candidate string   `json:"candidate,omitempty"`
	UserID    uint64   `json:"userid,omitempty"`
	Users     []uint64 `json:"users,omitempty"`
}

// struct for a group of connected users
type Room struct {
	ID        uint64
	Users     map[uint64]*websocket.Conn
	ReadyMap  map[uint64]bool
	JoinOrder []uint64
	HostID    uint64
}

// this handles connection and room management
type WebSocketManager struct {
	Connections     map[uint64]*websocket.Conn
	Rooms           map[uint64]*Room
	validApiKeys    map[string]bool
	apiKeyToUserID  map[string]uint64
	nextUserID      uint64
	nextRoomID      uint64
	candidateBuffer map[uint64][]Message
	messageChan     chan Message
	disconnectChan  chan uint64
}

// this initializes a new manager
func NewWebSocketManager() *WebSocketManager {
	wsm := &WebSocketManager{
		Connections:     make(map[uint64]*websocket.Conn),
		Rooms:           make(map[uint64]*Room),
		apiKeyToUserID:  make(map[string]uint64),
		candidateBuffer: make(map[uint64][]Message),
		nextUserID:      1,
		nextRoomID:      1,
		messageChan:     make(chan Message),
		disconnectChan:  make(chan uint64),
	}
	go wsm.run()
	return wsm
}

func (wsm *WebSocketManager) run() {
	for {
		select {
		case msg := <-wsm.messageChan:
			wsm.handleMessage(msg)
		case userID := <-wsm.disconnectChan:
			wsm.disconnectUser(userID)
		}
	}
}

func (wsm *WebSocketManager) SafeWriteJSON(conn *websocket.Conn, v interface{}) error {
	return conn.WriteJSON(v)
}

func (wsm *WebSocketManager) SetValidApiKeys(keys map[string]bool) {
	wsm.validApiKeys = keys
}

func (wsm *WebSocketManager) Authenticate(r *http.Request) bool {
	return wsm.validApiKeys[r.Header.Get("X-Api-Key")]
}

// this handles the initial API key authentication via HTTP
func (wsm *WebSocketManager) AuthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		ApiKey string `json:"apikey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if !wsm.validApiKeys[payload.ApiKey] {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := wsm.nextUserID
	wsm.apiKeyToUserID[payload.ApiKey] = userID
	wsm.nextUserID++

	resp := struct {
		UserID uint64 `json:"userid"`
	}{
		UserID: userID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (wsm *WebSocketManager) Handler(w http.ResponseWriter, r *http.Request) {
	if !wsm.Authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	apiKey := r.Header.Get("X-Api-Key")
	userID := wsm.apiKeyToUserID[apiKey]

	conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}

	if _, ok := wsm.Connections[userID]; ok {
		log.Printf("[WS] Duplicate connection attempt for user %d. Denying new connection.", userID)
		closeMsg := websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Duplicate connection detected")
		conn.WriteMessage(websocket.CloseMessage, closeMsg)
		conn.Close()
		return
	}
	wsm.Connections[userID] = conn

	// Flush buffered candidates if any
	wsm.flushBufferedMessages(userID)

	log.Printf("[WS] User %d connected", userID)

	go wsm.readMessages(userID, conn)
	go wsm.sendPings(userID, conn)
}

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
		log.Printf("[WS] User %d joined room %d", msg.Sender, msg.RoomID)

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
		log.Printf("[WS] Received start-session from peer %d", msg.Sender)

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

// handleStart handles the start message - closes connections and cleans the room
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
			_ = peerConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Switching to P2P"))
			_ = peerConn.Close()
		}
	}

	delete(wsm.Rooms, roomID)
}

func (wsm *WebSocketManager) AddUserToRoom(roomID, userID uint64) {
	room, exists := wsm.Rooms[roomID]
	if !exists {
		room = &Room{
			ID:       roomID,
			Users:    make(map[uint64]*websocket.Conn),
			ReadyMap: make(map[uint64]bool),
		}
		wsm.Rooms[roomID] = room
	}
	room.Users[userID] = wsm.Connections[userID]

	// semds notif that other users in the room the has joined
	for uid, conn := range room.Users {
		if uid != userID {
			_ = wsm.SafeWriteJSON(conn, Message{
				Type:   TypePeerJoined,
				RoomID: roomID,
				Sender: userID,
			})
		}
	}
	if _, ok := wsm.Connections[userID]; !ok {
		log.Printf("[WS WARNING] User %d not connected, cannot join room", userID)
		return
	}
	log.Printf("[WS] User %d joined room %d", userID, roomID)
}

func (wsm *WebSocketManager) forwardOrBuffer(senderID uint64, msg Message) {
	// Check if the target connection exists and if both users are in the same room
	conn, exists := wsm.Connections[msg.Target]
	inSameRoom := wsm.AreInSameRoom(msg.RoomID, []uint64{msg.Sender, msg.Target})

	log.Printf("[WS DEBUG] forwardOrBuffer type=%s from=%d to=%d exists=%v sameRoom=%v",
		msg.Type, senderID, msg.Target, exists, inSameRoom)

	if !exists || !inSameRoom {
		log.Printf("[WS DEBUG] Buffering %s from %d to %d", msg.Type, msg.Sender, msg.Target)
		// Buffer the message for the target user
		wsm.candidateBuffer[msg.Target] = append(wsm.candidateBuffer[msg.Target], msg)
		return
	}

	// Send the message directly if the connection exists
	if err := wsm.SafeWriteJSON(conn, msg); err != nil {
		log.Printf("[WS ERROR] Failed to send %s from %d to %d: %v", msg.Type, msg.Sender, msg.Target, err)
		// Handle disconnection if sending fails
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

//func (wsm *WebSocketManager) sendPeerListToUser(roomID uint64, userID uint64) {
//	peerListRequest := Message{
//		Type:   TypePeerList,
//		RoomID: roomID,
//		Sender: userID,
//	}
//	wsm.messageChan <- peerListRequest
//}

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
			Type:   TypePeerList,
			RoomID: roomID,
			Users:  peerList,
		})
	}
	log.Printf("[WS] Sent User %d: %d", userID, peerList)

}

func (wsm *WebSocketManager) AreInSameRoom(roomID uint64, userIDs []uint64) bool {
	room, exists := wsm.Rooms[roomID]
	if !exists {
		return false
	}

	for _, uid := range userIDs {
		if _, ok := room.Users[uid]; !ok {
			return false
		}
	}
	return true
}

// creates room for peers
func (wsm *WebSocketManager) CreateRoom(userID uint64) uint64 {
	roomID := wsm.nextRoomID
	wsm.nextRoomID++
	wsm.Rooms[roomID] = &Room{
		ID:    roomID,
		Users: map[uint64]*websocket.Conn{userID: wsm.Connections[userID]},
	}
	return roomID
}

func (wsm *WebSocketManager) disconnectUser(userID uint64) {
	// Close and remove the user's connection
	if conn, exists := wsm.Connections[userID]; exists {
		conn.Close()
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
					_ = peerConn.WriteJSON(Message{
						Type:   "peer-left",
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

	// Release the API key associated with this user ID
	for apiKey, id := range wsm.apiKeyToUserID {
		if id == userID {
			delete(wsm.apiKeyToUserID, apiKey)
			log.Printf("[WS] API key %s released for user %d", apiKey, userID)
			break
		}
	}
}

func (wsm *WebSocketManager) sendPings(userID uint64, conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	conn.SetPongHandler(func(string) error {
		return nil
	})

	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
			log.Printf("[WS] Ping to user %d failed: %v", userID, err)
			wsm.disconnectChan <- userID
			return
		}
	}
}
