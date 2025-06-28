package server

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

type MessageType int

// message type for the readmessages
const (
	TypeCreateRoom MessageType = iota
	TypeJoin
	TypeOffer
	TypeAnswer
	TypeICE
	TypeDisconnect
	TypeText
	TypePeerJoined
	TypeRoomCreated
	TypePeerList
	TypePeerReady
	TypeStart
	TypeStartP2P
	TypePeerListRequest
	TypeHostChanged
	TypeSendMessage
	TypePeerLeft
)

type Message struct {
	Type      MessageType `json:"type"`
	APIKey    string      `json:"apikey,omitempty"`
	Content   string      `json:"content,omitempty"`
	RoomID    uint64      `json:"roomid,omitempty"`
	Sender    uint64      `json:"from,omitempty"`
	Target    uint64      `json:"to,omitempty"`
	SDP       string      `json:"sdp,omitempty"`
	Candidate string      `json:"candidate,omitempty"`
	UserID    uint64      `json:"userid,omitempty"`
	Users     []uint64    `json:"users,omitempty"`
}

// struct for a group of connected users
type Room struct {
	ID        uint64
	Users     map[uint64]*Connection
	ReadyMap  map[uint64]bool
	JoinOrder []uint64
	HostID    uint64
}

func (t MessageType) String() string {
	switch t {
	case TypeCreateRoom:
		return "create-room"
	case TypeJoin:
		return "join-room"
	case TypeOffer:
		return "offer"
	case TypeAnswer:
		return "answer"
	case TypeICE:
		return "ice-candidate"
	case TypeDisconnect:
		return "disconnect"
	case TypeText:
		return "text"
	case TypePeerJoined:
		return "room-joined"
	case TypeRoomCreated:
		return "room-created"
	case TypePeerList:
		return "peer-list"
	case TypePeerReady:
		return "peer-ready"
	case TypeStart:
		return "start"
	case TypeStartP2P:
		return "start-session"
	case TypePeerListRequest:
		return "peer-list-request"
	case TypeHostChanged:
		return "host-changed"
	case TypeSendMessage:
		return "send-message"
	case TypePeerLeft:
		return "peer-left"
	default:
		return "unknown"
	}
}

func (t MessageType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *MessageType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case "create-room":
		*t = TypeCreateRoom
	case "join-room":
		*t = TypeJoin
	case "offer":
		*t = TypeOffer
	case "answer":
		*t = TypeAnswer
	case "ice-candidate":
		*t = TypeICE
	case "disconnect":
		*t = TypeDisconnect
	case "text":
		*t = TypeText
	case "room-joined":
		*t = TypePeerJoined
	case "room-created":
		*t = TypeRoomCreated
	case "peer-list":
		*t = TypePeerList
	case "peer-ready":
		*t = TypePeerReady
	case "start":
		*t = TypeStart
	case "start-session":
		*t = TypeStartP2P
	case "peer-list-request":
		*t = TypePeerListRequest
	case "host-changed":
		*t = TypeHostChanged
	case "send-message":
		*t = TypeSendMessage
	case "peer-left":
		*t = TypePeerLeft
	default:
		*t = -1
	}
	return nil
}

// this handles connection and room management
type WebSocketManager struct {
	Connections     map[uint64]*Connection
	Rooms           map[uint64]*Room
	validApiKeys    map[string]bool
	apiKeyToUserID  map[string]uint64
	nextUserID      uint64
	nextRoomID      uint64
	candidateBuffer map[uint64][]Message
	messageChan     chan Message
	disconnectChan  chan uint64
}

// this handles connection that starts its own goroutine
type Connection struct {
	UserID       uint64
	Conn         *websocket.Conn
	Incoming     chan Message
	Outgoing     chan Message
	Disconnected chan<- uint64
}

// this initializes a new manager
func NewWebSocketManager() *WebSocketManager {
	wsm := &WebSocketManager{
		Connections:     make(map[uint64]*Connection),
		Rooms:           make(map[uint64]*Room),
		apiKeyToUserID:  make(map[string]uint64),
		candidateBuffer: make(map[uint64][]Message),
		nextUserID:      1,
		nextRoomID:      1,
		messageChan:     make(chan Message),
		disconnectChan:  make(chan uint64),
	}
	go wsm.run()
	return wsm
}

func (wsm *WebSocketManager) run() {
	for {
		select {
		case msg := <-wsm.messageChan:
			wsm.handleMessage(msg)
		case userID := <-wsm.disconnectChan:
			wsm.disconnectUser(userID)
		}
	}
}
