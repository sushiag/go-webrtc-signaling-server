package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func TestAuthenticate(t *testing.T) {
	req, err := http.NewRequest("GET", "http://localhost:8080", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Sec-Websocket-Protocol", "replace-with-your-actual-token")
	if !authenticate(req) {
		t.Errorf("Expected true for valid token, got false")
	}

	req.Header.Set("Sec-Websocket-Protocol", "invalid-token")
	if authenticate(req) {
		t.Errorf("Expected false for invalid token, got true")
	}
}

func TestHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	url := "ws" + server.URL[4:] + "/ws"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Sec-Websocket-Protocol", "replace-with-your-actual-token")

	conn, _, err := websocket.DefaultDialer.Dial(req.URL.String(), req.Header)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	err = conn.WriteMessage(websocket.TextMessage, []byte("Hello, World!"))
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	if string(message) != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got '%s'", message)
	}
}

func TestClientIDGeneration(t *testing.T) {
	id1 := uuid.New().String()
	id2 := uuid.New().String()

	if id1 == id2 {
		t.Errorf("Expected different client IDs, got the same: %s", id1)
	}
}
