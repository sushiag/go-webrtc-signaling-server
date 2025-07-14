package server

import (
	"github.com/gorilla/websocket"

	smsg "signaling-msgs"
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
type WebSocketManager struct {
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
