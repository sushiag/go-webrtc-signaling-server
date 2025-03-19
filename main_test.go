package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
)

func TestGenerateHMACToken(t *testing.T) {
	expectedToken := generateHMACToken()
	hash := hmac.New(sha256.New, hmacKey)
	hash.Write([]byte("fixed-data"))
	expectedHash := hex.EncodeToString(hash.Sum(nil))

	if expectedToken != expectedHash {
		t.Errorf("Expected token %s, got %s", expectedHash, expectedToken)
	}
}

func TestAuthenticate(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("X-Auth-Token", generateHMACToken())

	if !authenticate(req) {
		t.Errorf("Authentication failed for a valid token")
	}

	req.Header.Set("X-Auth-Token", "invalid-token")
	if authenticate(req) {
		t.Errorf("Authentication succeeded for an invalid token")
	}
}

func TestWebSocketConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("WebSocket upgrade failed: %v", err)
			return
		}
		conn.Close()
	}))
	defer server.Close()

	url := "ws" + server.URL[4:] + "/ws"
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Errorf("Failed to connect to WebSocket: %v", err)
	}
	conn.Close()
}

func TestBroadcastMessage(t *testing.T) {
	roomID := "test-room"
	senderID := "sender"

	room := &Room{
		Clients: make(map[string]*Client),
	}
	rooms[roomID] = room

	mockConn := &websocket.Conn{}
	client := &Client{ID: senderID, Conn: mockConn}
	room.Clients[senderID] = client

	message := []byte("Hello, WebRTC!")
	broadcastMessage(roomID, senderID, message)
}

func TestDisconnectClient(t *testing.T) {
	roomID := "testRoom"
	rooms[roomID] = &Room{Clients: make(map[string]*Client)}

	client := &Client{ID: "client1", Room: roomID}
	rooms[roomID].Clients[client.ID] = client

	disconnectClient(client)

	if _, exists := rooms[roomID].Clients[client.ID]; exists {
		t.Errorf("Client was not removed from room after disconnection")
	}
}
