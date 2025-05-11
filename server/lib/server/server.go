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
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// message type for the readmessages
const (
	TypeCreateRoom   = "create-room"
	TypeJoin         = "join-room"
	TypeOffer        = "offer"
	TypeAnswer       = "answer"
	TypeICE          = "ice-candidate"
	TypeDisconnect   = "disconnect"
	TypeText         = "text"
	TypePeerJoined   = "peer-joined"
	TypeRoomCreated  = "room-created"
	TypePeerList     = "peer-list"
	TypePeerReady    = "peer-ready"
	TypeStartSession = "start-session"
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
	mu        sync.Mutex
}

type WebSocketManager struct {
	Connections     sync.Map
	Rooms           sync.Map
	validApiKeys    map[string]bool
	apiKeyToUserID  sync.Map
	nextUserID      uint64
	nextRoomID      uint64
	candidateBuffer sync.Map
	upgrader        websocket.Upgrader
	mtx             sync.Mutex
}

// NewWebSocketManager initializes a new WebSocketManager
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		validApiKeys:    make(map[string]bool),
		apiKeyToUserID:  sync.Map{},
		candidateBuffer: sync.Map{},
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

// AuthHandler handles the initial API key authentication via HTTP
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

	// Locking and handling apiKeyToUserID with sync.Map
	userIDInterface, exists := wm.apiKeyToUserID.Load(payload.ApiKey)
	var userID uint64
	if !exists {
		userID = atomic.AddUint64(&wm.nextUserID, 1)
		wm.apiKeyToUserID.Store(payload.ApiKey, userID)
		wm.nextUserID++
	} else {
		userID = userIDInterface.(uint64)
	}

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

	// Access userID in a thread-safe manner using sync.Map
	userIDInterface, exists := wm.apiKeyToUserID.Load(apiKey)
	var userID uint64
	if !exists {
		userID = wm.nextUserID
		wm.apiKeyToUserID.Store(apiKey, userID)
		wm.nextUserID++
	} else {
		userID = userIDInterface.(uint64)
	}

	conn, err := wm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}

	// Lock the Connections map to ensure no duplicate connections
	_, loaded := wm.Connections.LoadOrStore(userID, conn)
	if loaded {
		log.Printf("[WS] Duplicate connection attempt for user %d. Denying new connection.", userID)
		closeMsg := websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Duplicate connection detected")
		conn.WriteMessage(websocket.CloseMessage, closeMsg)
		conn.Close()
		return
	}

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
	wm.Connections.Delete(userID)
	conn.Close()

	log.Printf("[WS] User %d disconnected", userID)
}
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
			wm.disconnectUser(userID)
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

			roomInterface, _ := wm.Rooms.LoadOrStore(roomID, &Room{
				ID:        roomID,
				HostID:    userID,
				JoinOrder: []uint64{userID},
				Users:     make(map[uint64]*websocket.Conn),
				ReadyMap:  make(map[uint64]bool),
			})
			room := roomInterface.(*Room)

			// Add user to room's connection list
			if connIface, ok := wm.Connections.Load(userID); ok {
				room.Users[userID] = connIface.(*websocket.Conn)
			}

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
			roomInterface, exists := wm.Rooms.Load(msg.RoomID)
			if !exists {
				log.Printf("[WS] Room %d not found for user %d", msg.RoomID, userID)
				closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Room does not exist")
				_ = conn.WriteMessage(websocket.CloseMessage, closeMsg)
				return
			}

			room := roomInterface.(*Room)
			room.Users[userID] = conn
			room.JoinOrder = append(room.JoinOrder, userID)

			// Ensure host is set
			if room.HostID == 0 {
				room.HostID = userID
			}

			// Notify other users in the room
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

		case TypeStartSession:
			log.Printf("[WS] Received 'start' from user %d in room %d", msg.Sender, msg.RoomID)

			roomInterface, exists := wm.Rooms.Load(msg.RoomID)
			if !exists {
				log.Printf("[WS] Room %d does not exist", msg.RoomID)
				return
			}

			room := roomInterface.(*Room)
			for uid, peerConn := range room.Users {
				if peerConn != nil {
					log.Printf("[WS] Closing connection to user %d for P2P switch", uid)
					_ = peerConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Switching to P2P"))
					_ = peerConn.Close()
				}
			}

			wm.Rooms.Delete(msg.RoomID)

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
	roomInterface, exists := wm.Rooms.Load(roomID)
	if !exists {
		return
	}

	room := roomInterface.(*Room)

	var peerList []uint64
	for uid := range room.Users {
		if uid != userID {
			peerList = append(peerList, uid)
		}
	}

	connInterface, ok := wm.Connections.Load(userID)
	if ok {
		conn := connInterface.(*websocket.Conn)
		conn.WriteJSON(Message{
			Type:   TypePeerList,
			RoomID: roomID,
			Users:  peerList,
		})
	}
}
func (wm *WebSocketManager) forwardOrBuffer(senderID uint64, msg Message) {
	connInterface, exists := wm.Connections.Load(msg.Target)
	inSameRoom := wm.AreInSameRoom(msg.RoomID, []uint64{msg.Sender, msg.Target})

	log.Printf("[WS DEBUG] forwardOrBuffer type=%s from=%d to=%d exists=%v sameRoom=%v",
		msg.Type, senderID, msg.Target, exists, inSameRoom)

	if !exists || !inSameRoom {
		log.Printf("[WS DEBUG] Buffering %s from %d to %d", msg.Type, msg.Sender, msg.Target)
		// Safely store the buffered message
		bufferInterface, _ := wm.candidateBuffer.LoadOrStore(msg.Target, []Message{})
		buffered := bufferInterface.([]Message)
		buffered = append(buffered, msg)
		wm.candidateBuffer.Store(msg.Target, buffered)
		return
	}

	conn := connInterface.(*websocket.Conn)
	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("[WS ERROR] Failed to send %s from %d to %d: %v", msg.Type, msg.Sender, msg.Target, err)
		go wm.HandleDisconnect(msg)
	} else {
		log.Printf("[WS DEBUG] Sent %s from %d to %d", msg.Type, msg.Sender, msg.Target)
	}
}

func (wm *WebSocketManager) AddUserToRoom(roomID, userID uint64) {
	// Load or create the room
	roomInterface, _ := wm.Rooms.LoadOrStore(roomID, &Room{
		ID:        roomID,
		HostID:    userID,
		JoinOrder: []uint64{userID},
		Users:     make(map[uint64]*websocket.Conn),
		ReadyMap:  make(map[uint64]bool),
	})
	room := roomInterface.(*Room) // ssafe type assertion

	// Load the user's connection
	connIface, ok := wm.Connections.Load(userID)
	if !ok {
		log.Printf("[WS] User %d not found in connections", userID)
		return
	}

	conn := connIface.(*websocket.Conn)
	room.Users[userID] = conn

	// Lock the room mutex to prevent concurrent writes
	room.mu.Lock()
	defer room.mu.Unlock()

	// Alert other users in the room that this user has joined
	for uid, userConn := range room.Users {
		if uid != userID {
			if err := userConn.WriteJSON(Message{
				Type:   TypePeerJoined,
				RoomID: roomID,
				Sender: userID,
			}); err != nil {
				log.Printf("[WS] Failed to send message to user %d: %v", uid, err)
			}
		}
	}

	log.Printf("[WS] User %d added to room %d", userID, roomID)
}

func (wm *WebSocketManager) flushBufferedMessages(userID uint64) {
	bufferInterface, ok := wm.candidateBuffer.Load(userID)
	if !ok {
		return
	}

	buffered := bufferInterface.([]Message)
	connInterface, exists := wm.Connections.Load(userID)
	if !exists {
		return
	}

	conn := connInterface.(*websocket.Conn)
	var remaining []Message
	for _, msg := range buffered {
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("[WS ERROR] Failed to flush buffered message to %d: %v", userID, err)
			remaining = append(remaining, msg)
		}
	}

	if len(remaining) > 0 {
		wm.candidateBuffer.Store(userID, remaining)
	} else {
		wm.candidateBuffer.Delete(userID)
	}
}

func (wm *WebSocketManager) CreateRoom(userID uint64) uint64 {
	roomID := wm.nextRoomID
	wm.nextRoomID++

	connIface, ok := wm.Connections.Load(userID)
	if !ok {
		// Handle missing connection (e.g., panic, return 0, or log error)
		panic("user connection not found")
	}

	conn := connIface.(*websocket.Conn)

	room := &Room{
		ID:    roomID,
		Users: map[uint64]*websocket.Conn{userID: conn},
	}

	wm.Rooms.Store(roomID, room)
	return roomID
}

func (wm *WebSocketManager) AreInSameRoom(roomID uint64, userIDs []uint64) bool {
	roomInterface, exists := wm.Rooms.Load(roomID)
	if !exists {
		return false
	}

	room := roomInterface.(*Room)
	for _, uid := range userIDs {
		if _, ok := room.Users[uid]; !ok {
			return false
		}
	}
	return true
}

func (wm *WebSocketManager) disconnectUser(userID uint64) {
	// Close and remove connection
	if connInterface, exists := wm.Connections.Load(userID); exists {
		conn := connInterface.(*websocket.Conn)
		conn.Close()
		wm.Connections.Delete(userID)
	}

	// remove from candidateBuffer
	wm.candidateBuffer.Delete(userID)

	// remove from rooms and notify others
	wm.Rooms.Range(func(_, roomInterface interface{}) bool {
		room := roomInterface.(*Room)
		if _, inRoom := room.Users[userID]; inRoom {
			delete(room.Users, userID)

			// notify peers
			for _, conn := range room.Users {
				if conn != nil {
					_ = conn.WriteJSON(Message{
						Type:   "peer-left",
						Sender: userID,
					})
				}
			}

			log.Printf("[WS] User %d removed from room", userID)

			// delete room if empty
			if len(room.Users) == 0 {
				wm.Rooms.Delete(room.ID)
				log.Printf("[WS] Room %d deleted because it is empty", room.ID)
			}
		}
		return true
	})

	// release API key
	wm.apiKeyToUserID.Range(func(key, value interface{}) bool {
		apiKey := key.(string)
		id := value.(uint64)
		if id == userID {
			wm.apiKeyToUserID.Delete(apiKey)
			log.Printf("[WS] API key %s released for user %d", apiKey, userID)
			return false // stop iteration
		}
		return true
	})

}

func (wm *WebSocketManager) HandleDisconnect(msg Message) {
	roomInterface, exists := wm.Rooms.Load(msg.RoomID)
	if !exists {
		return
	}

	room := roomInterface.(*Room)
	wm.mtx.Lock()
	defer wm.mtx.Unlock()

	delete(room.Users, msg.Sender)

	// remove from JoinOrder
	room.JoinOrder = slices.Delete(room.JoinOrder, slices.Index(room.JoinOrder, msg.Sender), slices.Index(room.JoinOrder, msg.Sender)+1)

	// reassign host if needed
	if room.HostID == msg.Sender {
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
		wm.Rooms.Delete(msg.RoomID)
	}
}

func (wm *WebSocketManager) sendPings(userID uint64, conn *websocket.Conn) {
	ticker := time.NewTicker(5 * time.Second)
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

	time.Sleep(10 * time.Millisecond)

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
