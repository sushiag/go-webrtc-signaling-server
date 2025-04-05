package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// struct for websocket communication
type Message struct {
	Type    string `json:"type"`
	RoomID  string `json:"roomId"`
	Sender  string `json:"sender"`
	Content string `json:"content"`
}

// struct for websocket client
type Client struct {
	ID   string
	Conn *websocket.Conn
	Room *Room
}

// this manages WebSocket connections
type WebSocketManager struct {
	clients     map[string]*Client // API Key -> Client
	roomManager *RoomManager
	clientMtx   sync.RWMutex
	upgrader    websocket.Upgrader
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(roomManager *RoomManager) *WebSocketManager {
	return &WebSocketManager{
		clients:     make(map[string]*Client),
		roomManager: roomManager,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true }, // Customize as needed
		},
	}
}

// Authenticate checks the API key in the WebSocket request
func (wm *WebSocketManager) Authenticate(r *http.Request) bool {
	apiKey := r.Header.Get("X-Api-Key")
	// Add your API key validation logic here (e.g., check against a list or database)
	return apiKey != "" // For now, assume any non-empty key is valid
}

// Handler manages WebSocket connections, using API key for authentication
func (wm *WebSocketManager) Handler(w http.ResponseWriter, r *http.Request) {
	if !wm.Authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := wm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[WS] WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	// Get the API key from the request header
	apiKey := r.Header.Get("X-Api-Key")

	// Create a new client
	client := &Client{ID: apiKey, Conn: conn}

	// Store the client in the WebSocket manager
	wm.clientMtx.Lock()
	wm.clients[apiKey] = client
	wm.clientMtx.Unlock()

	log.Printf("[WS] Client %s connected", apiKey)

	defer func() {
		// Remove the client when it disconnects
		wm.clientMtx.Lock()
		delete(wm.clients, apiKey)
		wm.clientMtx.Unlock()
	}()

	// Send the API key to the client
	if err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type": "api_key", "api_key": "%s"}`, apiKey))); err != nil {
		log.Printf("[WS] Failed to send API key to client %s: %v", apiKey, err)
		return
	}

	// Read incoming WebSocket messages
	go wm.readMessages(client)
	wm.sendPings(client)
}

// readMessages handles signaling messages from clients
func (wm *WebSocketManager) readMessages(client *Client) {
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			log.Println("[ERROR] Read error:", err)
			return
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Println("[ERROR] Failed to parse WebSocket message:", err)
			continue
		}

		log.Printf("[WS] Received message from %s: %s", client.ID, msg.Type)

		// Forward the message to the correct room
		wm.roomManager.SendToRoom(msg.RoomID, client.ID, msg)
	}
}

// sendPings sends periodic pings to the client to keep the connection alive
func (wm *WebSocketManager) sendPings(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	pingFailures := 0
	for range ticker.C {
		if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			log.Println("[WS] Failed to send ping:", err)
			pingFailures++
			if pingFailures >= 3 {
				log.Printf("[WS] Client %s failed to respond to pings. Closing connection.", client.ID)
				client.Conn.Close()
				return
			}
		} else {
			pingFailures = 0 // Reset on successful ping
		}
	}
}
