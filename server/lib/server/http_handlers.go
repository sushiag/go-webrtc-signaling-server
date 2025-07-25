package server

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"

	smsg "signaling-msgs"
)

func handleWSEndpoint(w http.ResponseWriter, r *http.Request, newConnCh chan *Connection) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := getUserFromAPIKey(r)
	if err != nil {
		log.Printf("[WS] Unauthorized WebSocket attempt: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	header := http.Header{}
	header.Set("X-Client-ID", strconv.FormatUint(uint64(user.ID), 10))

	conn, err := upgrader.Upgrade(w, r, header)
	if err != nil {
		log.Printf("[WS] Failed to upgrade to WebSocket: %v", err)
		return
	}

	newConn := &Connection{
		UserID:   uint64(user.ID),
		Conn:     conn,
		Outgoing: make(chan smsg.MessageAnyPayload),
	}

	newConnCh <- newConn
	log.Printf("[WS] WebSocket connection established for user %s (ID %d)", user.Username, user.ID)
}
