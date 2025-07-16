package server

import (
	smsg "signaling-msgs"

	"github.com/gorilla/websocket"
	"github.com/sushiag/go-webrtc-signaling-server/server/lib/db"
)

// struct for a group of connected users
type Room struct {
	ID        uint64
	Users     map[uint64]*Connection
	ReadyMap  map[uint64]bool
	JoinOrder []uint64
	HostID    uint64
}

// this handles connection and room management
//
// TODO: we should refactor our data structure for the connections+rooms.
//
// Connections should be map[uint64]*Connection where Connection has the following:
//
//	type Connection struct {
//		clientID		uint64
//		roomID			room
//		isInRoom		bool
//		isRoomOwner	bool
//		... etc.
//	}
//
// then the Rooms should be map[roomID]*Room where Room is:
//
//	type Room struct {
//		clients	[]uint64
//		owner		uint64
//		... etc.
//	}
//
// The change in the connection struct will make it easier to query if two users are in
// the same room without iterating through the room clients struct
type WebSocketManager struct {
	//validApiKeys    map[string]bool
	//apiKeyToUserID  map[string]uint64
	//candidateBuffer map[uint64][]Message
	Queries *db.Queries

	Connections    map[uint64]*Connection
	Rooms          map[uint64]*Room
	nextUserID     uint64
	nextRoomID     uint64
	messageChan    chan *smsg.MessageRawJSONPayload
	disconnectChan chan uint64
	newConnChan    chan *Connection
}

// this handles connection that starts its own goroutine
type Connection struct {
	UserID       uint64
	Conn         *websocket.Conn
	Outgoing     chan smsg.MessageAnyPayload
	Disconnected chan<- uint64
}

// this initializes a new manager
func NewWebSocketManager() *WebSocketManager {
	wsm := &WebSocketManager{
		Connections:    make(map[uint64]*Connection),
		Rooms:          make(map[uint64]*Room),
		nextUserID:     1,
		nextRoomID:     1,
		messageChan:    make(chan *smsg.MessageRawJSONPayload),
		disconnectChan: make(chan uint64),
		newConnChan:    make(chan *Connection),
	}
	go wsm.run()
	return wsm
}

func (wsm *WebSocketManager) run() {
	for {
		select {
		case msg := <-wsm.messageChan:
			wsm.handleMessage(msg)
		case newConn := <-wsm.newConnChan:
			wsm.handleNewConnection(newConn)
		case userID := <-wsm.disconnectChan:
			wsm.disconnectUser(userID)
		}
	}
}
