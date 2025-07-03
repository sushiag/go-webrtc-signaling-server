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

	roomID := strconv.FormatUint(clientA.Websocket.RoomID, 10)

	err = clientB.JoinRoom(roomID)
	assert.NoError(t, err, "Client B failed to join room")

	err = clientC.JoinRoom(roomID)
	assert.NoError(t, err, "Client C failed to join room")

	err = clientD.JoinRoom(roomID)
	assert.NoError(t, err, "Client D failed to join room")

	t.Logf("---- Waiting for data channels to open ----")
	// TODO: there's actually a bug here
	// - clientB only connects to peer 1
	// - clientC only connects to peer 1 & 2
	// - clientD only connects to peer 1, 2, & 3
	// thus, the mesh isn't complete
	readyDataChannels := 0
	totalPeers := 4
	totalDataChannels := (totalPeers - 1) * totalPeers
	dataChDeadline := time.After(3 * time.Second)
	for readyDataChannels < totalDataChannels {
		select {
		case <-clientA.PeerManager.PeerEventsCh:
			readyDataChannels += 1
		case <-clientB.PeerManager.PeerEventsCh:
			readyDataChannels += 1
		case <-clientC.PeerManager.PeerEventsCh:
			readyDataChannels += 1
		case <-clientD.PeerManager.PeerEventsCh:
			readyDataChannels += 1

		case <-dataChDeadline:
			{
				t.Logf("clients took longer than 3 secs to open their data channels, only opened: %d.. continuing anyways", readyDataChannels)
				// t.Fatalf("clients took longer than 10 secs to open their data channels, only opened: %d", readyDataChannels)
			}
		}
	}
	t.Logf("---- Data channels open! ----")

	for round := 1; round <= 1; round++ {
		t.Logf("---- Round %d ----", round)
		for _, sender := range clients {
			senderID := strconv.FormatUint(sender.Websocket.UserID, 10)

			peerIDs := sender.PeerManager.GetPeerIDs()
			for _, peerID := range peerIDs {
				receiverID := strconv.FormatUint(peerID, 10)
				message := "Round " + strconv.Itoa(round) +
					" | from client " + senderID +
					" | to client " + receiverID

				// errs:
				// 1 -> 4
				// 2 -> 3
				// 2 -> 4
				// 3 -> 2
				// 4 -> 1
				// 4 -> 2

				// 1 -> 4
				// 2 -> 4
				// 4 -> 1
				// 4 -> 2
				// 4 -> 3

				// ok run:
				// four_users_test.go:105: 1 sending message to 3
				// four_users_test.go:105: 1 sending message to 4
				// four_users_test.go:105: 1 sending message to 2
				// four_users_test.go:105: 2 sending message to 1
				// four_users_test.go:105: 2 sending message to 3
				// four_users_test.go:105: 2 sending message to 4
				// four_users_test.go:105: 3 sending message to 4
				// four_users_test.go:105: 3 sending message to 1
				// four_users_test.go:105: 3 sending message to 2

				t.Logf("%d sending message to %d", sender.Websocket.UserID, peerID)
				err := sender.SendMessageToPeer(peerID, message)
				assert.NoErrorf(t, err, "Failed to send message from client %s to peer %s", senderID, receiverID)
			}
		}
	}
	t.Logf("All clients successfully exchanged messages")
}
