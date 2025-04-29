package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	clienthandle "github.com/sushiag/go-webrtc-signaling-server/client/clienthandler"
	webrtchandle "github.com/sushiag/go-webrtc-signaling-server/client/webrtchandler"
)

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func TestClientToClientSignaling(t *testing.T) {
	client1 := clienthandle.NewClient()
	client2 := clienthandle.NewClient()

	client1.ApiKey = "valid-api-key-1"
	client2.ApiKey = "valid-api-key-2"
	client1.ServerURL = fmt.Sprintf("ws://localhost:%s/ws", getEnv("WS_PORT", "8080"))
	client2.ServerURL = fmt.Sprintf("ws://localhost:%s/ws", getEnv("WS_PORT", "8080"))

	// Authenticate clients
	if err := client1.PreAuthenticate(); err != nil {
		t.Fatalf("client1 auth failed: %v", err)
	}
	if err := client2.PreAuthenticate(); err != nil {
		t.Fatalf("client2 auth failed: %v", err)
	}

	// Connect clients
	if err := client1.Init(); err != nil {
		t.Fatalf("client1 init failed: %v", err)
	}
	if err := client2.Init(); err != nil {
		t.Fatalf("client2 init failed: %v", err)
	}
	defer func() {
		client1.Close()
		client2.Close()
	}()

	peerManager1 := webrtchandle.NewPeerManager() // For client1
	peerManager2 := webrtchandle.NewPeerManager() // For client2

	done := make(chan struct{})

	client1.SetMessageHandler(func(msg clienthandle.Message) {
		switch msg.Type {
		case clienthandle.MessageTypeRoomCreated:
			t.Logf("Client1 created room %v", msg.RoomID)
			go func() {
				roomIDStr := fmt.Sprintf("%d", msg.RoomID)
				if err := client2.Join(roomIDStr); err != nil {
					t.Errorf("client2 join failed: %v", err)
				}
			}()
		case clienthandle.MessageTypePeerJoined:
			t.Logf("Client2 joined, starting offer from client1")
			go func() {
				if err := peerManager1.CreateAndSendOffer(client2.UserID, client1); err != nil {
					t.Errorf("client1 send offer failed: %v", err)
				}
			}()
		case clienthandle.MessageTypeICECandidate:
			t.Logf("Client1 received ICE candidate")
			go func() {
				if err := peerManager1.HandleICECandidate(msg); err != nil {
					t.Errorf("client1 handle ICE failed: %v", err)
				}
			}()
		case clienthandle.MessageTypeAnswer:
			t.Logf("Client1 received answer")
			go func() {
				if err := peerManager1.HandleAnswer(msg, client1); err != nil {
					t.Errorf("client1 handle answer failed: %v", err)
				}
			}()
		}
	})

	client2.SetMessageHandler(func(msg clienthandle.Message) {
		switch msg.Type {
		case clienthandle.MessageTypeOffer:
			t.Logf("Client2 received offer")
			go func() {
				if err := peerManager2.HandleOffer(msg, client2); err != nil {
					t.Errorf("client2 handle offer failed: %v", err)
				}
			}()
		case clienthandle.MessageTypeICECandidate:
			t.Logf("Client2 received ICE candidate")
			go func() {
				if err := peerManager2.HandleICECandidate(msg); err != nil {
					t.Errorf("client2 handle ICE failed: %v", err)
				}
				done <- struct{}{}
			}()
		case clienthandle.MessageTypeAnswer:
			t.Logf("Client2 received unexpected answer")
		}
	})

	// Start signaling by creating a room
	if err := client1.Start(); err != nil {
		t.Fatalf("client1 start failed: %v", err)
	}

	select {
	case <-done:
		t.Log("Client-to-client signaling succeeded")
	case <-time.After(30 * time.Second):
		t.Fatal("Timeout waiting for signaling to complete")
	}

	assert.True(t, client1.IsWebSocketClosed(), "Client1 WebSocket should be closed")
	assert.True(t, client2.IsWebSocketClosed(), "Client2 WebSocket should be closed")
}
