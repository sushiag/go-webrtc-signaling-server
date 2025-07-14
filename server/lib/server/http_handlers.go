package server

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"

	smsg "signaling-msgs"
)

// Handles the /ws endpoint
func handleWSEndpoint(w http.ResponseWriter, r *http.Request, auth *authHandler, newConnCh chan *Connection) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.Header.Get("X-Api-Key")

	userID, err := auth.authenticate(apiKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, http.Header{"X-Client-ID": []string{strconv.FormatUint(userID, 10)}})
	if err != nil {
		log.Printf("[SERVER] failed to upgrade to WS connection: %v", err)
		return
	}

	newConn := &Connection{
		UserID:   userID,
		Conn:     conn,
		Outgoing: make(chan smsg.MessageAnyPayload),
	}

	newConnCh <- newConn
}
