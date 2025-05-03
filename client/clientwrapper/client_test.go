package client_test

import (
	"testing"
	"time"

	client "github.com/sushiag/go-webrtc-signaling-server/client/clientwrapper"
)

func TestFullClientWorkflow(t *testing.T) {
	w := client.NewClient()

	// connect to signaling server
	if err := w.Connect(); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer w.Close()

	// create a new room
	if err := w.CreateRoom(); err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}

	// simulate some time to wait for peer signaling
	time.Sleep(2 * time.Second)

	// start the session (which triggers the sends offers)
	if err := w.StartSession(); err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	peerID := uint64(1)
	err := w.SendMessageToPeer(peerID, "Hello from test!")
	if err != nil {
		t.Logf("SendMessageToPeer failed (likely due to mock peer ID): %v", err)
	}
}
func TestP2PConnectionAfterServerClose(t *testing.T) {
	// define API keys for each client
	apiKeyA, apiKeyB := "valid-api-key-1", "valid-api-key-2"

	// create and then connect client
	clientA, clientB := client.NewClient(), client.NewClient()

	// set the API keys directly on the Client struct
	clientA.Client.ApiKey, clientB.Client.ApiKey = apiKeyA, apiKeyB

	// connect both clients to the signaling server
	if err := clientA.Connect(); err != nil {
		t.Fatalf("ClientA Connect failed: %v", err)
	}
	if err := clientB.Connect(); err != nil {
		t.Fatalf("ClientB Connect failed: %v", err)
	}

	// create a room with ClientA, then have ClientB join it
	if err := clientA.CreateRoom(); err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}

	// Add a brief sleep to ensure room is properly created before joining
	time.Sleep(1 * time.Second)

	// Ensure ClientB joins the room
	if err := clientB.JoinRoom("4"); err != nil { // adjust the RoomID manually for this unit test
		t.Fatalf("ClientB JoinRoom failed: %v", err)
	}

	// simulate a brief wait for signaling exchange to complete
	time.Sleep(2 * time.Second)

	// start the WebRTC sessions (this sends the offers and establishes the connection)
	if err := clientA.StartSession(); err != nil {
		t.Fatalf("ClientA StartSession failed: %v", err)
	}
	if err := clientB.StartSession(); err != nil {
		t.Fatalf("ClientB StartSession failed: %v", err)
	}

	// now both clients should be in the session, send a message from ClientA to ClientB
	var peerID uint64

	// get the peer ID from ClientA's PeerManager (should be ClientB)
	for id := range clientA.PeerManager.Peers {
		peerID = id
		break
	}

	// send a message from ClientA to ClientB
	if err := clientA.SendMessageToPeer(peerID, "Hello from ClientA!"); err != nil {
		t.Fatalf("ClientA SendMessageToPeer failed: %v", err)
	}

	// now, after sending the message, we close the signaling server to simulate the peer-to-peer-only state
	clientA.CloseServer()

	// test that the peer-to-peer communication still works
	// send another message from ClientA to ClientB
	if err := clientA.SendMessageToPeer(peerID, "Hello again from ClientA!"); err != nil {
		t.Fatalf("ClientA SendMessageToPeer failed after server close: %v", err)
	}

	// added a brief wait to allow for message delivery
	time.Sleep(1 * time.Second)

	// test passing!
	t.Logf("Test passed: Message sent from ClientA to ClientB even after server close")
}
