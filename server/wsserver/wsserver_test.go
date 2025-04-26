package wsserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// test to assin userid and sessionkey: passed test
func TestAssignUserIDAndSessionKey(t *testing.T) {
	wm := NewWebSocketManager()
	validKey := "test-key-123"
	wm.SetValidApiKeys(map[string]bool{
		validKey: true,
	})

	// 1st request
	payload := map[string]string{"apikey": validKey}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	wm.AuthHandler(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", res.StatusCode)
	}

	var response struct {
		UserID     uint64 `json:"userid"`
		SessionKey string `json:"sessionkey"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.UserID == 0 {
		t.Errorf("Expected non-zero userID, got %d", response.UserID)
	}
	if response.SessionKey == "" {
		t.Error("Expected non-empty session key")
	}

	// second request with the same apikey
	req2 := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewReader(body))
	rec2 := httptest.NewRecorder()
	wm.AuthHandler(rec2, req2)

	res2 := rec2.Result()
	defer res2.Body.Close()

	var response2 struct {
		UserID     uint64 `json:"userid"`
		SessionKey string `json:"sessionkey"`
	}
	if err := json.NewDecoder(res2.Body).Decode(&response2); err != nil {
		t.Fatalf("Failed to decode second response: %v", err)
	}

	if response.UserID != response2.UserID {
		t.Errorf("Expected same userID for repeated API key, got %d and %d", response.UserID, response2.UserID)
	}
}

// test to see if the assigned user id creates room: passed test
func TestAssignUserIDAndCreateRoom(t *testing.T) {
	manager := NewWebSocketManager()
	apiKey := "test-api-key"
	manager.SetValidApiKeys(map[string]bool{apiKey: true})

	authSrv := httptest.NewServer(http.HandlerFunc(manager.AuthHandler))
	defer authSrv.Close()

	wsSrv := httptest.NewServer(http.HandlerFunc(manager.Handler))
	defer wsSrv.Close()

	// auth to get assigned userid and sessionkey
	authBody := map[string]string{"apikey": apiKey}
	bodyBytes, _ := json.Marshal(authBody)

	resp, err := http.Post(authSrv.URL, "application/json", bytes.NewBuffer(bodyBytes))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var authResp struct {
		UserID     uint64 `json:"userid"`
		SessionKey string `json:"sessionkey"`
	}
	json.NewDecoder(resp.Body).Decode(&authResp)
	resp.Body.Close()

	// connect to websocket
	wsURL := "ws" + wsSrv.URL[len("http"):]

	headers := http.Header{}
	headers.Set("X-Api-Key", apiKey)

	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	require.NoError(t, err)
	defer wsConn.Close()

	// sends create room message
	err = wsConn.WriteJSON(Message{
		Type:   TypeCreateRoom,
		Sender: authResp.UserID,
	})
	require.NoError(t, err)

	// waits
	wsConn.SetReadDeadline(time.Now().Add(5 * time.Second))

	var roomCreatedMsg *Message
	for i := 0; i < 2; i++ {
		var msg Message
		err := wsConn.ReadJSON(&msg)
		require.NoError(t, err)

		if msg.Type == TypeRoomCreated {
			roomCreatedMsg = &msg
			break
		}
	}

	require.NotNil(t, roomCreatedMsg, "expected room-created message not received")
	require.Equal(t, TypeRoomCreated, roomCreatedMsg.Type)
	require.Equal(t, authResp.UserID, roomCreatedMsg.Sender)
	require.True(t, roomCreatedMsg.RoomID > 0)
}

// returns 401 if invalid key: passed test
func TestInvalidApiKeyReturns401(t *testing.T) {
	wm := NewWebSocketManager()
	wm.SetValidApiKeys(map[string]bool{
		"valid-key": true,
	})

	payload := map[string]string{"apikey": "invalid-key"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/auth", bytes.NewReader(body))
	w := httptest.NewRecorder()

	wm.AuthHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

// creates a new unique room ID: passed test
func TestCreateRoomCreatesUniqueRoomID(t *testing.T) {
	wm := NewWebSocketManager()
	userID := uint64(42)
	wm.Connections[userID] = nil // fake connection for testing

	roomID1 := wm.CreateRoom(userID)
	roomID2 := wm.CreateRoom(userID)

	if roomID1 == roomID2 {
		t.Errorf("Expected unique room IDs, got same ID: %d", roomID1)
	}
	if _, ok := wm.Rooms[roomID1]; !ok {
		t.Errorf("Room ID %d not created", roomID1)
	}
	if _, ok := wm.Rooms[roomID2]; !ok {
		t.Errorf("Room ID %d not created", roomID2)
	}
}

// returns false if any user is missing: passed test.
func TestAreInSameRoomReturnsFalseIfAnyMissing(t *testing.T) {
	wm := NewWebSocketManager()
	roomID := uint64(1)
	userA := uint64(101)
	userB := uint64(102)

	// Add only one user
	wm.Rooms[roomID] = &Room{
		ID:    roomID,
		Users: map[uint64]*websocket.Conn{userA: nil},
	}

	result := wm.AreInSameRoom(roomID, []uint64{userA, userB})
	if result {
		t.Error("Expected false, only one user is in the room")
	}
}

func TestPeerToPeerDisconnectCleanup(t *testing.T) {
	manager := NewWebSocketManager()
	apiKeys := map[string]bool{
		"peer1": true,
		"peer2": true,
	}
	manager.SetValidApiKeys(apiKeys)

	// Start WebSocket server
	wsSrv := httptest.NewServer(http.HandlerFunc(manager.Handler))
	defer wsSrv.Close()
	wsURL := "ws" + wsSrv.URL[len("http"):]

	// Connect Peer1
	headers1 := http.Header{}
	headers1.Set("X-Api-Key", "peer1")
	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, headers1)
	require.NoError(t, err)
	defer ws1.Close()

	// Create room
	err = ws1.WriteJSON(Message{
		Type:   TypeCreateRoom,
		Sender: 1, // assuming assigned ID will be 1
	})
	require.NoError(t, err)

	// Read room-created
	var msg Message
	err = ws1.ReadJSON(&msg)
	require.NoError(t, err)
	require.Equal(t, TypeRoomCreated, msg.Type)
	roomID := msg.RoomID

	// Connect Peer2
	headers2 := http.Header{}
	headers2.Set("X-Api-Key", "peer2")
	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, headers2)
	require.NoError(t, err)
	defer ws2.Close()

	// Peer2 joins room
	err = ws2.WriteJSON(Message{
		Type:   TypeJoin,
		Sender: 2,
		RoomID: roomID,
	})
	require.NoError(t, err)

	// Expect room-joined message (from server to Peer1)
	// Expect some P2P messages if you mock SDP exchange
	// (skip actual WebRTC for now — just mimic signaling)

	// Now simulate disconnect from Peer2
	err = ws2.Close()
	require.NoError(t, err)

	// Allow time for server to detect disconnect
	time.Sleep(1 * time.Second)

	// Check that Peer2 was removed from room
	room := manager.Rooms[roomID]
	require.NotNil(t, room)
	_, exists := room.Users[2]
	require.False(t, exists, "Expected Peer2 to be removed from room")

	// Optionally check Peer1 received a peer-disconnected message
	var notify Message
	err = ws1.ReadJSON(&notify)
	require.NoError(t, err)
	require.Equal(t, "peer-disconnected", notify.Type)
	require.Equal(t, uint64(2), notify.Sender)
}
