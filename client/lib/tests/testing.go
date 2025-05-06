package client_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	client "github.com/sushiag/go-webrtc-signaling-server/client/lib"
)

// ------------------------------
// Test NewClient
// ------------------------------
func TestNewClient(t *testing.T) {
	w := client.NewClient("ws://fake-endpoint")
	assert.NotNil(t, w)
	assert.NotNil(t, w.Client)
	assert.NotNil(t, w.PeerManager)
}

// ------------------------------
// Test SetApiKey
// ------------------------------
func TestSetApiKey(t *testing.T) {
	w := client.NewClient("ws://test")
	w.SetApiKey("test-key")
	assert.Equal(t, "test-key", w.Client.ApiKey)
}

// ------------------------------
// Test SetServerURL
// ------------------------------
func TestSetServerURL(t *testing.T) {
	w := client.NewClient("ws://initial")
	w.SetServerURL("ws://new-endpoint")
	assert.Equal(t, "ws://new-endpoint", w.Client.ServerURL)
}

// ------------------------------
// Test CreateRoom
// ------------------------------
func TestCreateRoom(t *testing.T) {
	w := client.NewClient("ws://test")
	// fake Create method
	w.Client.CreateRoom = func() error {
		return nil
	}

	err := w.CreateRoom()
	assert.NoError(t, err)
	assert.True(t, w.IsHost)
}

// ------------------------------
// Test JoinRoom
// ------------------------------
func TestJoinRoom(t *testing.T) {
	w := client.NewClient("ws://test")
	// fake JoinRoom method
	w.Client.JoinRoomFunc = func(id string) error {
		assert.Equal(t, "123", id)
		return nil
	}

	err := w.JoinRoom("123")
	assert.NoError(t, err)
	assert.False(t, w.IsHost)
}

// ------------------------------
// Test Connect
// ------------------------------
func TestConnect(t *testing.T) {
	w := client.NewClient("ws://test")

	// fake methods
	w.Client.PreAuthenticateFunc = func() error {
		return nil
	}
	w.Client.InitFunc = func() error {
		return nil
	}

	err := w.Connect()
	assert.NoError(t, err)
}

// ------------------------------
// Test Connect - Fails PreAuthenticate
// ------------------------------
func TestConnectPreAuthFail(t *testing.T) {
	w := client.NewClient("ws://test")
	w.Client.PreAuthenticateFunc = func() error {
		return errors.New("auth error")
	}
	err := w.Connect()
	assert.Error(t, err)
}

// ------------------------------
// Test StartSession
// ------------------------------
func TestStartSession(t *testing.T) {
	w := client.NewClient("ws://test")
	called := false
	w.Client.StartSessionFunc = func() error {
		called = true
		return nil
	}

	err := w.StartSession()
	assert.NoError(t, err)
	assert.True(t, called)
}

// ------------------------------
// Test SendMessageToPeer
// ------------------------------
func TestSendMessageToPeer(t *testing.T) {
	w := client.NewClient("ws://test")
	called := false

	w.PeerManager.SendDataToPeerFunc = func(id uint64, data []byte) error {
		called = true
		assert.Equal(t, uint64(1), id)
		assert.Equal(t, "hello", string(data))
		return nil
	}

	err := w.SendMessageToPeer(1, "hello")
	assert.NoError(t, err)
	assert.True(t, called)
}

// ------------------------------
// Test LeaveRoom
// ------------------------------
func TestLeaveRoom(t *testing.T) {
	w := client.NewClient("ws://test")
	called := false

	w.PeerManager.RemovePeerFunc = func(id uint64) {
		called = true
		assert.Equal(t, uint64(1), id)
	}

	w.LeaveRoom(1)
	assert.True(t, called)
}

// ------------------------------
// Test CloseServer as Host
// ------------------------------
func TestCloseServerAsHost(t *testing.T) {
	w := client.NewClient("ws://test")
	w.IsHost = true

	called := false
	w.PeerManager.CheckAllConnectedAndDisconnectFunc = func(c *client.WebSocketClient) {
		called = true
	}

	w.CloseServer()
	assert.True(t, called)
}

// ------------------------------
// Test CloseServer as Non-Host
// ------------------------------
func TestCloseServerAsNonHost(t *testing.T) {
	w := client.NewClient("ws://test")
	w.IsHost = false

	called := false
	w.PeerManager.CheckAllConnectedAndDisconnectFunc = func(c *client.WebSocketClient) {
		called = true
	}

	w.CloseServer()
	assert.False(t, called)
}

// ------------------------------
// Test Close
// ------------------------------
func TestClose(t *testing.T) {
	w := client.NewClient("ws://test")

	calledClient := false
	calledPM := false

	w.Client.CloseFunc = func() {
		calledClient = true
	}
	w.PeerManager.CloseAllFunc = func() {
		calledPM = true
	}

	w.Close()
	assert.True(t, calledClient)
	assert.True(t, calledPM)
}
