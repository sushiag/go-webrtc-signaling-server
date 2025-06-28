package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

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
