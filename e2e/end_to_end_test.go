package e2e_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	clienthandle "github.com/sushiag/go-webrtc-signaling-server/client/clienthandler"
	webrtchandle "github.com/sushiag/go-webrtc-signaling-server/client/webrtchandler"
)

// TestClientToClientSignaling performs end-to-end signaling test between two clients
func TestClientToClientSignaling(t *testing.T) {
	// Create two clients
	client1 := clienthandle.NewClient()
	client2 := clienthandle.NewClient()

	// Set API keys and server URLs
	client1.ApiKey = "valid-api-key-1"
	client1.ServerURL = fmt.Sprintf("ws://localhost:%s/ws", getEnv("WS_PORT", "8080"))
	client2.ApiKey = "valid-api-key-2"
	client2.ServerURL = fmt.Sprintf("ws://localhost:%s/ws", getEnv("WS_PORT", "8080"))

	// Perform pre-authentication
	if err := client1.PreAuthenticate(); err != nil {
		t.Fatalf("client1 auth failed: %v", err)
	}
	if err := client2.PreAuthenticate(); err != nil {
		t.Fatalf("client2 auth failed: %v", err)
	}

	// Initialize clients
	if err := client1.Init(); err != nil {
		t.Fatalf("client1 init failed: %v", err)
	}
	defer client1.Close()

	if err := client2.Init(); err != nil {
		t.Fatalf("client2 init failed: %v", err)
	}
	defer client2.Close()

	// Create PeerManager for client2
	pm2 := webrtchandle.NewPeerManager()

	// Channel to signal test completion
	done := make(chan struct{})

	// Setup client1 message handler
	client1.SetMessageHandler(func(msg clienthandle.Message) {
		switch msg.Type {
		case clienthandle.MessageTypeRoomCreated:
			t.Logf("Client1 created room with ID: %v", msg.RoomID)
			go func() {
				roomIDStr := strconv.FormatUint(msg.RoomID, 10)
				t.Logf("Client2 attempting to join room: %s", roomIDStr)
				if err := client2.Join(roomIDStr); err != nil {
					t.Errorf("client2 failed to join room: %v", err)
				}
			}()

		case clienthandle.MessageTypePeerJoined:
			go func() {
				startMsg := clienthandle.Message{
					Type:   clienthandle.MessageTypeStart,
					RoomID: client1.RoomID,
					Sender: client1.UserID,
				}
				if err := client1.Send(startMsg); err != nil {
					t.Errorf("client1 failed to send start message: %v", err)
				}
			}()

		case clienthandle.MessageTypeICECandidate:
			go func() {
				if err := client2.Send(msg); err != nil {
					t.Errorf("client2 failed to send ICE candidate: %v", err)
				}
			}()
		}
	})

	// Setup client2 message handler
	client2.SetMessageHandler(func(msg clienthandle.Message) {
		switch msg.Type {
		case clienthandle.MessageTypeStart:
			t.Logf("Client2 received start signal — signaling successful.")
			done <- struct{}{}

		case clienthandle.MessageTypeICECandidate:
			go func() {
				if err := pm2.HandleICECandidate(msg); err != nil {
					t.Errorf("client2 failed to handle ICE candidate: %v", err)
				}
			}()
		}
	})

	// Start by having client1 create a room
	if err := client1.Start(); err != nil {
		t.Fatalf("client1 failed to create room: %v", err)
	}

	// Wait for signaling to complete
	select {
	case <-done:
		t.Log("✅ End-to-end signaling test passed!")
	case <-time.After(15 * time.Second):
		t.Error("❌ Timeout: signaling between clients did not complete in time")
	}
}

// getEnv safely gets environment variables with fallback
func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// newTestServer simulates creating a test server
func newTestServer() *Server {
	// Initialize the server (mock or real server)
	server := &Server{
		// Set up necessary fields for the server
	}
	return server
}

// newTestClient simulates creating a test client
func newTestClient(server *Server) *Client {
	// Initialize the client
	client := &Client{
		Server: server,
		// Set up necessary fields for the client
	}
	return client
}

func TestHostGoesP2PAndDisconnects(t *testing.T) {
	server := newTestServer()
	host := newTestClient(server)

	// Host creates the room and joins
	roomID := "test-room"
	server.handleMessage(host, signaling.Message{Type: signaling.TypeCreateRoom, RoomID: roomID})

	// Host decides to go P2P
	startMessage := signaling.Message{Type: signaling.TypeStart, RoomID: roomID}
	server.handleMessage(host, startMessage)

	// Verify P2P status and removal from room
	if _, exists := server.rooms[roomID][host.id]; exists {
		t.Errorf("Host was not removed from the room after going P2P.")
	}

	// Verify other users are notified about the host leaving
	if !host.receivedP2PNotification() {
		t.Errorf("Host did not receive P2P notification.")
	}
}
