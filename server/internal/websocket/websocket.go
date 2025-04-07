package websocket

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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
	connections    map[uint64]*websocket.Conn // stores connections by user ID
	rooms          map[uint64]map[uint64]bool // stores users by room ID
	mtx            sync.RWMutex
	upgrader       websocket.Upgrader
	validApiKeys   map[string]bool
	apiKeyToUserID map[string]uint64 // maps API key to assigned user ID
	nextUserID     uint64            // counter for the next user ID to assign
}

// creates a new WebSocket manager
func NewWebSocketManager(api_keys_path string) *WebSocketManager {
	return &WebSocketManager{
		connections: make(map[uint64]*websocket.Conn),
		rooms:       make(map[uint64]map[uint64]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true }, // Customize as needed
		},
		apiKeyToUserID: make(map[string]uint64),
		nextUserID:     1, // Start user IDs from 1
	}
}

// checks the API key in the WebSocket request
func (wm *WebSocketManager) Authenticate(r *http.Request) bool {
	apiKey := r.Header.Get("X-Api-Key")
	return wm.validApiKeys[apiKey]
}

func (wm *WebSocketManager) SetValidApiKeys(keys map[string]bool) {
	wm.validApiKeys = keys
}

func LoadValidApiKeys(api_keys_path string) (map[string]bool, error) {
	validApiKeys := make(map[string]bool)

	// Open the .txt file
	file, err := os.Open("apikeys.txt") // Update the file path if necessary
	if err != nil {
		return nil, fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Add each API key to the map
		validApiKeys[scanner.Text()] = true
	}

	// Check for any errors encountered while reading the file
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("could not read file: %v", err)
	}

	return validApiKeys, nil
}

// this manages WebSocket connections, using API key for authentication
func (wm *WebSocketManager) Handler(w http.ResponseWriter, r *http.Request) {
	if !wm.Authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	apiKey := r.Header.Get("X-Api-Key")

	// Check if the API key has already been assigned a userID
	wm.mtx.Lock()
	userID, exists := wm.apiKeyToUserID[apiKey]
	if !exists {
		// Assign the next available userID
		userID = wm.nextUserID
		wm.apiKeyToUserID[apiKey] = userID
		wm.nextUserID++ // Increment for the next user
	}
	wm.mtx.Unlock()

	conn, err := wm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[WS] WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	// stores the client userID in the WebSocket manager
	wm.mtx.Lock()
	wm.connections[userID] = conn
	wm.mtx.Unlock()

	log.Printf("[WS] User %d connected", userID)

	// this removes the client userID when it disconnects
	defer func() {
		wm.mtx.Lock()
		delete(wm.connections, userID)
		wm.mtx.Unlock()
		log.Printf("[WS] User %d disconnected", userID)
	}()

	// Read incoming WebSocket messages
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

		switch msg.Type {
		case "join":
			wm.AddUserToRoom(msg.RoomID, userID)
		case "offer", "answer", "ice-candidate":
			if msg.Target != 0 {
				wm.forwardToTarget(msg)
			}
		case "disconnect":
			go wm.HandleDisconnect(msg)
		case "text":
			fmt.Printf("[SERVER] Received text message: %s\n", msg.Content)
		default:
			log.Printf("[WS] Unknown message type: %s from user %d", msg.Type, userID)
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

	if _, exists := wm.rooms[roomID]; !exists {
		wm.rooms[roomID] = make(map[uint64]bool)
	}
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

func (wm *WebSocketManager) Shutdown() {
	wm.mtx.Lock()
	defer wm.mtx.Unlock()

	for userID, conn := range wm.connections {
		conn.Close()
		delete(wm.connections, userID)
	}
	log.Println("[WS] All WebSocket connections closed")
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
				log.Printf("[WS] Client %d failed to respond to pings after 3 attempts. Closing connection.", userID)
				conn.Close()
				return
			}

		} else {
			pingFailures = 0
		}
	}
}

// this handles graceful closing of WebSocket connections once P2P is established
func (wm *WebSocketManager) HandleDisconnect(msg Message) {
	if !wm.AreInSameRoom(msg.RoomID, msg.Sender, msg.Target) {
		log.Printf("[WS] Disconnect failed: %d and %d not in room %d", msg.Sender, msg.Target, msg.RoomID)
		return
	}

	wm.mtx.Lock()
	defer wm.mtx.Unlock()

	// Disconnect sender
	if conn, exists := wm.connections[msg.Sender]; exists {
		conn.Close()
		delete(wm.connections, msg.Sender)
		delete(wm.rooms[msg.RoomID], msg.Sender)
		log.Printf("[WS] Disconnected sender %d", msg.Sender)
	}

	// Disconnect target
	if conn, exists := wm.connections[msg.Target]; exists {
		conn.Close()
		delete(wm.connections, msg.Target)
		delete(wm.rooms[msg.RoomID], msg.Target)
		log.Printf("[WS] Disconnected target %d", msg.Target)
	}

	// Clean up the room if it’s empty
	if len(wm.rooms[msg.RoomID]) == 0 {
		delete(wm.rooms, msg.RoomID)
		log.Printf("[WS] Room %d deleted", msg.RoomID)
	}
}
