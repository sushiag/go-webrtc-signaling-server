package wsserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestFullE2EFlow(t *testing.T) {
	manager := NewWebSocketManager()
	apiKeys := map[string]bool{
		"test-api-key-1": true,
		"test-api-key-2": true,
	}
	manager.SetValidApiKeys(apiKeys)

	// test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth":
			manager.AuthHandler(w, r)
		case "/ws":
			manager.Handler(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// url websocket
	wsURL := "ws" + server.URL[4:] + "/ws"

	// step 1:  Authenticate clients
	type authResp struct {
		UserID     uint64 `json:"userid"`
		SessionKey string `json:"sessionkey"`
	}
	auth := func(apikey string) authResp {
		body, _ := json.Marshal(map[string]string{
			"apikey": apikey,
		})
		resp, err := http.Post(server.URL+"/auth", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var out authResp
		json.NewDecoder(resp.Body).Decode(&out)
		return out
	}

	client1 := auth("test-api-key-1")
	client2 := auth("test-api-key-2")

	// step 2: connect the clients
	dial := func(apikey string) *websocket.Conn {
		headers := http.Header{}
		headers.Set("X-Api-Key", apikey)

		ws, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
		require.NoError(t, err)
		return ws
	}

	ws1 := dial("test-api-key-1")
	defer ws1.Close()

	ws2 := dial("test-api-key-2")
	defer ws2.Close()

	// message for timeout
	readMsg := func(ws *websocket.Conn) Message {
		ws.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, data, err := ws.ReadMessage()
		require.NoError(t, err)

		var msg Message
		require.NoError(t, json.Unmarshal(data, &msg))
		return msg
	}

	// step 3.5: create room for client 1
	require.NoError(t, ws1.WriteJSON(Message{
		Type: TypeCreateRoom,
	}))

	roomCreated := readMsg(ws1)
	require.Equal(t, TypeRoomCreated, roomCreated.Type)
	roomID := roomCreated.RoomID
	require.NotZero(t, roomID)

	//  step 3.5: created room with invited, client 2 joins
	require.NoError(t, ws2.WriteJSON(Message{
		Type:   TypeJoin,
		RoomID: roomID,
	}))

	joinConfirm := readMsg(ws2)
	require.Equal(t, TypeCreateRoom, joinConfirm.Type)
	require.Equal(t, roomID, joinConfirm.RoomID)

	peerList := readMsg(ws2)
	require.Equal(t, TypePeerList, peerList.Type)
	require.Equal(t, roomID, peerList.RoomID)
	require.Len(t, peerList.Users, 1)
	require.Equal(t, client1.UserID, peerList.Users[0])

	// step 4: client 1 gets event of another peer joining
	peerJoined := readMsg(ws1)
	require.Equal(t, TypePeerJoined, peerJoined.Type)
	require.Equal(t, client2.UserID, peerJoined.Sender)

	// step 5: client 1 offers client 2
	offerSDP := "fake-offer-sdp"
	require.NoError(t, ws1.WriteJSON(Message{
		Type:   TypeOffer,
		RoomID: roomID,
		Sender: client1.UserID,
		Target: client2.UserID,
		SDP:    offerSDP,
	}))

	recvOffer := readMsg(ws2)
	require.Equal(t, TypeOffer, recvOffer.Type)
	require.Equal(t, offerSDP, recvOffer.SDP)

	// step 6: client 2 answers
	answerSDP := "fake-answer-sdp"
	require.NoError(t, ws2.WriteJSON(Message{
		Type:   TypeAnswer,
		RoomID: roomID,
		Sender: client2.UserID,
		Target: client1.UserID,
		SDP:    answerSDP,
	}))

	recvAnswer := readMsg(ws1)
	require.Equal(t, TypeAnswer, recvAnswer.Type)
	require.Equal(t, answerSDP, recvAnswer.SDP)

	// step 7: client 1 sends fake ice candidates
	candidate := "fake-candidate"
	require.NoError(t, ws1.WriteJSON(Message{
		Type:      TypeICE,
		RoomID:    roomID,
		Sender:    client1.UserID,
		Target:    client2.UserID,
		Candidate: candidate,
	}))

	recvICE := readMsg(ws2)
	require.Equal(t, TypeICE, recvICE.Type)
	require.Equal(t, candidate, recvICE.Candidate)

	// step 8: client 1 sends 'start'
	require.NoError(t, ws1.WriteJSON(Message{
		Type:   TypeStart,
		RoomID: roomID,
		Sender: client1.UserID,
	}))

	// both should disconnect
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errs := make(chan error, 2)

	go func() {
		_, _, err := ws1.ReadMessage()
		errs <- err
	}()

	go func() {
		_, _, err := ws2.ReadMessage()
		errs <- err
	}()

	select {
	case <-ctx.Done():
		t.Fatal("timeout waiting for disconnect")
	case err := <-errs:
		require.Error(t, err) // websocket closes
	}
}
