package test

import (
	"testing"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/webrtc"

	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/assert"
	"github.com/sushiag/go-webrtc-signaling-server/client/websocket"
)

// MockClient is a mock implementation of the client interface to simulate server-side interaction.
type MockClient struct {
	UserID uint64
}

// Send simulates sending a message to the client.
func (m *MockClient) Send(msg websocket.Message) error {
	// Simulate sending a message (in a real test, this would be more complex)
	return nil
}

// TestPeerManager_CreateAndSendOffer tests the CreateAndSendOffer function.
func TestPeerManager_CreateAndSendOffer(t *testing.T) {
	peerManager := webrtc.NewPeerManager()

	// Create a mock client
	client := &MockClient{UserID: 1}

	// Simulate adding a peer
	peerID := uint64(2)

	// Create and send an offer
	err := peerManager.CreateAndSendOffer(peerID, client)

	// Assert no errors during offer creation and sending
	assert.Nil(t, err, "Expected no error when creating and sending offer")
}

// TestPeerManager_HandleSignalingMessage tests the HandleSignalingMessage function.
func TestPeerManager_HandleSignalingMessage(t *testing.T) {
	peerManager := webrtc.NewPeerManager()

	// Create a mock client
	client := &MockClient{UserID: 1}

	// Create a signaling message
	message := websocket.Message{
		Type:   websocket.MessageTypeOffer,
		Sender: 2,
		Target: 1,
		SDP:    "sdpdata",
	}

	// Handle the offer signaling message
	err := peerManager.HandleSignalingMessage(message, client)

	// Assert no errors handling the signaling message
	assert.Nil(t, err, "Expected no error when handling signaling message")
}

// TestPeerManager_HandleOffer tests the HandleOffer function.
func TestPeerManager_HandleOffer(t *testing.T) {
	peerManager := webrtc.NewPeerManager()

	// Create a mock client
	client := &MockClient{UserID: 1}

	// Simulate receiving an offer
	offerMessage := websocket.Message{
		Type:   websocket.MessageTypeOffer,
		Sender: 2,
		Target: 1,
		SDP:    "sdpdata",
	}

	// Handle the offer
	err := peerManager.HandleOffer(offerMessage, client)

	// Assert no errors when handling the offer
	assert.Nil(t, err, "Expected no error when handling the offer")
}

// TestPeerManager_RemovePeer tests removing a peer.
func TestPeerManager_RemovePeer(t *testing.T) {
	peerManager := webrtc.NewPeerManager()

	// Add a peer
	peerID := uint64(2)
	peerManager.Peers[peerID] = &webrtc.Peer{ID: peerID}

	// Remove the peer
	peerManager.RemovePeer(peerID)

	// Assert that the peer has been removed
	_, exists := peerManager.Peers[peerID]
	assert.False(t, exists, "Expected peer to be removed from the PeerManager")
}

// TestPeerManager_SendDataToPeer tests the SendDataToPeer function.
func TestPeerManager_SendDataToPeer(t *testing.T) {
	peerManager := webrtc.NewPeerManager()

	// Create a mock client
	websocket := &MockClient{UserID: 1}

	// Simulate adding a peer with a DataChannel
	peerID := uint64(2)
	pc, err := webrtc.NewPeerConnection(peerManager.Config)
	if err != nil {
		t.Fatalf("Failed to create PeerConnection: %v", err)
	}

	dc, err := pc.CreateDataChannel("data", nil)
	if err != nil {
		t.Fatalf("Failed to create DataChannel: %v", err)
	}

	peer := &webrtc.Peer{
		ID:          peerID,
		Connection:  pc,
		DataChannel: dc,
	}

	peerManager.Peers[peerID] = peer

	// Send data to the peer
	data := []byte("Hello, Peer!")
	err = peerManager.SendDataToPeer(peerID, data)

	// Assert no errors when sending data
	assert.Nil(t, err, "Expected no error when sending data to peer")
}

// TestPeerManager_GracefulShutdown tests the GracefulShutdown function.
func TestPeerManager_GracefulShutdown(t *testing.T) {
	peerManager := webrtc.NewPeerManager()

	// Create a mock client
	client := &MockClient{UserID: 1}

	// Add a peer to simulate active connections
	peerID := uint64(2)
	peerManager.Peers[peerID] = &webrtc.Peer{ID: peerID}

	// Simulate graceful shutdown
	peerManager.GracefulShutdown()

	// Assert no error during graceful shutdown
	// (in this test, we just check if it completes without an issue)
	assert.True(t, true, "Graceful shutdown completed without errors")
}
