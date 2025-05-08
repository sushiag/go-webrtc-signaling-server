package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"

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
	mtx             sync.RWMutex
	writeLock       sync.Mutex
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

func (wsm *WebSocketManager) SafeWriteJSON(conn *websocket.Conn, v interface{}) error {
	wsm.writeLock.Lock()
	defer wsm.writeLock.Unlock()
	return conn.WriteJSON(v)
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
		UserID uint64 `json:"userid"`
	}{
		UserID: userID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

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
	}
	wm.mtx.Unlock()

	conn, err := wm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}

	wm.mtx.Lock()
	if _, ok := wm.Connections[userID]; ok {
		wm.mtx.Unlock()
		log.Printf("[WS] Duplicate connection attempt for user %d. Denying new connection.", userID)

		// disconnect second user
		closeMsg := websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Duplicate connection detected")
		conn.WriteMessage(websocket.CloseMessage, closeMsg)
		conn.Close()
		return
	}
	wm.Connections[userID] = conn
	wm.mtx.Unlock()

	// Flush buffered candidates if any
	wm.flushBufferedMessages(userID)

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

	// Cleanup on exit
	wm.mtx.Lock()
	delete(wm.Connections, userID)
	wm.mtx.Unlock()
	// conn.Close() commenting out for testing

	log.Printf("[WS] User %d disconnected", userID)
}

// sends messages to the websocket
func (wm *WebSocketManager) readMessages(userID uint64, conn *websocket.Conn) {
	defer func() {
		wm.disconnectUser(userID)
		conn.Close()
		log.Printf("[WS] User %d disconnected", userID)
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error for user %d: %v", userID, err)
			wm.disconnectUser(userID) // clean up connection on error
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

			roomID := wm.CreateRoom(userID)

			// Set initial host
			wm.mtx.Lock()
			room := wm.Rooms[roomID]
			room.HostID = userID
			room.JoinOrder = append(room.JoinOrder, userID)
			wm.mtx.Unlock()

			resp := Message{
				Type:   TypeRoomCreated,
				RoomID: roomID,
				Sender: userID,
			}
			if err := conn.WriteJSON(resp); err != nil {
				log.Printf("[WS] Failed to send room-created to user %d: %v", userID, err)
			} else {
				log.Printf("[WS] Sent room-created to user %d: %+v", userID, resp)
			}

		case TypeJoin:
			wm.mtx.Lock()
			room, exists := wm.Rooms[msg.RoomID]
			if !exists {
				wm.mtx.Unlock()
				log.Printf("[WS] Room %d not found for user %d", msg.RoomID, userID)
				closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Room does not exist")
				_ = conn.WriteMessage(websocket.CloseMessage, closeMsg)
				return
			}

			room.Users[userID] = conn
			room.JoinOrder = append(room.JoinOrder, userID)

			// If no host set yet (paranoia check)
			if room.HostID == 0 {
				room.HostID = userID
			}
			wm.mtx.Unlock()

			// Notify others
			for uid, peerConn := range room.Users {
				if uid != userID {
					_ = peerConn.WriteJSON(Message{
						Type:   TypePeerJoined,
						RoomID: msg.RoomID,
						Sender: userID,
					})
				}
			}

			_ = conn.WriteJSON(Message{
				Type:   TypeCreateRoom,
				RoomID: msg.RoomID,
			})

			log.Printf("[WS] User %d joined room %d", userID, msg.RoomID)
			wm.sendPeerListToUser(msg.RoomID, userID)

		case TypeOffer, TypeAnswer, TypeICE:
			log.Printf("[WS] Forwarding %s from %d to %d in room %d", msg.Type, msg.Sender, msg.Target, msg.RoomID)
			wm.forwardOrBuffer(userID, msg)

		case TypePeerList, "peer-list-request":

			wm.sendPeerListToUser(msg.RoomID, userID)

		case TypeStart:
			log.Printf("[WS] Received 'start' from user %d in room %d", msg.Sender, msg.RoomID)

			wm.mtx.Lock()
			room, exists := wm.Rooms[msg.RoomID]
			wm.mtx.Unlock()

			if !exists {
				log.Printf("[WS] Room %d does not exist", msg.RoomID)
				return
			}

			for uid, peerConn := range room.Users {
				if peerConn != nil {
					log.Printf("[WS] Closing connection to user %d for P2P switch", uid)
					_ = peerConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Switching to P2P"))
					_ = peerConn.Close()
				}
			}

			wm.mtx.Lock()
			delete(wm.Rooms, msg.RoomID)
			wm.mtx.Unlock()

		case TypeDisconnect:
			go wm.HandleDisconnect(msg)

		case TypeText:
			log.Printf("[WS] Text from %d: %s", userID, msg.Content)
		case "start-session":

			log.Printf("[WS] Received start-session from peer %d", userID)
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

	for uid, conn := range room.Users {
		if uid != userID {
			conn.WriteJSON(Message{
				Type:   TypePeerJoined,
				RoomID: roomID,
				Sender: userID,
			})
		}
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

func (wm *WebSocketManager) maybeDeleteRoom(roomID uint64) {
	if room, ok := wm.Rooms[roomID]; ok && len(room.Users) == 0 {
		delete(wm.Rooms, roomID)
		log.Printf("[WS] Room %d deleted because it is empty", roomID)
	}
}

func (wm *WebSocketManager) disconnectUser(userID uint64) {
	wm.mtx.Lock()
	defer wm.mtx.Unlock()

	// Close and remove connection
	if conn, exists := wm.Connections[userID]; exists {
		conn.Close()
		delete(wm.Connections, userID)
	}

	// remove from candidateBuffer
	delete(wm.candidateBuffer, userID)

	// remove from rooms and notify others
	for roomID, room := range wm.Rooms {
		if _, inRoom := room.Users[userID]; inRoom {
			delete(room.Users, userID)

			// notify peers
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

			// delete room if empty
			if len(room.Users) == 0 {
				delete(wm.Rooms, roomID)
				log.Printf("[WS] Room %d deleted because it is empty", roomID)
			}
		}
	}

	// released  apikey from the serveer
	for apiKey, id := range wm.apiKeyToUserID {
		if id == userID {
			delete(wm.apiKeyToUserID, apiKey)
			log.Printf("[WS] API key %s released for user %d", apiKey, userID)
			break
		}
	}

}

// handles the disconnection gracefully
func (wm *WebSocketManager) HandleDisconnect(msg Message) {
	roomID := msg.RoomID
	userID := msg.Sender

	wm.mtx.Lock()
	room, exists := wm.Rooms[roomID]
	if !exists {
		wm.mtx.Unlock()
		return
	}

	delete(room.Users, userID)

	// remove from JoinOrder
	for i, id := range room.JoinOrder {
		if id == userID {
			room.JoinOrder = slices.Delete(room.JoinOrder, i, i+1)
			break
		}
	}

	// reassign host if needed
	if room.HostID == userID {
		if len(room.JoinOrder) > 0 {
			newHostID := room.JoinOrder[0]
			room.HostID = newHostID
			newHostConn := room.Users[newHostID]

			if newHostConn != nil {
				_ = newHostConn.WriteJSON(Message{
					Type:   "host-info",
					Sender: newHostID,
				})
			}
		} else {
			room.HostID = 0
		}
	}

	// clean up empty room
	if len(room.Users) == 0 {
		delete(wm.Rooms, roomID)
	}
	wm.mtx.Unlock()
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

func StartServer(port string) (*http.Server, string) {
	// the server address and port
	host := os.Getenv("SERVER_HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	// Combine the address and port
	serverUrl := fmt.Sprintf("%s:%s", host, port)

	listener, err := net.Listen("tcp", serverUrl)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}

	// update the URL with the assigned port
	serverUrl = listener.Addr().String()
	log.Printf("[SERVER] Starting on %s\n", serverUrl)

	apiKeyPath := os.Getenv("APIKEY_PATH")
	if apiKeyPath == "" {
		apiKeyPath = "apikeys.txt"
	}

	manager := NewWebSocketManager()

	apiKeys, err := LoadValidApiKeys(apiKeyPath)
	if err != nil {
		log.Printf("[SERVER] Failed to load API keys: %v", err)
	}
	manager.SetValidApiKeys(apiKeys)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth", manager.AuthHandler)
	mux.HandleFunc("/ws", manager.Handler)

	server := &http.Server{
		Addr:    serverUrl,
		Handler: mux,
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("[SERVER] Error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	return server, serverUrl
}

// LoadValidApiKeys loads API keys from a file
func LoadValidApiKeys(path string) (map[string]bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	keys := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		keys[scanner.Text()] = true
	}
	return keys, scanner.Err()
}
