package e2e_test

import (
	"net/http"
	"testing"
	"time"

	client "github.com/sushiag/go-webrtc-signaling-server/client/clientwrapper"
	"github.com/sushiag/go-webrtc-signaling-server/server/wsserver"
)

func startTestServer() *http.Server {
	manager := wsserver.NewWebSocketManager()

	// Add test API keys
	manager.SetValidApiKeys(map[string]bool{
		"test-api-key-1": true,
		"test-api-key-2": true,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/auth", manager.AuthHandler)
	mux.HandleFunc("/ws", manager.Handler)

	server := &http.Server{
		Addr:    "127.0.0.1:8081", // Use a different port for testing
		Handler: mux,
	}

	go func() {
		_ = server.ListenAndServe()
	}()

	// Optional: wait a moment for the server to be ready
	time.Sleep(100 * time.Millisecond)

	return server
}

func TestEndToEndSignaling(t *testing.T) {
	server := startTestServer()
	defer server.Close()

	// Here, run your actual test code that:
	// 1. Creates clients with `NewClient()`
	// 2. Calls `Connect()`, `CreateRoom()`, `JoinRoom()`, etc.
	// 3. Asserts WebRTC connections are working, etc.

	clientA := client.NewClient()
	clientB := client.NewClient()

	err := clientA.client.Connect()
	if err != nil {
		t.Fatalf("Client A failed to connect: %v", err)
	}

	err = clientB.client.Connect()
	if err != nil {
		t.Fatalf("Client B failed to connect: %v", err)
	}

	err = clientA.client.CreateRoom()
	if err != nil {
		t.Fatalf("Client A failed to create room: %v", err)
	}

	roomID := clientA.client.Client.RoomID
	err = clientB.client.JoinRoom(roomID)
	if err != nil {
		t.Fatalf("Client B failed to join room: %v", err)
	}

	// You can add logic here to test message exchange, session start, etc.
}
