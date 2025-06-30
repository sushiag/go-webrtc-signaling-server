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

	apiKeyA, apiKeyB := "valid-api-key-1", "valid-api-key-2"

	clientA := client.NewClient(serverURL)
	clientB := client.NewClient(serverURL)

	clientA.Websocket.ApiKey = apiKeyA
	clientB.Websocket.ApiKey = apiKeyB

	err := clientA.Connect()
	assert.NoError(t, err, "Client A failed to connect")

	err = clientB.Connect()
	assert.NoError(t, err, "Client B failed to connect")

	err = clientA.CreateRoom()
	assert.NoError(t, err, "Client A: failed to create room")

	time.Sleep(500 * time.Millisecond)

	roomID := strconv.FormatUint(clientA.Websocket.RoomID, 10)

	time.Sleep(500 * time.Millisecond)

	err = clientB.JoinRoom(roomID)
	assert.NoError(t, err, "Client B failed to join room")
	time.Sleep(2 * time.Second)

	clients := []*client.Client{clientA, clientB}
	for round := 1; round <= 1; round++ {
		t.Logf("---- Round %d ----", round)
		for _, sender := range clients {
			for _, peerID := range sender.PeerManager.GetPeerIDs() {
				message := "Round " + strconv.Itoa(round) + " from client " + strconv.FormatUint(sender.Websocket.UserID, 10)
				err := sender.SendMessageToPeer(peerID, message)
				assert.NoErrorf(t, err, "Failed to send message from client %d to peer %d", sender.Websocket.UserID, peerID)
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Logf("All clients successfully exchanged messages for 1 round.")
}
