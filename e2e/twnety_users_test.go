package e2e_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	client "github.com/sushiag/go-webrtc-signaling-server/client/lib"
	server "github.com/sushiag/go-webrtc-signaling-server/server/lib/server"
)

func TestEndToEndSignalingTwentyUsers(t *testing.T) {
	server, serverUrl := server.StartServer("0")
	defer server.Close()

	numClients := 10
	baseApiKey := "valid-api-key-"
	clients := make([]*client.Client, numClients)

	// 20 lucky rolls, which will fail
	for i := 0; i < numClients; i++ {
		c := client.NewClient(serverUrl)
		c.Websocket.ApiKey = baseApiKey + strconv.Itoa(i+1)
		clients[i] = c
	}

	for i, c := range clients {
		err := c.Connect()
		assert.NoError(t, err, "Client %d failed to connect", i)
	}

	// First client creates a room
	err := clients[0].CreateRoom()
	assert.NoError(t, err, "Client 0 failed to create room")

	time.Sleep(1 * time.Second)
	roomID := strconv.FormatUint(clients[0].Websocket.RoomID, 10)

	// other clients join the room
	for i := 1; i < numClients; i++ {
		err = clients[i].JoinRoom(roomID)
		assert.NoErrorf(t, err, "Client %d failed to join room", i)
	}

	time.Sleep(2 * time.Second)

	// client starts the session
	err = clients[0].StartSession()
	assert.NoError(t, err, "Client 0 failed to start session")

	t.Logf("All 20 clients successfully exchanged messages")
}
