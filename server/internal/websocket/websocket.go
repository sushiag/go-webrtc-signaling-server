package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// struct for websocket communication
type Message struct {
	Type    string `json:"type"`
	RoomID  uint64 `json:"roomId"`
	Sender  uint64 `json:"sender"`
	Target  uint64 `json:"target,omitempty"`
	Content string `json:"content"`
}

// struct for websocketManager with room logic
type WebSocketManager struct {
	connections map[uint64]*websocket.Conn // stores connections by user ID
	rooms       map[uint64]map[uint64]bool // stores users by room ID
	mtx         sync.RWMutex
	upgrader    websocket.Upgrader
}

// creates a new WebSocket manager
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		connections: make(map[uint64]*websocket.Conn),
		rooms:       make(map[uint64]map[uint64]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true }, // Customize as needed
		},
	}
}

// checks the API key in the WebSocket request
func (wm *WebSocketManager) Authenticate(r *http.Request) bool {
	apiKey := r.Header.Get("X-Api-Key")
	validApiKeys := map[string]bool{
		"valid-api-key-1": true,
		"valid-api-key-2": true,
		"valid-api-key-3": true,
	}

	return validApiKeys[apiKey]
}

// this manages WebSocket connections, using API key for authentication
func (wm *WebSocketManager) Handler(w http.ResponseWriter, r *http.Request) {
	if !wm.Authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// get the API key AND UserID from the request header
	apiKey := r.Header.Get("X-Api-Key")
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		log.Println("[WS] Failed to convert: ", err)
		return
	}

	conn, err := wm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[WS] WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	// stores the client userid in the WebSocket manager
	wm.mtx.Lock()
	wm.connections[userID] = conn
	wm.mtx.Unlock()

	log.Printf("[WS] Client %s connected", apiKey)
	defer func() {
		// removes the client apikey when it disconnects
		wm.mtx.Lock()
		delete(wm.connections, userID)
		wm.mtx.Unlock()
	}()

	// sends the API key to the client
	if err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type": "api_key", "api_key": "%s"}`, apiKey))); err != nil {
		log.Printf("[WS] Failed to send API key to client %s: %v", apiKey, err)
		return
	}

	// read incoming WebSocket messages
	go wm.readMessages(userID, conn)
	wm.sendPings(userID, conn)
}

// readMessages handles signaling messages from clients
func (wm *WebSocketManager) readMessages(userID uint64, conn *websocket.Conn) {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[WS] Read error from %d: %v", userID, err)
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Println("[WS] Invalid message:", err)
			continue
		}

		log.Printf("[WS] Received message from %d of type %s", userID, msg.Type)

		// Handle room join
		if msg.RoomID != 0 {
			wm.AddUserToRoom(msg.RoomID, userID)
		}

		switch msg.Type {
		case "signal":
			if msg.Target != 0 {
				wm.forwardToTarget(msg)
			}
		case "disconnect":
			go wm.HandleDisconnect(msg)
		}
	}
}

// this sends a message to all users in a specific room
func (wm *WebSocketManager) SendToRoom(roomID uint64, sender uint64, msg Message) {
	wm.mtx.RLock()
	defer wm.mtx.RUnlock()

	for userID := range wm.rooms[roomID] {
		if userID == sender {
			continue // for the msg not to send the message back to the sender
		}

		conn, exists := wm.connections[userID]
		if !exists {
			log.Printf("[WS] User %d not connected", userID)
			continue
		}

		// Send the message to the connected user
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("[WS] Failed to send message to %d: %v", userID, err)
		}
	}
}

// forwardToTarget sends the message to the specified target user
func (wm *WebSocketManager) forwardToTarget(msg Message) {
	wm.mtx.RLock()
	targetConn, exists := wm.connections[msg.Target]
	wm.mtx.RUnlock()

	if !exists {
		log.Printf("[WS] Target user %d not connected", msg.Target)
		return
	}

	// Check if both are in the same room
	if !wm.AreInSameRoom(msg.RoomID, msg.Sender, msg.Target) {
		log.Printf("[WS] Users %d and %d not in same room %d", msg.Sender, msg.Target, msg.RoomID)
		return
	}

	if err := targetConn.WriteJSON(msg); err != nil {
		log.Printf("[WS] Failed to forward message to %d: %v", msg.Target, err)
	}
}

// AddUserToRoom adds a user to a specific room
func (wm *WebSocketManager) AddUserToRoom(roomID uint64, userID uint64) {
	wm.mtx.Lock()
	defer wm.mtx.Unlock()

	// Initialize the room if it doesn't exist
	if _, exists := wm.rooms[roomID]; !exists {
		wm.rooms[roomID] = make(map[uint64]bool)
	}
	// Add user to the room
	wm.rooms[roomID][userID] = true
	log.Printf("[WS] User %d added to room %d", userID, roomID)
}

// AreInSameRoom checks if two users are in the same room
func (wm *WebSocketManager) AreInSameRoom(roomID uint64, user1 uint64, user2 uint64) bool {
	wm.mtx.RLock()
	defer wm.mtx.RUnlock()

	// check if both users are in the room
	if room, exists := wm.rooms[roomID]; exists {
		return room[user1] && room[user2]
	}
	return false
}

// sendPings sends periodic pings to the client to keep the connection alive
func (wm *WebSocketManager) sendPings(userID uint64, conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	pingFailures := 0
	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			log.Println("[WS] Failed to send ping:", err)
			pingFailures++
			if pingFailures >= 3 {
				log.Printf("[WS] Client %d failed to respond to pings. Closing connection.", userID)
				conn.Close()
				return
			}
		} else {
			pingFailures = 0 // Reset on successful ping
		}
	}
}

// HandleDisconnect handles graceful closing of WebSocket connections once P2P is established
func (wm *WebSocketManager) HandleDisconnect(msg Message) {
	if !wm.AreInSameRoom(msg.RoomID, msg.Sender, msg.Target) {
		log.Printf("[WS] Disconnect failed: %d and %d not in room %d", msg.Sender, msg.Target, msg.RoomID)
		return
	}

	wm.mtx.Lock()
	defer wm.mtx.Unlock()

	// Close sender's connection
	if conn, exists := wm.connections[msg.Sender]; exists {
		conn.Close()
		delete(wm.connections, msg.Sender)
		delete(wm.rooms[msg.RoomID], msg.Sender)
		log.Printf("[WS] Disconnected sender %d", msg.Sender)
	}

	// Close target's connection
	if conn, exists := wm.connections[msg.Target]; exists {
		conn.Close()
		delete(wm.connections, msg.Target)
		delete(wm.rooms[msg.RoomID], msg.Target)
		log.Printf("[WS] Disconnected target %d", msg.Target)
	}

	// Optionally delete the room if empty
	if len(wm.rooms[msg.RoomID]) == 0 {
		delete(wm.rooms, msg.RoomID)
		log.Printf("[WS] Room %d deleted", msg.RoomID)
	}
}
