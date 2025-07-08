package e2e_test

import (
	"database/sql"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	client "github.com/sushiag/go-webrtc-signaling-server/client/lib"
	"github.com/sushiag/go-webrtc-signaling-server/server/lib/db"
	server "github.com/sushiag/go-webrtc-signaling-server/server/lib/server"
)

func TestP2PAfterStartSession(t *testing.T) {
	conn, err := sql.Open("sqlite3", "file:test.db?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}

	queries := db.New(conn)
	server, serverURL := server.StartServer("0", queries)
	defer server.Close()

	apiKeyA := "valid-api-key-1"
	apiKeyB := "valid-api-key-2"

	clientA := client.NewClient(serverURL)
	clientB := client.NewClient(serverURL)

	clientA.SetApiKey(apiKeyA)
	clientB.SetApiKey(apiKeyB)

	err = clientA.Connect()
	assert.NoError(t, err, "Client A failed to connect")

	err = clientB.Connect()
	assert.NoError(t, err, "Client B failed to connect")

	err = clientA.CreateRoom()
	assert.NoError(t, err, "Client A failed to create room")

	time.Sleep(500 * time.Millisecond)

	roomID := strconv.FormatUint(clientA.Websocket.RoomID, 10)
	err = clientB.JoinRoom(roomID)
	assert.NoError(t, err, "Client B failed to join room")

	time.Sleep(500 * time.Millisecond)

	err = clientA.StartSession()
	assert.NoError(t, err, "Client A failed to send start-session")

	time.Sleep(500 * time.Millisecond)

	assert.Nil(t, clientA.Websocket.Conn, "Client A should be disconnected from server")
	assert.Nil(t, clientB.Websocket.Conn, "Client B should be disconnected from server")

	time.Sleep(500 * time.Millisecond)

	err = clientA.SendMessageToPeer(clientB.Websocket.UserID, "P2P-only message from A")
	assert.NoError(t, err, "Client A failed to send P2P message to Client B after StartSession")

	err = clientB.SendMessageToPeer(clientA.Websocket.UserID, "P2P-only message from B")
	assert.NoError(t, err, "Client B failed to send P2P message to Client A after StartSession")

	t.Log("Two clients successfully communicated P2P after StartSession and server disconnect.")
}
