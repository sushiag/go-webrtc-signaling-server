package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sushiag/go-webrtc-signaling-server/internal/room"
)

type Message struct {
	Type    string `json:"type"`
	RoomID  string `json:"room_id,omitempty"`
	Sender  string `json:"sender,omitempty"`
	Content string `json:"content,omitempty"`
}

type WebSocketManager struct { // handles the WebSocket connections
	upgrader    websocket.Upgrader
	roomManager *room.RoomManager
	clientMtx   sync.Mutex
	clients     map[string]*room.Client
}

func NewWebSocketManager(allowedOrigin string) *WebSocketManager {
	return &WebSocketManager{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return allowedOrigin == "*" || origin == allowedOrigin
			},
		},
		roomManager: room.NewRoomManager(),
		clients:     make(map[string]*room.Client),
	}
}

func (wm *WebSocketManager) Handler(w http.ResponseWriter, r *http.Request, authenticate func(*http.Request) bool) {
	if !authenticate(r) { // manages WebSocket connections
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := wm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[WS] WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	clientID := r.RemoteAddr
	client := &room.Client{ID: clientID, Conn: conn}

	wm.clientMtx.Lock()
	wm.clients[clientID] = client
	wm.clientMtx.Unlock()

	log.Println("[WS] Client connected:", clientID)

	defer wm.disconnectClient(client)

	go wm.sendPings(client)
	wm.readMessages(client)
}

func (wm *WebSocketManager) readMessages(client *room.Client) { //listens for messages from the clients
	log.Printf("[WS] client %s has start readin messages. \n ", client.ID)
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			log.Printf("[WS] Client %s encountered an error: %v\n", client.ID, err)
			break
		}

		var msg Message // skips if no error
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[WS] Client %s send an invalid message format: %s\n", client.ID, string(message))
			continue
		}

		log.Printf("[WS] Client %s sent a '%s': %s\n ", client.ID, msg.Type, msg.Content)

		switch msg.Type {
		case "join":
			log.Printf("[WS] Client %s has joining the room %s. \n ", client.ID, msg.RoomID)
			wm.roomManager.AddClient(msg.RoomID, client)
		case "offer", "asnwer", "ice-candidate":
			log.Printf("[WS] Client %s is forwarding a '%s' message to room %s. \n", client.ID, msg.Type, msg.RoomID)
			wm.roomManager.BroadcastMessage(msg.RoomID, client.ID, []byte(msg.Content))
		default:
			log.Printf("[WS] Client sent %s unknown message type: %s\n", client.ID, msg.Type)
		}
	}
}

func (wm *WebSocketManager) disconnectClient(client *room.Client) {
	wm.clientMtx.Lock()
	defer wm.clientMtx.Unlock()

	delete(wm.clients, client.ID)
	log.Println("[WS] cLient has disconnected:", client.ID)
}

func (wm *WebSocketManager) sendPings(client *room.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := client.Conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
			log.Println("[WS] Ping error:", err)
			return
		}
	}
}
