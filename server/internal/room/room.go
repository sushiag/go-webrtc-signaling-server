package room

import (
	"log"
	"sync"

	"github.com/gorilla/websocket" // Assuming you're using this package
)

// struct for rooms and client connections
type RoomManager struct {
	rooms map[string]*Room
	mtx   sync.Mutex
}

type Message struct {
	SenderID string
	Content  string
}

// Client represents a single client in a room
type Client struct {
	ID   string
	Conn *websocket.Conn
	Room *Room
}

// Room represents a chat room containing clients
type Room struct {
	ID      string
	Clients map[string]*Client
}

// NewRoomManager creates a new RoomManager
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
	}
}

// AddClient adds a client to a room
func (rm *RoomManager) AddClient(roomID string, client *Client) {
	rm.mtx.Lock()
	defer rm.mtx.Unlock()

	// If room does not exist, create it
	if _, exists := rm.rooms[roomID]; !exists {
		rm.rooms[roomID] = &Room{
			ID:      roomID,
			Clients: make(map[string]*Client),
		}
	}

	// Add the client to the room
	rm.rooms[roomID].Clients[client.ID] = client
}

// RemoveClient removes a client from a room
func (rm *RoomManager) RemoveClient(roomID string, client *Client) {
	rm.mtx.Lock()
	defer rm.mtx.Unlock()

	room, exists := rm.rooms[roomID]
	if !exists {
		log.Printf("Room %s does not exist.", roomID)
		return
	}

	delete(room.Clients, client.ID)

	// If the room has no more clients, remove it from the room manager
	if len(room.Clients) == 0 {
		delete(rm.rooms, roomID)
	}
}

// SendToRoom sends a message to all clients in a room
func (rm *RoomManager) SendToRoom(roomID, senderID string, message Message) {
	rm.mtx.Lock()
	defer rm.mtx.Unlock()

	room, exists := rm.rooms[roomID]
	if !exists {
		log.Printf("Room %s does not exist.", roomID)
		return
	}

	// Send the message to all clients in the room except the sender
	for _, client := range room.Clients {
		if client.ID == senderID {
			continue
		}

		if err := client.Conn.WriteJSON(message); err != nil {
			log.Printf("[Room] Failed to send message to client %s: %v", client.ID, err)
		}
	}
}
