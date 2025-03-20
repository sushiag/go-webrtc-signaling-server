package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

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
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			log.Println("[WS] Read error:", err)
			break
		}

		var msg Message
		json.Unmarshal(message, &msg)

		switch msg.Type {
		case "join":
			wm.roomManager.AddClient(msg.RoomID, client)
		case "message":
			wm.roomManager.BroadcastMessage(msg.RoomID, client.ID, []byte(msg.Content))
		}
	}
}
