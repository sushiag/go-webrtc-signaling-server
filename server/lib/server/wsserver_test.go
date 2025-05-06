package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAuthHandler_ValidKey(t *testing.T) {
	manager := NewWebSocketManager()
	manager.SetValidApiKeys(map[string]bool{"valid-key": true})

	body := `{"apikey": "valid-key"}`
	req := httptest.NewRequest("POST", "/auth", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	manager.AuthHandler(w, req)
	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", resp.StatusCode)
	}

	var responseBody struct {
		UserID uint64 `json:"userid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		t.Fatal("failed to decode response:", err)
	}
	if responseBody.UserID == 0 {
		t.Error("expected non-zero userID")
	}
}

func TestAuthHandler_InvalidKey(t *testing.T) {
	manager := NewWebSocketManager()
	manager.SetValidApiKeys(map[string]bool{"valid-key": true})

	body := `{"apikey": "invalid-key"}`
	req := httptest.NewRequest("POST", "/auth", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	manager.AuthHandler(w, req)
	resp := w.Result()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

func TestCreateRoom(t *testing.T) {
	manager := NewWebSocketManager()
	fakeUserID := uint64(123)

	manager.Connections[fakeUserID] = nil // simulate user connection
	roomID := manager.CreateRoom(fakeUserID)

	if roomID == 0 {
		t.Fatal("expected non-zero roomID")
	}
	room := manager.Rooms[roomID]
	if room == nil {
		t.Fatal("room was not created")
	}
	if _, exists := room.Users[fakeUserID]; !exists {
		t.Errorf("user %d not added to room", fakeUserID)
	}
}

func TestAreInSameRoom(t *testing.T) {
	manager := NewWebSocketManager()
	user1 := uint64(1)
	user2 := uint64(2)

	manager.Connections[user1] = nil
	manager.Connections[user2] = nil

	roomID := manager.CreateRoom(user1)
	manager.AddUserToRoom(roomID, user2)

	if !manager.AreInSameRoom(roomID, []uint64{user1, user2}) {
		t.Error("expected users to be in the same room")
	}
}

func TestLoadValidApiKeys(t *testing.T) {
	file, err := os.CreateTemp("", "apikeys_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	_, _ = file.WriteString("testkey1\nkey2\n")
	_ = file.Close()

	keys, err := LoadValidApiKeys(file.Name())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	if !keys["testkey1"] || !keys["key2"] {
		t.Error("expected keys not found in result")
	}
}
