package e2e_test

import (
	"strconv"
	"testing"
	"time"

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

	time.Sleep(600 * time.Millisecond)

	roomID := strconv.FormatUint(clientA.Websocket.RoomID, 10)
	err = clientB.JoinRoom(roomID)
	assert.NoError(t, err, "Client B failed to join room")

	time.Sleep(500 * time.Millisecond)

	err = clientA.SendMessageToPeer(clientB.Websocket.UserID, "hello from A")
	assert.NoError(t, err, "Client A failed to send message to Client B before StartSession")

	err = clientB.SendMessageToPeer(clientA.Websocket.UserID, "hello from B")
	assert.NoError(t, err, "Client B failed to send message to Client A before StartSession")

	time.Sleep(500 * time.Millisecond) // adjusted time so it doesn't close before peers connects
	err = clientA.StartSession()
	assert.NoError(t, err, "Client A failed to send start-session")

	time.Sleep(500 * time.Millisecond)

	assert.Nil(t, clientA.Websocket.Conn, "Client A should be disconnected from server")
	assert.Nil(t, clientB.Websocket.Conn, "Client B should be disconnected from server")

	err = clientA.SendMessageToPeer(clientB.Websocket.UserID, "P2P-only message from A")
	assert.NoError(t, err, "Client A failed to send P2P message to Client B after StartSession")

	err = clientB.SendMessageToPeer(clientA.Websocket.UserID, "P2P-only message from B")
	assert.NoError(t, err, "Client B failed to send P2P message to Client A after StartSession")

	t.Log("Two clients successfully communicated P2P after StartSession and server disconnect.")
}
