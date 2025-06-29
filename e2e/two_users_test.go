package e2e_test

import (
	"strconv"
	"testing"

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

	roomID := strconv.FormatUint(clientA.Websocket.RoomID, 10)

	err = clientB.JoinRoom(roomID)
	assert.NoError(t, err, "Client B failed to join room")

	clients := []*client.Client{clientA, clientB}
	for round := 1; round <= 1; round++ {
		t.Logf("---- Round %d ----", round)
		for _, sender := range clients {
			for peerID := range sender.PeerManager.Peers {
				message := "Round " + strconv.Itoa(round) + " from client " + strconv.FormatUint(sender.Websocket.UserID, 10)
				err := sender.SendMessageToPeer(peerID, message)
				assert.NoErrorf(t, err, "Failed to send message from client %d to peer %d", sender.Websocket.UserID, peerID)
			}
		}
	}

	t.Logf("All clients successfully exchanged messages for 1 round.")
}
