package e2e_test

import (
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	client "github.com/sushiag/go-webrtc-signaling-server/client/clientwrapper"
	"github.com/sushiag/go-webrtc-signaling-server/server/wsserver"
)

func startTestServer() *http.Server {
	log.Println("running on: 127.0.0.1:8080")
	manager := wsserver.NewWebSocketManager()

	manager.SetValidApiKeys(map[string]bool{
		"valid-api-key-1": true,
		"valid-api-key-2": true,
	})
	mux := http.NewServeMux()
	mux.HandleFunc("/auth", manager.AuthHandler)
	mux.HandleFunc("/ws", manager.Handler)

	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: mux,
	}

	go func() {
		_ = server.ListenAndServe()
	}()

	time.Sleep(100 * time.Millisecond)

	return server
}
func TestEndToEndSignaling(t *testing.T) {
	server := startTestServer()
	defer server.Close() // ensure server is closed after the test

	// pre-set API keys only for testing
	apiKeyA, apiKeyB := "valid-api-key-1", "valid-api-key-2"

	// create two client instances
	clientA := client.NewClient()
	clientB := client.NewClient()

	// pre-set apikeys for testing directly to each client
	clientA.Client.ApiKey = apiKeyA
	clientB.Client.ApiKey = apiKeyB

	// both clients to connect to the signaling server
	clientA.SetServerURL("ws://localhost:8080/ws")
	clientB.SetServerURL("ws://localhost:8080/ws")

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
	clientA.Client.RoomID = 1
	joinRoomID := "1" // fixed RoomID

	// client B joins the room created by Client A
	err = clientB.JoinRoom(joinRoomID)
	assert.NoError(t, err, "Client B failed to join room")

	// wait for signaling to complete and establish peer connection
	time.Sleep(2 * time.Second)

	// host (ClientA) starts the session
	err = clientA.StartSession()
	assert.NoError(t, err, "Client A failed to start session")

	// optionally, start Client B's session as well if needed
	err = clientB.StartSession()
	assert.NoError(t, err, "Client B failed to start session")

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
