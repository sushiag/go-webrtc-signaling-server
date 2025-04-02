package websocket

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
	"log"
	"net/http"
	"server/room"
	"sync"
	"time"
)

type Client struct {
	ID     string
	Conn   *websocket.Conn
	RoomID string
}

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
	clients        map[string]*websocket.Conn // client_id -> Conn
	DataChannelMtx sync.RWMutex
	DataChannels   map[string]*webrtc.DataChannel
}

func NewWebSocketManager() *WebSocketManager {

	return &WebSocketManager{
		upgrader:     websocket.Upgrader{},
		roomManager:  room.NewRoomManager(),
		clients:      make(map[string]*websocket.Conn),
		DataChannels: make(map[string]*webrtc.DataChannel),
	}
}

func generateUniqueClientID() string {
	return uuid.New().String()
}

func (wm *WebSocketManager) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return wm.upgrader.Upgrade(w, r, nil)
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

// Listens for messages from the clients
func (wm *WebSocketManager) ReadMessages(clientId string) {
	conn := wm.clients[clientId]

	log.Printf("[WS] Reading messages from client %s.\n", clientId)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[WS] Client %s encountered an error: %v\n", clientId, err)
			break
		}

		var msg Message // skips if no error
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[WS] Client %s sent an invalid message format: %s\n", clientId, string(message))
			continue
		}

		log.Printf("[WS] Client %s sent a '%s': %s\n", clientId, msg.Type, msg.Content)

		// switch msg.Type {
		// case "join":
		// 	log.Printf("[WS] Client %s is joining room %s.\n", clientId, msg.RoomID)
		//
		// 	if wm.roomManager.GetRoom(msg.RoomID) == nil {
		// 		wm.roomManager.CreateRoom(msg.RoomID)
		// 	}
		// 	wm.roomManager.AddClient(msg.RoomID, client)
		// case "offer", "answer", "ice-candidate":
		// 	log.Printf("[WS] Client %s forwarding a '%s' message to room %s.\n", client.ID, msg.Type, msg.RoomID)
		// 	wm.roomManager.BroadcastMessage(msg.RoomID, client.ID, []byte(msg.Content))
		// case "signal":
		// 	log.Printf("[WS] Client %s is broadcasting signal message to room %s.\n", client.ID, msg.RoomID)
		// 	room := wm.roomManager.GetRoom(msg.RoomID)
		// 	if room == nil {
		// 		log.Printf("[WS] Room %s not found.", msg.RoomID)
		// 		continue
		// 	}
		// 	data, err := json.Marshal(msg)
		// 	if err != nil {
		// 		log.Printf("[WS] Failed to serialize signal message from %s: %v", client.ID, err)
		// 		continue
		// 	}
		// 	room.BroadcastToOthers(data, client)
		// case "data-message":
		// 	log.Printf("[WS] Client %s sent a data message: %s\n", client.ID, msg.Content)
		// 	wm.sendDataToDataChannel(client.ID, []byte(msg.Content))
		// default:
		// 	log.Printf("[WS] Client %s sent an unknown message type: %s\n", client.ID, msg.Type)
		// }
	}
}

func (wm *WebSocketManager) AddClient(clientId string, conn *websocket.Conn) {
	wm.clients[clientId] = conn
}

func (wm *WebSocketManager) DisconnectClient(clientId string) {
	// Delete the reference to the WS connection in the connection map
	delete(wm.clients, clientId)
	log.Println("[WS] Client disconnected:", clientId)
}

func (wm *WebSocketManager) SendPings(clientId string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		conn := wm.clients[clientId]
		if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
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

	wm.DataChannels[clientID] = dataChannel
}

func (wm *WebSocketManager) sendDataToDataChannel(clientID string, message []byte) {
	dataChannel, exists := wm.DataChannels[clientID]
	if !exists || dataChannel == nil {
		log.Printf("[WS] No DataChannel found for client %s\n", clientID)
		return
	}

	err := dataChannel.Send(message)
	if err != nil {
		log.Printf("[WS] Error sending message to DataChannel for client %s: %v\n", clientID, err)
	}
}
