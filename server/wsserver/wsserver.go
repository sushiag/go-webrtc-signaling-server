package wsserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// message type for the readmessages
const (
	TypeCreateRoom  = "create-room"
	TypeJoin        = "join-room"
	TypeOffer       = "offer"
	TypeAnswer      = "answer"
	TypeICE         = "ice-candidate"
	TypeDisconnect  = "disconnect"
	TypeText        = "text"
	TypePeerJoined  = "peer-joined"
	TypeRoomCreated = "room-created"
	TypePeerList    = "peer-list"
	TypePeerReady   = "peer-ready"
	TypeStart       = "start"
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
	ID       uint64
	Users    map[uint64]*websocket.Conn
	ReadyMap map[uint64]bool
}

// this handles connection and room management
type WebSocketManager struct {
	Connections     map[uint64]*websocket.Conn
	Rooms           map[uint64]*Room
	mtx             sync.RWMutex
	upgrader        websocket.Upgrader
	validApiKeys    map[string]bool
	apiKeyToUserID  map[string]uint64
	nextUserID      uint64
	nextRoomID      uint64
	candidateBuffer map[uint64][]Message
}

// this initializes a new manager
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		Connections:     make(map[uint64]*websocket.Conn),
		Rooms:           make(map[uint64]*Room),
		apiKeyToUserID:  make(map[string]uint64),
		candidateBuffer: make(map[uint64][]Message),
		nextUserID:      1,
		nextRoomID:      1,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (wm *WebSocketManager) SetValidApiKeys(keys map[string]bool) {
	wm.validApiKeys = keys
}

func (wm *WebSocketManager) Authenticate(r *http.Request) bool {
	return wm.validApiKeys[r.Header.Get("X-Api-Key")]
}

// this handles the initial API key authentication via HTTP
func (wm *WebSocketManager) AuthHandler(w http.ResponseWriter, r *http.Request) {
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

	if !wm.validApiKeys[payload.ApiKey] {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	wm.mtx.Lock()
	userID, exists := wm.apiKeyToUserID[payload.ApiKey]
	if !exists {
		userID = wm.nextUserID
		wm.apiKeyToUserID[payload.ApiKey] = userID
		wm.nextUserID++
	}
	wm.mtx.Unlock()

	resp := struct {
		UserID     uint64 `json:"userid"`
		SessionKey string `json:"sessionkey"`
	}{
		UserID:     userID,
		SessionKey: uuid.New().String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// this upgrades HTTP to WebSocket and starts communication
func (wm *WebSocketManager) Handler(w http.ResponseWriter, r *http.Request) {
	if !wm.Authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	apiKey := r.Header.Get("X-Api-Key")

	wm.mtx.Lock()
	userID, exists := wm.apiKeyToUserID[apiKey]
	if !exists {
		userID = wm.nextUserID
		wm.apiKeyToUserID[apiKey] = userID
		wm.nextUserID++
	} else if oldConn, ok := wm.Connections[userID]; ok {
		oldConn.Close()
		delete(wm.Connections, userID)
	}
	wm.mtx.Unlock()

	conn, err := wm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}

	wm.mtx.Lock()
	wm.Connections[userID] = conn
	wm.mtx.Unlock()

	log.Printf("[WS] User %d connected", userID)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		wm.readMessages(userID, conn)
	}()

	go func() {
		defer wg.Done()
		wm.sendPings(userID, conn)
	}()

	wg.Wait()

	wm.mtx.Lock()
	delete(wm.Connections, userID)
	wm.mtx.Unlock()
	conn.Close()
	log.Printf("[WS] User %d disconnected", userID)
}

// sends messages to the websocket
func (wm *WebSocketManager) readMessages(userID uint64, conn *websocket.Conn) {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error for user %d: %v", userID, err)
			wm.disconnectUser(userID) // Clean up connection on error
			if ce, ok := err.(*websocket.CloseError); ok {
				switch ce.Code {
				case websocket.CloseNormalClosure:
					log.Printf("[WS] User %d disconnected normally", userID)
				case websocket.CloseGoingAway:
					log.Printf("[WS] User %d is going away", userID)
				case websocket.CloseAbnormalClosure:
					log.Printf("[WS] Abnormal closure for user %d", userID)
				default:
					log.Printf("[WS] User %d closed with code %d: %s", userID, ce.Code, ce.Text)
				}
			} else {
				log.Printf("[WS] Normal disconnect from %d: %v", userID, err)
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("[WS] Invalid message from %d: %v", userID, err)
			continue
		}

		switch msg.Type {
		case TypeCreateRoom:
			log.Printf("[WS] User %d requested to create a room", userID)
			roomID := wm.nextRoomID
			wm.nextRoomID++
			wm.AddUserToRoom(roomID, userID)

			resp := Message{
				Type:   TypeRoomCreated,
				RoomID: roomID,
				Sender: userID,
			}
			if err := conn.WriteJSON(resp); err != nil {
				log.Printf("[WS ERROR] Failed to send room-created to user %d: %v", userID, err)
			} else {
				log.Printf("[WS DEBUG] Sent room-created to user %d: %+v", userID, resp)
			}
		case TypeJoin:
			room, exists := wm.Rooms[msg.RoomID]
			if !exists {
				log.Printf("[WS] Room %d not found for user %d", msg.RoomID, userID)

				closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Room does not exist")
				_ = conn.WriteMessage(websocket.CloseMessage, closeMsg)

				conn.Close()

				wm.mtx.Lock()
				delete(wm.Connections, userID)
				wm.mtx.Unlock()
				return
			}

			room.Users[userID] = wm.Connections[userID]

			for uid, conn := range room.Users {
				if uid != userID {
					conn.WriteJSON(Message{
						Type:   TypePeerJoined,
						RoomID: msg.RoomID,
						Sender: userID,
					})
				}
			}

			conn.WriteJSON(Message{
				Type:   TypeCreateRoom,
				RoomID: msg.RoomID,
			})
			log.Printf("[WS] User %d joined room %d", userID, msg.RoomID)
			wm.sendPeerListToUser(msg.RoomID, userID)

		case TypeOffer, TypeAnswer, TypeICE:
			log.Printf("[WS] Forwarding or buffering %s from %d to %d in room %d", msg.Type, msg.Sender, msg.Target, msg.RoomID)
			wm.forwardOrBuffer(userID, msg)

		case TypePeerList:

			wm.sendPeerListToUser(msg.RoomID, userID)

		case TypeStart:
			log.Printf("[WS] Received 'start' signal from user %d in room %d", msg.Sender, msg.RoomID)

			wm.mtx.Lock()
			room, exists := wm.Rooms[msg.RoomID]
			wm.mtx.Unlock()

			if !exists {
				log.Printf("[WS] Room %d does not exist", msg.RoomID)
				return
			}

			for uid := range room.Users {
				conn := wm.Connections[uid]
				if conn != nil {
					log.Printf("[WS] Closing connection to user %d for P2P start", uid)
					_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Switching to P2P"))
					_ = conn.Close()
				}
			}

			// Clean up the room
			wm.mtx.Lock()
			delete(wm.Rooms, msg.RoomID)
			wm.mtx.Unlock()

		case TypeDisconnect:
			go wm.HandleDisconnect(msg)

		case TypeText:
			log.Printf("[WS] Text from %d: %s", userID, msg.Content)

		default:
			log.Printf("[WS] Unknown message type: %s", msg.Type)
		}
	}
}

func (wm *WebSocketManager) sendPeerListToUser(roomID uint64, userID uint64) {
	wm.mtx.RLock()
	room, exists := wm.Rooms[roomID]
	wm.mtx.RUnlock()

	if exists {
		var peerList []uint64
		for uid := range room.Users {
			if uid != userID {
				peerList = append(peerList, uid)
			}
		}

		if conn, ok := wm.Connections[userID]; ok {
			conn.WriteJSON(Message{
				Type:   TypePeerList,
				RoomID: roomID,
				Users:  peerList,
			})
		}
	}
}
func (wm *WebSocketManager) forwardOrBuffer(senderID uint64, msg Message) {
	wm.mtx.RLock()
	conn, exists := wm.Connections[msg.Target]
	inSameRoom := wm.AreInSameRoom(msg.RoomID, []uint64{msg.Sender, msg.Target})
	wm.mtx.RUnlock()

	log.Printf("[WS DEBUG] forwardOrBuffer type=%s from=%d to=%d exists=%v sameRoom=%v",
		msg.Type, senderID, msg.Target, exists, inSameRoom)

	if !exists || !inSameRoom {
		log.Printf("[WS DEBUG] Buffering %s from %d to %d", msg.Type, msg.Sender, msg.Target)
		wm.mtx.Lock()
		wm.candidateBuffer[msg.Target] = append(wm.candidateBuffer[msg.Target], msg)
		wm.mtx.Unlock()
		return
	}

	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("[WS ERROR] Failed to send %s from %d to %d: %v", msg.Type, msg.Sender, msg.Target, err)
		go wm.HandleDisconnect(msg)
	} else {
		log.Printf("[WS DEBUG] Sent %s from %d to %d", msg.Type, msg.Sender, msg.Target)
	}
}

func (wm *WebSocketManager) AddUserToRoom(roomID, userID uint64) {
	wm.mtx.Lock()
	defer wm.mtx.Unlock()

	room, exists := wm.Rooms[roomID]
	if !exists {
		room = &Room{
			ID:       roomID,
			Users:    make(map[uint64]*websocket.Conn),
			ReadyMap: make(map[uint64]bool),
		}
		wm.Rooms[roomID] = room
	}
	room.Users[userID] = wm.Connections[userID]

	// notify peers
	for uid, conn := range room.Users {
		if uid != userID {
			conn.WriteJSON(Message{
				Type:   TypePeerJoined,
				RoomID: roomID,
				Sender: userID,
			})
		}
	}
	go wm.flushBufferedMessages(userID)

	// send peer list to the newly joined user
	if conn, ok := wm.Connections[userID]; ok {
		var peerList []uint64
		for uid := range room.Users {
			if uid != userID {
				peerList = append(peerList, uid)
			}
		}
		conn.WriteJSON(Message{
			Type:    TypePeerList,
			RoomID:  roomID,
			Content: encodePeers(peerList),
		})
	}
}
func (wm *WebSocketManager) flushBufferedMessages(userID uint64) {
	wm.mtx.Lock()
	defer wm.mtx.Unlock()

	buffered, ok := wm.candidateBuffer[userID]
	if !ok {
		return
	}

	conn, exists := wm.Connections[userID]
	if !exists {
		return
	}

	var remaining []Message
	for _, msg := range buffered {
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("[WS ERROR] Failed to flush buffered message to %d: %v", userID, err)
			remaining = append(remaining, msg)
		}
	}

	if len(remaining) > 0 {
		wm.candidateBuffer[userID] = remaining
	} else {
		delete(wm.candidateBuffer, userID)
	}
}

// creates room for peers
func (wm *WebSocketManager) CreateRoom(userID uint64) uint64 {
	wm.mtx.Lock()
	defer wm.mtx.Unlock()

	roomID := wm.nextRoomID
	wm.nextRoomID++
	wm.Rooms[roomID] = &Room{
		ID:    roomID,
		Users: map[uint64]*websocket.Conn{userID: wm.Connections[userID]},
	}
	return roomID
}

func (wm *WebSocketManager) AreInSameRoom(roomID uint64, userIDs []uint64) bool {
	wm.mtx.RLock()
	defer wm.mtx.RUnlock()

	room, exists := wm.Rooms[roomID]
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

func (wm *WebSocketManager) disconnectUser(userID uint64) {
	wm.mtx.Lock()
	defer wm.mtx.Unlock()

	// Close and remove connection
	if conn, exists := wm.Connections[userID]; exists {
		conn.Close()
		delete(wm.Connections, userID)
	}

	// Remove from candidateBuffer
	delete(wm.candidateBuffer, userID)

	// Remove from rooms and notify others
	for roomID, room := range wm.Rooms {
		if _, inRoom := room.Users[userID]; inRoom {
			delete(room.Users, userID)

			// Notify other peers in the room that this user left
			for _, conn := range room.Users {
				if conn != nil {
					_ = conn.WriteJSON(Message{
						Type:   "peer-left",
						RoomID: roomID,
						Sender: userID,
					})
				}
			}

			log.Printf("[WS] User %d removed from room %d", userID, roomID)

			// If room is empty after removal, delete the room
			if len(room.Users) == 0 {
				delete(wm.Rooms, roomID)
				log.Printf("[WS] Room %d deleted because it is empty", roomID)
			}
		}
	}
}

// handles the disconnection gracefully
func (wm *WebSocketManager) HandleDisconnect(msg Message) {
	if !wm.AreInSameRoom(msg.RoomID, []uint64{msg.Sender, msg.Target}) {
		log.Printf("[WS] Disconnect failed: not in same room")
		return
	}

	wm.mtx.Lock()
	defer wm.mtx.Unlock()

	room, exists := wm.Rooms[msg.RoomID]
	if !exists {
		return
	}

	for _, userID := range []uint64{msg.Sender, msg.Target} {
		if conn, ok := wm.Connections[userID]; ok {
			if conn != nil {
				conn.Close()
			}
			delete(wm.Connections, userID)
			delete(room.Users, userID)
			log.Printf("[WS] User %d disconnected from room %d", userID, msg.RoomID)
		}
	}

	if len(room.Users) == 0 {
		delete(wm.Rooms, msg.RoomID)
		log.Printf("[WS] Room %d deleted", msg.RoomID)
	}
}

// keeps server alive
func (wm *WebSocketManager) sendPings(userID uint64, conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	failures := 0
	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			failures++
			if failures >= 3 {
				log.Printf("[WS] Ping timeout for user %d", userID)
				conn.Close()
				return
			}
		} else {
			failures = 0
		}
	}
}

func encodePeers(peers []uint64) string {
	return fmt.Sprintf("%v", peers)
}
