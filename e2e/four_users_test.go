package e2e_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	client "github.com/sushiag/go-webrtc-signaling-server/client/lib"
	server "github.com/sushiag/go-webrtc-signaling-server/server/lib/server"
)

func TestEndToEndSignalingFourUsers(t *testing.T) {
	server, serverUrl := server.StartServer("0")
	defer server.Close()

	apiKeyA, apiKeyB, apiKeyC, apiKeyD := "valid-api-key-1", "valid-api-key-2", "valid-api-key-3", "valid-api-key-4"

	clientA := client.NewClient(serverUrl)
	clientB := client.NewClient(serverUrl)
	clientC := client.NewClient(serverUrl)
	clientD := client.NewClient(serverUrl)

	clientA.Websocket.ApiKey = apiKeyA
	clientB.Websocket.ApiKey = apiKeyB
	clientC.Websocket.ApiKey = apiKeyC
	clientD.Websocket.ApiKey = apiKeyD

	clients := []*client.Client{clientA, clientB, clientC, clientD}

	for i, c := range clients {
		err := c.Connect()
		assert.NoError(t, err, "Client %d failed to connect", i)
	}

	err := clientA.CreateRoom()
	assert.NoError(t, err, "Client A failed to create room")

	time.Sleep(500 * time.Millisecond)

	roomID := strconv.FormatUint(clientA.Websocket.RoomID, 10)

	err = clientB.JoinRoom(roomID)
	assert.NoError(t, err, "Client B failed to join room")

	err = clientC.JoinRoom(roomID)
	assert.NoError(t, err, "Client C failed to join room")

	err = clientD.JoinRoom(roomID)
	assert.NoError(t, err, "Client D failed to join room")

	time.Sleep(5 * time.Second)

	for round := 1; round <= 1; round++ {
		t.Logf("---- Round %d ----", round)
		for _, sender := range clients {
			for peerID := range sender.PeerManager.Peers {
				message := "Round " + strconv.Itoa(round) + " from client " + strconv.FormatUint(sender.Websocket.UserID, 10)
				err := sender.SendMessageToPeer(peerID, message)
				assert.NoErrorf(t, err, "Failed to send message from client %d to peer %d", sender.Websocket.UserID, peerID)
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Logf("All clients successfully exchanged messages")
}
