package clientwrapper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sushiag/go-webrtc-signaling-server/client/clienthandle"
	"github.com/sushiag/go-webrtc-signaling-server/client/clientwrapper"
)

func TestEndToEndClientWrapper(t *testing.T) {
	msgCh := make(chan string, 1)

	// Initialize host
	host, err := clientwrapper.New()
	assert.NoError(t, err)
	defer host.Close()

	// Create room
	err = host.Create()
	assert.NoError(t, err)
	roomID := host.RoomID()
	assert.NotZero(t, roomID)

	// Initialize peer
	peer, err := clientwrapper.New()
	assert.NoError(t, err)
	defer peer.Close()

	// Set message handler to capture incoming message
	peer.SetMessageHandler(func(msg clienthandle.Message) {
		if msg.Type == clienthandle.MessageTypeSendMessage {
			msgCh <- msg.Data
		}
	})

	// Peer joins host's room
	err = peer.Join(roomID)
	assert.NoError(t, err)

	// Start session
	assert.NoError(t, host.Start())
	assert.NoError(t, peer.Start())

	// Let P2P connection establish
	time.Sleep(1 * time.Second)

	// Host sends message to peer
	err = host.Send(peer.UserID(), "hello peer")
	assert.NoError(t, err)

	// Check if peer received it
	select {
	case msg := <-msgCh:
		assert.Equal(t, "hello peer", msg)
	case <-time.After(3 * time.Second):
		t.Fatal("Timed out waiting for peer to receive message")
	}
}
