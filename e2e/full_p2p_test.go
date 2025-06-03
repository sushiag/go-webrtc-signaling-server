package e2e_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	client "github.com/sushiag/go-webrtc-signaling-server/client/lib"
	server "github.com/sushiag/go-webrtc-signaling-server/server/lib/server"
)

func TestP2PAfterStartSession(t *testing.T) {
	srv, serverURL := server.StartServer("0")
	defer srv.Close()

	apiKeyA := "valid-api-key-1"
	apiKeyB := "valid-api-key-2"

	clientA := client.NewClient(serverURL)
	clientB := client.NewClient(serverURL)

	clientA.SetApiKey(apiKeyA)
	clientB.SetApiKey(apiKeyB)

	err := clientA.Connect()
	assert.NoError(t, err, "Client A failed to connect")

	err = clientB.Connect()
	assert.NoError(t, err, "Client B failed to connect")

	err = clientA.CreateRoom()
	assert.NoError(t, err, "Client A failed to create room")

	roomID := strconv.FormatUint(clientA.Websocket.RoomID, 10)
	err = clientB.JoinRoom(roomID)
	assert.NoError(t, err, "Client B failed to join room")

	err = clientA.SendMessageToPeer(clientB.Websocket.UserID, "hello from A")
	assert.NoError(t, err, "Client A failed to send message to Client B before StartSession")

	err = clientB.SendMessageToPeer(clientA.Websocket.UserID, "hello from B")
	assert.NoError(t, err, "Client B failed to send message to Client A before StartSession")

	err = clientA.StartSession()
	assert.NoError(t, err, "Client A failed to send start-session")

	err = clientA.SendMessageToPeer(clientB.Websocket.UserID, "are you still there?")
	assert.NoError(t, err, "Client A should still be able to send message to Client B via P2P")

	err = clientB.SendMessageToPeer(clientA.Websocket.UserID, "yes!")
	assert.NoError(t, err, "Client B should still be able to send message to Client A via P2P")

	t.Log("Two clients successfully communicated P2P after StartSession and server disconnect.")
}
