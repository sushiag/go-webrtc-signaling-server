package server

import (
	"encoding/json"
	"log"
	"net/http"

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

func NewConnection(userID uint64, conn *websocket.Conn, msgOut chan<- Message, disconnectOut chan<- uint64) *Connection {
	c := &Connection{
		UserID:       userID,
		Conn:         conn,
		Incoming:     make(chan Message),
		Outgoing:     make(chan Message),
		Disconnected: disconnectOut,
	}

	go c.readLoop(msgOut)
	go c.writeLoop()
	return c
}

func (c *Connection) readLoop(msgOut chan<- Message) {
	defer func() {
		c.Disconnected <- c.UserID
		c.Conn.Close()
		log.Printf("[WS] User %d disconnected (readLoop)", c.UserID)
	}()

	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error for user %d: %v", c.UserID, err)
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("[WS] Invalid message from %d: %v", c.UserID, err)
			continue
		}

		msg.Sender = c.UserID
		msgOut <- msg
	}
}

func (c *Connection) writeLoop() {
	for msg := range c.Outgoing {
		if err := c.Conn.WriteJSON(msg); err != nil {
			log.Printf("[WS] Write error to %d: %v", c.UserID, err)
			c.Disconnected <- c.UserID
			return
		}
	}
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

func (wsm *WebSocketManager) SafeWriteJSON(c *Connection, v Message) error {
	c.Outgoing <- v
	return nil
}

func (wsm *WebSocketManager) SetValidApiKeys(keys map[string]bool) {
	wsm.validApiKeys = keys
}

func (wsm *WebSocketManager) Authenticate(r *http.Request) bool {
	return wsm.validApiKeys[r.Header.Get("X-Api-Key")]
}

// this handles the initial API key authentication via HTTP
func (wsm *WebSocketManager) AuthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		ApiKey string `json:"apikey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if !wsm.validApiKeys[payload.ApiKey] {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := wsm.nextUserID
	wsm.apiKeyToUserID[payload.ApiKey] = userID
	wsm.nextUserID++

	resp := struct {
		UserID uint64 `json:"userid"`
	}{
		UserID: userID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (wsm *WebSocketManager) Handler(w http.ResponseWriter, r *http.Request) {
	if !wsm.Authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	apiKey := r.Header.Get("X-Api-Key")
	userID := wsm.apiKeyToUserID[apiKey]

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}

	if _, ok := wsm.Connections[userID]; ok {
		log.Printf("[WS] Duplicate connection attempt for user %d. Denying new connection.", userID)
		closeMsg := websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Duplicate connection detected")
		conn.WriteMessage(websocket.CloseMessage, closeMsg)
		conn.Close()
		return
	}
	connection := NewConnection(userID, conn, wsm.messageChan, wsm.disconnectChan)
	wsm.Connections[userID] = connection

	// Flush buffered candidates if any
	wsm.flushBufferedMessages(userID)

	log.Printf("[WS] User %d connected", userID)

	c := NewConnection(userID, conn, wsm.messageChan, wsm.disconnectChan)
	wsm.Connections[userID] = c
	go wsm.sendPings(userID, conn)
}
