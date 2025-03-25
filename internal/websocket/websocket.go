package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
	"github.com/sushiag/go-webrtc-signaling-server/internal/room"
)

type Message struct {
	Type    string `json:"type"`
	RoomID  string `json:"room_id,omitempty"`
	Sender  string `json:"sender,omitempty"`
	Content string `json:"content,omitempty"`
}

type WebSocketManager struct { // handles the WebSocket connections
	upgrader       websocket.Upgrader
	roomManager    *room.RoomManager
	clientMtx      sync.Mutex
	clients        map[string]*room.Client
	dataChannels   map[string]*webrtc.DataChannel // stores active data channels
	DataChannelMtx sync.RWMutex
	DataChannels   map[string]*webrtc.DataChannel
}

func NewWebSocketManager(allowedOrigin string) *WebSocketManager {
	return &WebSocketManager{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return allowedOrigin == "*" || origin == allowedOrigin
			},
		},
		roomManager:  room.NewRoomManager(),
		clients:      make(map[string]*room.Client),
		dataChannels: make(map[string]*webrtc.DataChannel),
	}
}

func generateUniqueClientID() string {
	return uuid.New().String()
}

func (wm *WebSocketManager) Handler(w http.ResponseWriter, r *http.Request, authenticate func(*http.Request) bool) {
	if !authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := wm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[WS] WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	clientID := generateUniqueClientID()
	client := &room.Client{ID: clientID, Conn: conn}

	wm.clientMtx.Lock()
	wm.clients[clientID] = client
	wm.clientMtx.Unlock()

	log.Println("[WS] Client connected:", clientID)

	defer wm.disconnectClient(client)

	// Send the assigned clientID to the client
	client.Conn.WriteMessage(websocket.TextMessage, []byte(`{"type": "client_id", "client_id": "`+clientID+`"}`))

	go wm.sendPings(client)
	wm.readMessages(client)
}

func (wm *WebSocketManager) SendToRoom(roomID, senderID string, message Message) {
	log.Printf("[WS] Sending message to room %s from %s\n", roomID, senderID)

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("[WS] Failed to serialize message: %v", err)
		return
	}
	wm.roomManager.BroadcastMessage(roomID, senderID, data)
}

func (wm *WebSocketManager) readMessages(client *room.Client) { //listens for messages from the clients
	log.Printf("[WS] Client %s started reading messages.\n", client.ID)
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			log.Printf("[WS] Client %s encountered an error: %v\n", client.ID, err)
			break
		}

		var msg Message // skips if no error
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[WS] Client %s sent an invalid message format: %s\n", client.ID, string(message))
			continue
		}

		log.Printf("[WS] Client %s sent a '%s': %s\n", client.ID, msg.Type, msg.Content)

		switch msg.Type {
		case "join":
			log.Printf("[WS] Client %s is joining the room %s.\n", client.ID, msg.RoomID)
			wm.roomManager.CreateRoom(msg.RoomID) // pre-create room
			wm.roomManager.AddClient(msg.RoomID, client)
		case "offer", "answer", "ice-candidate":
			log.Printf("[WS] Client %s forwarding a '%s' message to room %s.\n", client.ID, msg.Type, msg.RoomID)
			wm.roomManager.BroadcastMessage(msg.RoomID, client.ID, []byte(msg.Content))
		case "signal":
			log.Printf("[WS] Client %s is broadcasting signal message to room %s.\n", client.ID, msg.RoomID)
			room := wm.roomManager.GetRoom(msg.RoomID)
			if room == nil {
				log.Printf("[WS] Room %s not found.", msg.RoomID)
				continue
			}
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("[WS] Failed to serialize signal message from %s: %v", client.ID, err)
				continue
			}
			room.BroadcastToOthers(data, client)
		case "data-message":
			log.Printf("[WS] Client %s sent a data message: %s\n", client.ID, msg.Content)
			wm.sendDataToDataChannel(client.ID, []byte(msg.Content))
		default:
			log.Printf("[WS] Client %s sent an unknown message type: %s\n", client.ID, msg.Type)
		}
	}
}

func (wm *WebSocketManager) disconnectClient(client *room.Client) {
	wm.clientMtx.Lock()
	defer wm.clientMtx.Unlock()

	delete(wm.clients, client.ID)
	log.Println("[WS] Client disconnected:", client.ID)
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

func (wm *WebSocketManager) SetupDataChannel(peerConnection *webrtc.PeerConnection, clientID string) {
	dataChannel, err := peerConnection.CreateDataChannel("default", nil)
	if err != nil {
		log.Println("[WS] Error creating DataChannel:", err)
		return
	}

	dataChannel.OnOpen(func() {
		log.Println("[WS] DataChannel opened for client:", clientID)
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("[WS] DataChannel received message from %s: %s\n", clientID, string(msg.Data))
	})

	wm.dataChannels[clientID] = dataChannel
}

func (wm *WebSocketManager) sendDataToDataChannel(clientID string, message []byte) {
	dataChannel, exists := wm.dataChannels[clientID]
	if !exists || dataChannel == nil {
		log.Printf("[WS] No DataChannel found for client %s\n", clientID)
		return
	}

	err := dataChannel.Send(message)
	if err != nil {
		log.Printf("[WS] Error sending message to DataChannel for client %s: %v\n", clientID, err)
	}
}
