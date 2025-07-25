package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	server "github.com/sushiag/go-webrtc-signaling-server/server/lib/server"
)

func TestAuthEndpoints(t *testing.T) {
	srv, serverURL := server.StartServer("0", server.NewWebSocketManager().Queries)
	defer srv.Close()

	baseURL := fmt.Sprintf("http://%s", serverURL)
	username := "user_e2e"
	password := "pass123!@#"

	t.Run("Register", func(t *testing.T) {
		body := map[string]string{
			"username": username,
			"password": password,
		}
		resp := post(t, baseURL+"/register", body)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("ResetPassword", func(t *testing.T) {
		body := map[string]string{
			"username":     username,
			"old_password": password,
			"new_password": "newpass456!@#",
		}
		resp := post(t, baseURL+"/newpassword", body)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("RegenerateAPIKey", func(t *testing.T) {
		body := map[string]string{
			"username": username,
			"password": "newpass456!@#",
		}
		resp := post(t, baseURL+"/regenerate", body)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
	t.Run("WebSocketConnectionWithAPIKey", func(t *testing.T) {
		// Get latest API key by regenerating
		body := map[string]string{
			"username": username,
			"password": "newpass456!@#",
		}
		resp := post(t, baseURL+"/regenerate", body)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var res struct {
			APIKey string `json:"api_key"`
		}
		err := json.NewDecoder(resp.Body).Decode(&res)
		require.NoError(t, err)
		require.NotEmpty(t, res.APIKey)

		// Connect via WebSocket using the API key in header
		wsURL := fmt.Sprintf("ws://%s/ws", serverURL)
		header := http.Header{}
		header.Set("Authorization", "Bearer "+res.APIKey)

		dialer := websocket.Dialer{}
		conn, _, err := dialer.Dial(wsURL, header)
		require.NoError(t, err)
		defer conn.Close()

	})

}

func post(t *testing.T, url string, data map[string]string) *http.Response {
	t.Helper()

	b, err := json.Marshal(data)
	require.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	require.NoError(t, err)

	return resp
}
