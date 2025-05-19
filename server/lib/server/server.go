package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	TypeCreateRoom  = "create-room"
	TypeJoin        = "join-room"
	TypeOffer       = "offer"
	TypeAnswer      = "answer"
	TypeICE         = "ice-candidate"
	TypeDisconnect  = "disconnect"
	TypeText        = "text"
	TypePeerJoined  = "peer-joined"
	TypeRoomCreated = "room-created"
	TypePeerList    = "peer-list"
	TypePeerReady   = "peer-ready"
	TypeStart       = "start"
	CmdText         = "text"
	CmdStartSession = "start-session"
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
type Room struct {
	ID        uint64
	Users     map[uint64]*websocket.Conn
	ReadyMap  map[uint64]bool
	JoinOrder []uint64
	HostID    uint64
}

type managerCommand struct {
	cmd      string
	message  Message
	conn     *websocket.Conn
	response chan any
	respChan chan managerCommand
	roomID   uint64
	userID   uint64
	err      error
}
type WebSocketManager struct {
	commandChan chan managerCommand
	upgrader    websocket.Upgrader
	//nextUserID        uint64
	//nextRoomID        uint64
}

func NewWebSocketManager() *WebSocketManager {
	m := &WebSocketManager{
		commandChan: make(chan managerCommand),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
	go m.managerLoop()
	return m
}

func (wm *WebSocketManager) SetValidApiKeys(keys map[string]bool) {
	for k := range keys {
		resp := make(chan any)
		wm.commandChan <- managerCommand{
			cmd:      "register-api-key",
			message:  Message{APIKey: k},
			response: resp,
		}
		<-resp
	}
}
func (wm *WebSocketManager) managerLoop() {
	connections := make(map[uint64]*websocket.Conn)
	rooms := make(map[uint64]*Room)
	apiKeyToUserID := make(map[string]uint64)
	candidateBuffer := make(map[uint64][]Message)
	nextUserID := uint64(1)
	nextRoomID := uint64(1)
	validApiKeys := make(map[string]bool)

	log.Printf("[WS] Manager loop started")

	for cmd := range wm.commandChan {
		log.Printf("[WS] Received command: %s from user %d", cmd.cmd, cmd.userID)

		switch cmd.cmd {
		case "clear-api-keys":
			validApiKeys = make(map[string]bool)
			cmd.response <- struct{}{}

		case "register-api-key":
			key := cmd.message.APIKey
			if _, exists := validApiKeys[key]; !exists {
				validApiKeys[key] = true
				log.Printf("[WS] Registered API key: %s", key)
			}
			if cmd.response != nil {
				cmd.response <- true
			}

		case "auth-assign-userid":
			apiKey := cmd.message.APIKey
			if !validApiKeys[apiKey] {
				log.Printf("[WS] Unauthorized API key: %s", apiKey)
				cmd.response <- uint64(0)
				break
			}
			userID, ok := apiKeyToUserID[apiKey]
			if !ok {
				userID = nextUserID
				nextUserID++
				apiKeyToUserID[apiKey] = userID
				log.Printf("[WS] Assigned new userID %d to API key %s", userID, apiKey)
			} else {
				log.Printf("[WS] Existing userID %d found for API key %s", userID, apiKey)
			}
			cmd.response <- userID

		case "register-connection":
			userID := cmd.message.UserID
			if _, exists := connections[userID]; exists {
				log.Printf("[WS] Connection already exists for user %d", userID)
				cmd.response <- true
			} else {
				connections[userID] = cmd.conn
				log.Printf("[WS] Registered new connection for user %d", userID)
				cmd.response <- false
			}

		case "unregister-connection":
			userID := cmd.message.UserID
			delete(connections, userID)
			log.Printf("[WS] Unregistered connection for user %d", userID)

		case "add_user_to_room":
			room, exists := rooms[cmd.roomID]
			if !exists {
				// Create new room
				room = &Room{
					ID:        cmd.roomID,
					HostID:    cmd.userID,
					JoinOrder: []uint64{cmd.userID},
					Users:     make(map[uint64]*websocket.Conn),
					ReadyMap:  make(map[uint64]bool),
				}
				rooms[cmd.roomID] = room
				log.Printf("[WS] Created new room %d with host %d", cmd.roomID, cmd.userID)
			}

			conn, ok := connections[cmd.userID]
			if !ok {
				log.Printf("[WS] Connection not found for user %d when adding to room %d", cmd.userID, cmd.roomID)
				break
			}

			room.Users[cmd.userID] = conn
			room.JoinOrder = append(room.JoinOrder, cmd.userID)
			log.Printf("[WS] Added user %d to room %d", cmd.userID, cmd.roomID)

			// Notify other users
			for uid, peerConn := range room.Users {
				if uid != cmd.userID {
					err := peerConn.WriteJSON(Message{
						Type:   TypePeerJoined,
						RoomID: cmd.roomID,
						Sender: cmd.userID,
					})
					if err != nil {
						log.Printf("[WS] Failed to notify user %d of new peer %d: %v", uid, cmd.userID, err)
					} else {
						log.Printf("[WS] Notified user %d of new peer %d", uid, cmd.userID)
					}
				}
			}

		case "create_room":
			roomID := nextRoomID
			nextRoomID++

			room := &Room{
				ID:        roomID,
				HostID:    cmd.userID,
				JoinOrder: []uint64{cmd.userID},
				Users:     make(map[uint64]*websocket.Conn),
				ReadyMap:  make(map[uint64]bool),
			}
			if conn, ok := connections[cmd.userID]; ok {
				room.Users[cmd.userID] = conn
			} else {
				log.Printf("[WS] Connection not found for user %d when creating room", cmd.userID)
			}
			rooms[roomID] = room
			log.Printf("[WS] Created room %d with host %d", roomID, cmd.userID)

			resp := Message{
				Type:   TypeRoomCreated,
				RoomID: roomID,
				Sender: cmd.userID,
			}
			if err := cmd.conn.WriteJSON(resp); err != nil {
				log.Printf("[WS] Failed to send room-created to user %d: %v", cmd.userID, err)
				if cmd.respChan != nil {
					cmd.respChan <- managerCommand{err: err}
				}
			} else {
				log.Printf("[WS] Sent room-created to user %d: %+v", cmd.userID, resp)
				if cmd.respChan != nil {
					cmd.respChan <- managerCommand{}
				}
			}

		case "join_room":
			room, exists := rooms[cmd.roomID]
			if !exists {
				log.Printf("[WS] Room %d not found for user %d", cmd.roomID, cmd.userID)
				if cmd.conn != nil {
					closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Room does not exist")
					_ = cmd.conn.WriteMessage(websocket.CloseMessage, closeMsg)
				}
				continue
			}

			room.Users[cmd.userID] = cmd.conn
			room.JoinOrder = append(room.JoinOrder, cmd.userID)
			log.Printf("[WS] User %d joined room %d", cmd.userID, cmd.roomID)

			if room.HostID == 0 {
				room.HostID = cmd.userID
				log.Printf("[WS] Set user %d as host of room %d", cmd.userID, cmd.roomID)
			}

			for uid, peerConn := range room.Users {
				if uid != cmd.userID {
					err := peerConn.WriteJSON(Message{
						Type:   TypePeerJoined,
						RoomID: room.ID,
						Sender: cmd.userID,
					})
					if err != nil {
						log.Printf("[WS] Failed to notify user %d of new peer %d: %v", uid, cmd.userID, err)
					} else {
						log.Printf("[WS] Notified user %d of new peer %d", uid, cmd.userID)
					}
				}
			}

			err := cmd.conn.WriteJSON(Message{
				Type:   TypeCreateRoom,
				RoomID: room.ID,
			})
			if err != nil {
				log.Printf("[WS] Failed to send room info to user %d: %v", cmd.userID, err)
			} else {
				log.Printf("[WS] Sent room info to user %d", cmd.userID)
			}

		case "forward_or_buffer":
			targetConn, exists := connections[cmd.message.Target]
			inSameRoom := wm.areInSameRoom(cmd.message.RoomID, []uint64{cmd.message.Sender, cmd.message.Target})
			log.Printf("[WS] Forward or buffer message from %d to %d (inSameRoom=%v, targetConn exists=%v)", cmd.message.Sender, cmd.message.Target, inSameRoom, exists)

			if !exists || !inSameRoom {
				candidateBuffer[cmd.message.Target] = append(candidateBuffer[cmd.message.Target], cmd.message)
				log.Printf("[WS] Buffered message for user %d (total buffered: %d)", cmd.message.Target, len(candidateBuffer[cmd.message.Target]))
				break
			}

			if err := targetConn.WriteJSON(cmd.message); err != nil {
				log.Printf("[WS] Failed to forward message to %d: %v", cmd.message.Target, err)
			} else {
				log.Printf("[WS] Forwarded message to user %d", cmd.message.Target)
			}

		case "flush_buffered":
			buffered := candidateBuffer[cmd.userID]
			if len(buffered) == 0 {
				log.Printf("[WS] No buffered messages to flush for user %d", cmd.userID)
				break
			}

			conn, ok := connections[cmd.userID]
			if !ok {
				log.Printf("[WS] No connection for user %d when flushing buffer", cmd.userID)
				break
			}

			log.Printf("[WS] Flushing %d buffered messages for user %d", len(buffered), cmd.userID)
			var remaining []Message
			for _, msg := range buffered {
				if err := conn.WriteJSON(msg); err != nil {
					log.Printf("[WS] Failed to send buffered message to user %d: %v", cmd.userID, err)
					remaining = append(remaining, msg)
				}
			}

			if len(remaining) > 0 {
				candidateBuffer[cmd.userID] = remaining
				log.Printf("[WS] %d buffered messages remain for user %d after flush", len(remaining), cmd.userID)
			} else {
				delete(candidateBuffer, cmd.userID)
				log.Printf("[WS] All buffered messages flushed for user %d", cmd.userID)
			}

		case "send-peer-list":
			room, exists := rooms[cmd.roomID]
			if !exists {
				log.Printf("[WS] Room %d not found when sending peer list to user %d", cmd.roomID, cmd.userID)
				break
			}
			conn, ok := connections[cmd.userID]
			if !ok {
				log.Printf("[WS] Connection not found for user %d when sending peer list", cmd.userID)
				break
			}

			var peerList []uint64
			for uid := range room.Users {
				if uid != cmd.userID {
					peerList = append(peerList, uid)
				}
			}
			log.Printf("[WS] Sending peer list to user %d: %v", cmd.userID, peerList)

			err := conn.WriteJSON(Message{
				Type:   TypePeerList,
				RoomID: cmd.roomID,
				Users:  peerList,
			})
			if err != nil {
				log.Printf("[WS] Failed to send peer list to user %d: %v", cmd.userID, err)
			}

		case "start_p2p":
			room, exists := rooms[cmd.roomID]
			if !exists {
				log.Printf("[WS] Room %d not found when starting P2P for user %d", cmd.roomID, cmd.userID)
				break
			}
			conn, ok := connections[cmd.userID]
			if !ok {
				log.Printf("[WS] Connection not found for user %d when starting P2P", cmd.userID)
				break
			}

			room.ReadyMap[cmd.userID] = true
			log.Printf("[WS] User %d marked ready in room %d", cmd.userID, cmd.roomID)

			err := conn.WriteJSON(Message{
				Type:   TypeStart,
				RoomID: cmd.roomID,
				Sender: cmd.userID,
			})
			if err != nil {
				log.Printf("[WS] Failed to send ready notification to user %d: %v", cmd.userID, err)
			}

			readyCount := 0
			for _, ready := range room.ReadyMap {
				if ready {
					readyCount++
				}
			}
			if readyCount == len(room.Users) {
				log.Printf("[WS] All users ready in room %d, starting P2P signaling", cmd.roomID)
				for uid, peerConn := range room.Users {
					if uid != cmd.userID {
						err := peerConn.WriteJSON(Message{
							Type:   TypeStart,
							RoomID: cmd.roomID,
							Sender: cmd.userID,
						})
						if err != nil {
							log.Printf("[WS] Failed to notify user %d about P2P start: %v", uid, err)
						} else {
							log.Printf("[WS] Notified user %d to start P2P", uid)
						}
					}
				}
			}

		default:
			log.Printf("[WS] Unknown command: %s", cmd.cmd)
		}
	}
}

func (wm *WebSocketManager) Authenticate(r *http.Request) bool {
	apiKey := r.Header.Get("X-Api-Key")
	resp := make(chan any)
	wm.commandChan <- managerCommand{
		cmd:      "check-api-key",
		message:  Message{APIKey: apiKey},
		response: resp,
	}
	res := <-resp
	return res.(bool)

}

func (wm *WebSocketManager) AuthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("[AUTH] Invalid method %s, only POST allowed", r.Method)
		return
	}

	var payload struct {
		ApiKey string `json:"apikey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("[AUTH] Failed to decode request body: %v", err)
		return
	}

	log.Printf("[AUTH] AuthHandler called with API key: '%s'", payload.ApiKey)

	resp := make(chan any)
	wm.commandChan <- managerCommand{
		cmd: "auth-assign-userid",
		message: Message{
			APIKey: payload.ApiKey,
		},
		response: resp,
	}

	result := <-resp
	userID, ok := result.(uint64)
	if !ok || userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Printf("[AUTH] Unauthorized API key: '%s' (userID: %v, ok: %v)", payload.ApiKey, userID, ok)
		return
	}

	log.Printf("[AUTH] API key authorized, assigned userID: %d", userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		UserID uint64 `json:"userid"`
	}{UserID: userID})
}

func (wm *WebSocketManager) Handler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-Api-Key")
	resp := make(chan any)

	wm.commandChan <- managerCommand{
		cmd:      "auth-assign-userid",
		message:  Message{APIKey: apiKey},
		response: resp,
	}
	result := <-resp
	userID := result.(uint64)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := wm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}

	resp = make(chan any)
	wm.commandChan <- managerCommand{
		cmd:      "register-connection",
		message:  Message{UserID: userID},
		conn:     conn,
		response: resp,
	}
	alreadyConnected := (<-resp).(bool)
	if alreadyConnected {
		log.Printf("[WS] Duplicate connection for user %d", userID)
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Duplicate connection"))
		conn.Close()
		return
	}

	log.Printf("[WS] User %d connected", userID)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		wm.readMessages(userID, conn)
	}()

	go func() {
		defer wg.Done()
		wm.sendPings(userID, conn)
	}()

	wg.Wait()

	wm.commandChan <- managerCommand{
		cmd:     "unregister-connection",
		message: Message{UserID: userID},
	}
	log.Printf("[WS] User %d disconnected", userID)
}

func (wm *WebSocketManager) readMessages(userID uint64, conn *websocket.Conn) {
	defer func() {
		wm.DisconnectUser(userID)
		conn.Close()
		log.Printf("[WS] User %d disconnected", userID)
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error for user %d: %v", userID, err)
			wm.DisconnectUser(userID)
			if ce, ok := err.(*websocket.CloseError); ok {
				switch ce.Code {
				case websocket.CloseNormalClosure:
					log.Printf("[WS] User %d disconnected normally", userID)
				case websocket.CloseGoingAway:
					log.Printf("[WS] User %d is going away", userID)
				case websocket.CloseAbnormalClosure:
					log.Printf("[WS] Abnormal closure for user %d", userID)
				default:
					log.Printf("[WS] User %d closed with code %d: %s", userID, ce.Code, ce.Text)
				}
			} else {
				log.Printf("[WS] Normal disconnect from %d: %v", userID, err)
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("[WS] Invalid message from %d: %v", userID, err)
			continue
		}

		switch msg.Type {
		case TypeCreateRoom:
			log.Printf("[WS] User %d requested to create a room", userID)
			respChan := make(chan managerCommand)
			wm.commandChan <- managerCommand{
				cmd:      "create-room",
				userID:   userID,
				conn:     conn,
				respChan: respChan,
			}
			resp := <-respChan
			if resp.err != nil {
				log.Printf("[WS] Error creating room: %v", resp.err)
			}

			wm.commandChan <- managerCommand{
				cmd:    "flush_buffered",
				userID: userID,
			}

		case TypeJoin:
			wm.commandChan <- managerCommand{
				cmd:    "join-room",
				userID: userID,
				conn:   conn,
				roomID: msg.RoomID,
			}

			wm.commandChan <- managerCommand{
				cmd:    "flush_buffered",
				userID: userID,
			}

		case TypeOffer, TypeAnswer, TypeICE:
			log.Printf("[WS] Forwarding %s from %d to %d in room %d", msg.Type, msg.Sender, msg.Target, msg.RoomID)
			wm.commandChan <- managerCommand{
				cmd:     "forward_or_buffer",
				message: msg,
			}

		case TypePeerList, "peer-list-request":
			wm.commandChan <- managerCommand{
				cmd:    "send_peer_list",
				userID: userID,
				roomID: msg.RoomID,
			}

		case TypeStart:
			log.Printf("[WS] Received 'start' from user %d in room %d", msg.Sender, msg.RoomID)
			wm.commandChan <- managerCommand{
				cmd:    "start_p2p",
				roomID: msg.RoomID,
			}

		case TypeDisconnect:
			wm.commandChan <- managerCommand{
				cmd:     "disconnect",
				message: msg,
			}

		case TypeText:
			wm.commandChan <- managerCommand{
				cmd:     "text",
				userID:  userID,
				message: msg,
			}

		case "start-session":
			wm.commandChan <- managerCommand{
				cmd:    "start-session",
				userID: userID,
			}

		default:
			log.Printf("[WS] Unknown message type: %s", msg.Type)
		}
	}
}

//func (wm *WebSocketManager) sendPeerListToUser(roomID, userID uint64) {
//	wm.commandChan <- managerCommand{
//		cmd:    "send_peer_list",
//		roomID: roomID,
//		userID: userID,
//	}
//}

//func (wm *WebSocketManager) forwardOrBuffer(senderID uint64, msg Message) {
//	wm.commandChan <- managerCommand{
//		cmd:     "forward_or_buffer",
//		userID:  senderID,
//		message: msg,
//	}
// }

func (wm *WebSocketManager) AddUserToRoom(roomID, userID uint64) {
	wm.commandChan <- managerCommand{
		cmd:    "add_user_to_room",
		roomID: roomID,
		userID: userID,
	}
}

//func (wm *WebSocketManager) flushBufferedMessages(userID uint64) {
//	wm.commandChan <- managerCommand{
//		cmd:    "flush_buffer",
//		userID: userID,
//	}
//}

func (wm *WebSocketManager) CreateRoom(userID uint64) {
	wm.commandChan <- managerCommand{
		cmd:    "create_room",
		userID: userID,
	}
}

func (wm *WebSocketManager) areInSameRoom(roomID uint64, userIDs []uint64) bool {
	resp := make(chan any)
	wm.commandChan <- managerCommand{
		cmd:      "same_room_query",
		roomID:   roomID,
		message:  Message{Users: userIDs},
		response: resp,
	}
	result := <-resp
	inRoom, ok := result.(bool)
	return ok && inRoom
}

func (wm *WebSocketManager) DisconnectUser(userID uint64) {
	wm.commandChan <- managerCommand{
		cmd:    "disconnect_user",
		userID: userID,
	}
}

func (wm *WebSocketManager) HandleDisconnect(msg Message) {
	wm.commandChan <- managerCommand{
		cmd:    "handle_disconnect",
		userID: msg.Sender,
	}
}

func (wm *WebSocketManager) sendPings(userID uint64, conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	failures := 0
	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			failures++
			if failures >= 3 {
				log.Printf("[WS] Ping timeout for user %d", userID)
				conn.Close()
				return
			}
		} else {
			failures = 0
		}
	}
}
