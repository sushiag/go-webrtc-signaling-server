package e2e_test

import (
	"fmt"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/sushiag/go-webrtc-signaling-server/client"
	server "github.com/sushiag/go-webrtc-signaling-server/server/lib/server"
)

func TestClientAuthFlow(t *testing.T) {
	srv, serverAddr := server.StartServer("0", server.NewWebSocketManager().Queries)
	defer srv.Close()

	baseURL := fmt.Sprintf("http://%s", serverAddr)
	username := "spongebob"
	initialPassword := "initialPass123"
	newPassword := "newSecurePass456"

	require.NoError(t, client.RegisterUser(baseURL, username, initialPassword))

	require.NoError(t, client.ResetPassword(baseURL, username, initialPassword, newPassword))

	apiKey, err := client.RegenerateAPIKey(baseURL, username, newPassword)
	require.NoError(t, err)
	require.NotEmpty(t, apiKey)

	require.NoError(t, os.Setenv("API_KEY", apiKey))

	c, err := client.NewClient(fmt.Sprintf("ws://%s/ws", serverAddr))
	require.NoError(t, err)

	roomID, err := c.CreateRoom()
	require.NoError(t, err)
	require.NotZero(t, roomID)
}
