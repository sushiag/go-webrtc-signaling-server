package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// message type for the readmessages
const (
	TypeCreateRoom      = "create-room"
	TypeJoin            = "join-room"
	TypeOffer           = "offer"
	TypeAnswer          = "answer"
	TypeICE             = "ice-candidate"
	TypeDisconnect      = "disconnect"
	TypeText            = "text"
	TypePeerJoined      = "room-joined"
	TypeRoomCreated     = "room-created"
	TypePeerList        = "peer-list"
	TypePeerReady       = "peer-ready"
	TypeStart           = "start"
	TypeStartP2P        = "start-session"
	TypePeerListRequest = "peer-list-request"
)

type Message struct {
	Type      string   `json:"type"`
	APIKey    string   `json:"apikey,omitempty"`
	Content   string   `json:"content,omitempty"`
	RoomID    uint64   `json:"roomid,omitempty"`
	Sender    uint64   `json:"from,omitempty"`
	Target    uint64   `json:"to,omitempty"`
	SDP       string   `json:"sdp,omitempty"`
	Candidate string   `json:"candidate,omitempty"`
	UserID    uint64   `json:"userid,omitempty"`
	Users     []uint64 `json:"users,omitempty"`
}

// struct for a group of connected users
type Room struct {
	ID        uint64
	Users     map[uint64]*websocket.Conn
	ReadyMap  map[uint64]bool
	JoinOrder []uint64
	HostID    uint64
}

// this handles connection and room management
type WebSocketManager struct {
	Connections     map[uint64]*websocket.Conn
	Rooms           map[uint64]*Room
	validApiKeys    map[string]bool
	apiKeyToUserID  map[string]uint64
	nextUserID      uint64
	nextRoomID      uint64
	candidateBuffer map[uint64][]Message
	messageChan     chan Message
	disconnectChan  chan uint64
}

// this initializes a new manager
func NewWebSocketManager() *WebSocketManager {
	wsm := &WebSocketManager{
		Connections:     make(map[uint64]*websocket.Conn),
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

func (wsm *WebSocketManager) SafeWriteJSON(conn *websocket.Conn, v interface{}) error {
	return conn.WriteJSON(v)
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

	conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
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
	wsm.Connections[userID] = conn

	// Flush buffered candidates if any
	wsm.flushBufferedMessages(userID)

	log.Printf("[WS] User %d connected", userID)

	go wsm.readMessages(userID, conn)
	go wsm.sendPings(userID, conn)
}
