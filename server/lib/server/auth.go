package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	smsg "signaling-msgs"

	"github.com/gorilla/websocket"
)

func (wsm *WebSocketManager) SafeWriteJSON(c *Connection, v smsg.MessageAnyPayload) error {
	c.Outgoing <- v
	return nil
}
func (wsm *WebSocketManager) Authenticate(r *http.Request) (uint64, bool) {
	apikey := r.Header.Get("X-Api-Key")
	if apikey == "" {
		log.Printf("[AUTHENTICATION] Missing API key")
		return 0, false
	}

	user, err := queries.GetUserByApikeys(r.Context(), apikey)
	if err != nil {
		log.Printf("[AUTHENTICATION] Invalid API key")
		return 0, false
	}

	log.Printf("[AUTHENTICATION] User %s#%d authenticated", user.Username, user.ID)
	return uint64(user.ID), true
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

	user, err := queries.GetUserByApikeys(r.Context(), payload.ApiKey)
	if err != nil {
		http.Error(w, "Unauthorized access", http.StatusUnauthorized)
		return
	}

	w.Header().Set("X-Client-ID", strconv.FormatUint(uint64(user.ID), 10))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		UserID   uint64 `json:"userID"`
		Username string `json:"username"`
	}{
		UserID:   uint64(user.ID),
		Username: user.Username,
	})
}

func (wsm *WebSocketManager) Handler(w http.ResponseWriter, r *http.Request) {
	userID, ok := wsm.Authenticate(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

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

	log.Printf("[WS] User %d connected", userID)

}
