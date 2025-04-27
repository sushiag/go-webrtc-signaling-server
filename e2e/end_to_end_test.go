package e2e_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	clienthandle "github.com/sushiag/go-webrtc-signaling-server/client/clienthandler"
	webrtchandle "github.com/sushiag/go-webrtc-signaling-server/client/webrtchandler"
)

// Track if client2 received ICE
var gotICEFromClient1 bool

func TestClientToClientSignaling(t *testing.T) {
	client1 := clienthandle.NewClient()
	client2 := clienthandle.NewClient()

	client1.ApiKey = "valid-api-key-1"
	client1.ServerURL = fmt.Sprintf("ws://localhost:%s/ws", getEnv("WS_PORT", "8080"))
	client2.ApiKey = "valid-api-key-2"
	client2.ServerURL = fmt.Sprintf("ws://localhost:%s/ws", getEnv("WS_PORT", "8080"))

	// Authenticate clients before proceeding
	if err := client1.PreAuthenticate(); err != nil {
		t.Fatalf("client1 auth failed: %v", err)
	}
	if err := client2.PreAuthenticate(); err != nil {
		t.Fatalf("client2 auth failed: %v", err)
	}

	// Initialize clients and establish WebSocket connections
	if err := client1.Init(); err != nil {
		t.Fatalf("client1 init failed: %v", err)
	}
	defer client1.Close()

	if err := client2.Init(); err != nil {
		t.Fatalf("client2 init failed: %v", err)
	}
	defer client2.Close()

	// Setup PeerManager for handling ICE candidates
	pm2 := webrtchandle.NewPeerManager()
	client2.UserID = 2

	// This channel will signal when signaling is complete
	done := make(chan struct{})
	// Set message handler for client2
	client2.SetMessageHandler(func(msg clienthandle.Message) {
		switch msg.Type {
		case clienthandle.MessageTypeStart:
			t.Logf("Client2 received start signal — now waiting for ICE candidates...")

		case clienthandle.MessageTypeICECandidate:
			t.Logf("Client2 received ICE candidate from client1.")

			go func() {
				if err := pm2.HandleICECandidate(msg); err != nil {
					t.Errorf("client2 failed to handle ICE candidate: %v", err)
				}
			}()

			gotICEFromClient1 = true
			// Once we get ICE from client1, signaling is considered complete
			done <- struct{}{}
		}
	})

	// Set message handler for client1
	client1.SetMessageHandler(func(msg clienthandle.Message) {
		switch msg.Type {
		case clienthandle.MessageTypeRoomCreated:
			t.Logf("Client1 created room with ID: %v", msg.RoomID)
			// Simulate client2 joining the room
			go func() {
				roomIDStr := fmt.Sprintf("%d", msg.RoomID)
				if err := client2.Join(roomIDStr); err != nil {
					t.Errorf("client2 failed to join room: %v", err)
				}
			}()

		case clienthandle.MessageTypePeerJoined:
			t.Logf("Client2 joined the room. Sending start message.")
			startMsg := clienthandle.Message{
				Type:   clienthandle.MessageTypeStart,
				RoomID: client1.RoomID,
				Sender: client1.UserID,
			}
			if err := client1.Send(startMsg); err != nil {
				t.Errorf("client1 failed to send start message: %v", err)
			}

		case clienthandle.MessageTypeICECandidate:
			t.Logf("Client1 received ICE candidate from server.")
			// Forward ICE candidate to client2
			go func() {
				if err := client2.Send(msg); err != nil {
					t.Errorf("client1 failed to forward ICE candidate to client2: %v", err)
				}
			}()
		}
	})

	// Start the signaling: Client1 creates room
	if err := client1.Start(); err != nil {
		t.Fatalf("client1 failed to create room: %v", err)
	}

	// Wait for ICE exchange to complete or timeout
	select {
	case <-done:
		t.Log("✅ End-to-end signaling test passed!")
	case <-time.After(90 * time.Second):
		t.Fatal("❌ Timeout: signaling between clients did not complete in time")
	}

	// Ensure the WebSocket connections are properly closed
	assert.True(t, client1.IsWebSocketClosed(), "WebSocket for client1 should be closed")
	assert.True(t, client2.IsWebSocketClosed(), "WebSocket for client2 should be closed")
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
