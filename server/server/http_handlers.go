package server

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/sushiag/go-webrtc-signaling-server/server/server/db"

	smsg "signaling-msgs"
)

// This handles the /ws endpoint for upgrading the HTTP request
func handleWSEndpoint(w http.ResponseWriter, r *http.Request, newConnCh chan *Connection, queries *db.Queries) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// This authenticate the client if they match the API-key from the database
	user, err := getUserFromAPIKey(r, queries)
	if err != nil {
		log.Printf("[WS] Unauthorized WebSocket attempt: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// This upgrades the http request to a websocket connection
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// This attach the user's ID as a custom response header
	header := http.Header{}
	header.Set("X-Client-ID", strconv.FormatUint(uint64(user.ID), 10))

	// This performs the actual upgrade of the websocket connection
	conn, err := upgrader.Upgrade(w, r, header)
	if err != nil {
		log.Printf("[WS] Failed to upgrade to WebSocket: %v", err)
		return
	}

	// This creayes a new connection instance for thois websocket
	newConn := &Connection{
		UserID:   uint64(user.ID),
		Conn:     conn,
		Outgoing: make(chan smsg.MessageAnyPayload),
	}

	// This sends the new connection into the manager's channel to be handled
	newConnCh <- newConn
	log.Printf("[WS] WebSocket connection established for user %s (ID %d)", user.Username, user.ID)
}
