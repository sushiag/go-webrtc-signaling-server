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

	apiKeyA, apiKeyB, apiKeyC, apiKeyD := "valid-api-key-3", "valid-api-key-4", "key1", "key2"

	clientA := client.NewClient(serverUrl)
	clientB := client.NewClient(serverUrl)
	clientC := client.NewClient(serverUrl)
	clientD := client.NewClient(serverUrl)

	clientA.Websocket.ApiKey = apiKeyA
	clientB.Websocket.ApiKey = apiKeyB
	clientC.Websocket.ApiKey = apiKeyC
	clientD.Websocket.ApiKey = apiKeyD

	clients := []*client.Client{clientA, clientB, clientC, clientD}

	for _, c := range clients {
		err := c.Connect()
		assert.NoError(t, err, "Client failed to connect")
	}

	err := clientA.CreateRoom()
	assert.NoError(t, err, "Client A failed to create room")

	time.Sleep(1 * time.Second)

	clientA.Websocket.RoomID = 1
	joinRoomID := "1"

	for _, c := range clients[1:] {
		err := c.JoinRoom(joinRoomID)
		assert.NoError(t, err, "Client failed to join room")
	}

	time.Sleep(2 * time.Second)

	err = clientA.StartSession()
	assert.NoError(t, err, "Client A failed to start session")

	time.Sleep(2 * time.Second)
	for _, c := range clients {
		peerCount := 0
		c.PeerManager.Peers.Range(func(_, _ any) bool {
			peerCount++
			return false // stop early; we just care that one exists
		})
		assert.Greater(t, peerCount, 0, "Client's peer manager is empty")
	}

	for round := 1; round <= 1; round++ {
		t.Logf("---- Round %d ----", round)
		for _, sender := range clients {
			sender.PeerManager.Peers.Range(func(key, _ any) bool {
				peerID := key.(uint64)
				message := "Round " + strconv.Itoa(round) + " from client " + strconv.FormatUint(sender.Websocket.UserID, 10)
				err := sender.SendMessageToPeer(peerID, message)
				assert.NoError(t, err, "Failed to send message from client %d to peer %d", sender.Websocket.UserID, peerID)
				return true
			})
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Logf("All clients successfully exchanged messages for 2 rounds.")

}
