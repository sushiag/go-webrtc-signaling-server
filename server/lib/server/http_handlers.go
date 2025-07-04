package server

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[SERVER] failed to upgrade to WS connection: %v", err)
		return
	}

	newConn := &Connection{
		UserID:   userID,
		Conn:     conn,
		Incoming: make(chan Message),
		Outgoing: make(chan Message),
	}

	newConnCh <- newConn
}
