package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	wsserver "server/wsserver"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func writeTempAPIKeyFile(t *testing.T, keys []string) string {
	tmpFile, err := ioutil.TempFile("", "apikeys.txt")
	assert.NoError(t, err)

	for _, key := range keys {
		_, err := tmpFile.WriteString(key + "\n")
		assert.NoError(t, err)
	}

	err = tmpFile.Close()
	assert.NoError(t, err)

	return tmpFile.Name()
}

// load the txt file of api keys: passed test
func TestLoadValidApiKeys(t *testing.T) {
	filePath := writeTempAPIKeyFile(t, []string{"valid-key-1", "valid-key-2"})
	defer os.Remove(filePath)

	keys, err := LoadValidApiKeys(filePath)
	assert.NoError(t, err)
	assert.True(t, keys["valid-key-1"])
	assert.True(t, keys["valid-key-2"])
	assert.False(t, keys["invalid-key"])
}

// authhanlder :passed test
func TestAuthHandler(t *testing.T) {
	wsManager := wsserver.NewWebSocketManager()
	wsManager.SetValidApiKeys(map[string]bool{"test-key": true})

	server := httptest.NewServer(http.HandlerFunc(wsManager.AuthHandler))
	defer server.Close()

	payload := map[string]string{"apikey": "test-key"}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(server.URL, "application/json", bytes.NewReader(body))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()
	var response struct {
		UserID     uint64 `json:"userid"`
		SessionKey string `json:"sessionkey"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.NotZero(t, response.UserID)
	assert.NotEmpty(t, response.SessionKey)
}

// test with valid test: passed test
func TestWebSocketHandler_WithValidApiKey(t *testing.T) {
	wsManager := wsserver.NewWebSocketManager()
	wsManager.SetValidApiKeys(map[string]bool{"valid-key": true})
	server := httptest.NewServer(http.HandlerFunc(wsManager.Handler))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]

	header := http.Header{}
	header.Set("X-Api-Key", "valid-key")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	assert.NoError(t, err)
	conn.Close()
}

// test with invalid apikey: passed test
func TestWebSocketHandler_WithInvalidApiKey(t *testing.T) {
	wsManager := wsserver.NewWebSocketManager()
	wsManager.SetValidApiKeys(map[string]bool{"valid-key": true})

	server := httptest.NewServer(http.HandlerFunc(wsManager.Handler))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]

	header := http.Header{}
	header.Set("X-Api-Key", "invalid-key")

	_, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	assert.Error(t, err)
}
