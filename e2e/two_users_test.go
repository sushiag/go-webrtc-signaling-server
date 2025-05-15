package e2e_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	client "github.com/sushiag/go-webrtc-signaling-server/client/lib"
	server "github.com/sushiag/go-webrtc-signaling-server/server/lib/server"
)

func TestEndToEndSignaling(t *testing.T) {
	srv, serverURL := server.StartServer("0")
	defer srv.Close() // ensure server is closed after the test

	// Pre-set API keys for testing
	apiKeyA, apiKeyB := "valid-api-key-1", "valid-api-key-2"

	// Create two client instances
	clientA := client.NewClient(serverURL)
	clientB := client.NewClient(serverURL)

	clientA.Websocket.ApiKey = apiKeyA
	clientB.Websocket.ApiKey = apiKeyB

	err := clientA.Connect()
	assert.NoError(t, err, "Client A failed to connect")

	err = clientB.Connect()
	assert.NoError(t, err, "Client B failed to connect")

	// Client A creates a room
	err = clientA.CreateRoom()
	assert.NoError(t, err, "Client A failed to create room")

	// Wait briefly for the room to be registered
	time.Sleep(1 * time.Second)

	// Use the actual RoomID from Client A
	roomID := strconv.FormatUint(clientA.Websocket.RoomID, 10)

	// Client B joins the room
	err = clientB.JoinRoom(roomID)
	assert.NoError(t, err, "Client B failed to join room")

	// Wait for signaling to complete
	time.Sleep(2 * time.Second)

	// Host (Client A) starts the session
	err = clientA.StartSession()
	assert.NoError(t, err, "Client A failed to start session")

	// Wait for peer connection to be fully established
	time.Sleep(2 * time.Second)

	// Simulate one round of message exchange
	clients := []*client.Client{clientA, clientB}
	for round := 1; round <= 1; round++ {
		t.Logf("---- Round %d ----", round)
		for _, sender := range clients {
			sender.PeerManager.Peers.Range(func(key, value any) bool {
				peerID := key.(uint64)
				message := "Round " + strconv.Itoa(round) + " from client " + strconv.FormatUint(sender.Websocket.UserID, 10)
				err := sender.SendMessageToPeer(peerID, message)
				assert.NoErrorf(t, err, "Failed to send message from client %d to peer %d", sender.Websocket.UserID, peerID)
				return true
			})
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Logf("All clients successfully exchanged messages for 1 round.")
}
