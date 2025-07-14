package server

import (
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"

	smsg "signaling-msgs"
)

type MessageType int

// message type for the readmessages
const (
	MessageTypeOffer MessageType = iota
	MessageTypeAnswer
	MessageTypeICECandidate
	MessageTypePeerJoined
	MessageTypePeerLeft
	MessageTypeDisconnect
	MessageTypeSendMessage
	MessageTypePeerList
	MessageTypeHostChanged
	MessageTypeStartSession
	MessageTypeRoomCreated
	MessageTypeCreateRoom
	MessageTypeJoinRoom
	MessageTypePeerListReq
	MessageTypeRoomJoined
	MessageTypeSetUserID
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
	case MessageTypeCreateRoom:
		return "create-room"
	case MessageTypeJoinRoom:
		return "join-room"
	case MessageTypeOffer:
		return "offer"
	case MessageTypeAnswer:
		return "answer"
	case MessageTypeICECandidate:
		return "ice-candidate"
	case MessageTypeDisconnect:
		return "disconnect"
	case MessageTypePeerJoined:
		return "peer-joined"
	case MessageTypeRoomCreated:
		return "room-created"
	case MessageTypePeerList:
		return "peer-list"
	case MessageTypeStartSession:
		return "start-session"
	case MessageTypePeerListReq:
		return "peer-list-request"
	case MessageTypeHostChanged:
		return "host-changed"
	case MessageTypeSendMessage:
		return "send-message"
	case MessageTypePeerLeft:
		return "peer-left"
	case MessageTypeSetUserID:
		return "set-user-id"
	case MessageTypeRoomJoined:
		return "room-joined"
	default:
		return fmt.Sprintf("unknown (%d)", t)
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
		*t = MessageTypeCreateRoom
	case "join-room":
		*t = MessageTypeJoinRoom
	case "offer":
		*t = MessageTypeOffer
	case "answer":
		*t = MessageTypeAnswer
	case "ice-candidate":
		*t = MessageTypeICECandidate
	case "disconnect":
		*t = MessageTypeDisconnect
	case "room-joined":
		*t = MessageTypePeerJoined
	case "room-created":
		*t = MessageTypeRoomCreated
	case "peer-list":
		*t = MessageTypePeerList
	case "start-session":
		*t = MessageTypeStartSession
	case "peer-list-request":
		*t = MessageTypePeerListReq
	case "host-changed":
		*t = MessageTypeHostChanged
	case "send-message":
		*t = MessageTypeSendMessage
	case "peer-left":
		*t = MessageTypePeerLeft
	default:
		*t = -1
	}
	return nil
}

// this handles connection and room management
type WebSocketManager struct {
	Connections     map[uint64]*Connection
	Rooms           map[uint64]*Room
	nextUserID      uint64
	nextRoomID      uint64
	candidateBuffer map[uint64][]Message
	messageChan     chan *smsg.MessageRawJSONPayload
	disconnectChan  chan uint64
	newConnChan     chan *Connection
}

// this handles connection that starts its own goroutine
type Connection struct {
	UserID       uint64
	Conn         *websocket.Conn
	Incoming     chan Message
	Outgoing     chan smsg.MessageAnyPayload
	Disconnected chan<- uint64
}

// this initializes a new manager
func NewWebSocketManager() *WebSocketManager {
	wsm := &WebSocketManager{
		Connections:     make(map[uint64]*Connection),
		Rooms:           make(map[uint64]*Room),
		candidateBuffer: make(map[uint64][]Message),
		nextUserID:      1,
		nextRoomID:      1,
		messageChan:     make(chan *smsg.MessageRawJSONPayload),
		disconnectChan:  make(chan uint64),
		newConnChan:     make(chan *Connection),
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
