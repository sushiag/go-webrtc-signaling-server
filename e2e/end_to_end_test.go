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
	server, serverUrl := server.StartServer("0")
	defer server.Close() // ensure server is closed after the test

	// pre-set API keys only for testing
	apiKeyA, apiKeyB := "valid-api-key-1", "valid-api-key-2"

	// create two client instances
	clientA := client.NewClient(serverUrl)
	clientB := client.NewClient(serverUrl)

	// pre-set apikeys for testing directly to each client
	clientA.Websocket.ApiKey = apiKeyA
	clientB.Websocket.ApiKey = apiKeyB

	err := clientA.Connect()
	assert.NoError(t, err, "Client A failed to connect")

	err = clientB.Connect()
	assert.NoError(t, err, "Client B failed to connect")

	// client A creates a room
	err = clientA.CreateRoom()
	assert.NoError(t, err, "Client A failed to create room")

	// timer for room creation is fully acknowledged before proceeding
	time.Sleep(1 * time.Second)

	// client A sets the fixed RoomID
	clientA.Websocket.RoomID = 1
	joinRoomID := "1" // fixed RoomID

	// client B joins the room created by Client A
	err = clientB.JoinRoom(joinRoomID)
	assert.NoError(t, err, "Client B failed to join room")

	// wait for signaling to complete and establish peer connection
	time.Sleep(2 * time.Second)

	// host (ClientA) starts the session
	err = clientA.StartSession()
	assert.NoError(t, err, "Client A failed to start session")

	err = clientA.StartSession()
	assert.NoError(t, err, "Client A failed to start session")

	// wait for peer connection to be fully established
	time.Sleep(2 * time.Second)

	// assert that both clients are connected and have valid peers
	assert.NotNil(t, clientA.PeerManager.Peers, "Client A's peer manager is empty")
	assert.NotNil(t, clientB.PeerManager.Peers, "Client B's peer manager is empty")

	// send a test message from ClientA to ClientB
	var peerID uint64
	for id := range clientA.PeerManager.Peers {
		peerID = id
		break
	}

	// ensure the peerID is valid and then send the message
	assert.NotZero(t, peerID, "No valid peer found for Client A to send message")

	err = clientA.SendMessageToPeer(peerID, "Hello from ClientA!")
	assert.NoError(t, err, "Client A failed to send message to Client B")

	// log success message
	t.Logf("End-to-end signaling test passed: Clients connected, room created, sessions started, and message sent.")
}

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
		assert.NotNil(t, c.PeerManager.Peers, "Client's peer manager is empty")
	}

	for round := 1; round <= 5; round++ {
		t.Logf("---- Round %d ----", round)
		for _, sender := range clients {
			for peerID := range sender.PeerManager.Peers {
				message := "Round " + strconv.Itoa(round) + " from client " + strconv.FormatUint(sender.Websocket.UserID, 10)
				err := sender.SendMessageToPeer(peerID, message)
				assert.NoError(t, err, "Failed to send message from client %d to peer %d", sender.Websocket.UserID, peerID)
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Logf("All clients successfully exchanged messages for 5 rounds.")
}
