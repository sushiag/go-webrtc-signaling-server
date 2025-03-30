package room

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type SignalingMessage struct {
	Type     string          `json:"type"` // "offer", "answer", "ice"
	Payload  json.RawMessage `json:"payload"`
	SenderID string          `json:"sender_id"`
	TargetID string          `json:"target_id,omitempty"`
}

type Client struct {
	ID   string
	Conn *websocket.Conn
}

type Room struct {
	ID      string
	Clients map[string]*Client
	Mutex   sync.RWMutex
}

type RoomManager struct {
	rooms map[string]*Room
	Mtx   sync.RWMutex
	log   *logrus.Entry
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
		log:   logrus.WithField("component", "room"),
	}
}

func (rm *RoomManager) CreateRoom(roomID string) {
	rm.Mtx.Lock()
	defer rm.Mtx.Unlock()

	if _, exists := rm.rooms[roomID]; !exists {
		rm.rooms[roomID] = &Room{
			ID:      roomID,
			Clients: make(map[string]*Client),
		}
		log.Println("[ROOM] Created:", roomID) // create a room if it doesn't exist
	}
}

func (rm *RoomManager) GetRoom(roomID string) *Room {
	rm.Mtx.RLock()
	defer rm.Mtx.RUnlock()

	return rm.rooms[roomID]

}

func (rm *RoomManager) AddClient(roomID string, client *Client) {
	rm.Mtx.Lock()
	room, exists := rm.rooms[roomID]
	if !exists {
		room = &Room{
			ID:      roomID,
			Clients: make(map[string]*Client),
		}
		rm.rooms[roomID] = room

	}
	rm.Mtx.Unlock()

	room.Mutex.Lock()
	room.Clients[client.ID] = client
	room.Mutex.Unlock()

	log.Printf("[ROOM] Client %s has joined room %s", client.ID, roomID) // adds a client to a room
}

func (rm *RoomManager) ListClients(roomID string) []string {
	room := rm.GetRoom(roomID)
	if room == nil {
		return []string{}
	}
	var clients []string
	for clientID := range room.Clients {
		clients = append(clients, clientID)
	}
	return clients
}

func (rm *RoomManager) RemoveClient(roomID, clientID string) {
	rm.Mtx.RLock()
	room, exists := rm.rooms[roomID]
	rm.Mtx.RUnlock()

	if !exists {
		log.Printf("[ROOM] Non-existent room: %s", roomID)
		return
	}

	room.Mutex.Lock()
	delete(room.Clients, clientID)
	empty := len(room.Clients) == 0
	room.Mutex.Unlock()

	if empty {
		rm.Mtx.Lock()
		delete(rm.rooms, roomID)
		rm.Mtx.Unlock()
		log.Println("[ROOM] Deleted empty room:", roomID)
	}

	log.Printf("[ROOM] Client %s has left room %s", clientID, roomID) // removes a client from a room
}

func (rm *RoomManager) BroadcastMessage(roomID, senderID string, message []byte) {
	rm.Mtx.RLock()
	room, exists := rm.rooms[roomID]
	rm.Mtx.RUnlock()

	if !exists {
		log.Println("[ROOM] Non-existent room:", roomID) // sends a message to all clients in a room except the sender
		return
	}

	room.Mutex.RLock()
	defer room.Mutex.RUnlock()

	for id, client := range room.Clients {
		if id != senderID {
			err := client.Conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("[ROOM] Failed to send message to %s: %v", id, err)
			}
		}
	}
}

func (r *Room) BroadcastToOthers(message []byte, sender *Client) {
	r.Mutex.RLock()
	defer r.Mutex.Unlock()

	for id, client := range r.Clients {
		if id != sender.ID {
			err := client.Conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("[ROOM] failed to send message to %s: %v", id, err)
			}
		}
	}
}
