package client

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"

	sm "signaling-msgs"
)

func TestSignalingRespondToPings(t *testing.T) {
	testApiKey := "ABC-123-DEF"

	expectedFlow := []mockMsgFlow{
		{
			waitFor: nil,
			toSend: &sm.Message{
				MsgType: sm.Ping,
			},
		},
		{
			waitFor: &sm.Message{
				MsgType: sm.Pong,
			},
			toSend: nil,
		},
	}
	wsEndpoint, serverDoneCh := startMockServer(t, testApiKey, expectedFlow)

	mngr, err := newSignalingManager(wsEndpoint, testApiKey, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), mngr.wsClientID)
	t.Log("signaling manager created")

	select {
	case <-time.After(time.Second * 3):
		{
			t.Error("server did not close in time")
		}
	case <-serverDoneCh:
		{
		}
	}
}

func TestSignalingCreateRoom(t *testing.T) {
	testApiKey := "ABC-123-DEF"

	expectedFlow := []mockMsgFlow{
		{
			waitFor: &sm.Message{
				MsgType: sm.CreateRoom,
			},
			toSend: &sm.Message{
				MsgType: sm.RoomCreated,
				Payload: sm.ToRawMessagePayload(sm.RoomCreatedPayload{
					RoomID: uint64(5),
				}),
			},
		},
	}
	wsEndpoint, serverDoneCh := startMockServer(t, testApiKey, expectedFlow)

	eventOutCh := make(chan Event)
	mngr, err := newSignalingManager(wsEndpoint, testApiKey, nil, eventOutCh)
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), mngr.wsClientID)
	t.Log("signaling manager created")

	mngr.wsSendCh <- sm.Message{MsgType: sm.CreateRoom}

	select {
	case <-time.After(time.Second):
		{
			t.Error("server did not respond in time")
		}
	case resp := <-eventOutCh:
		{
			assert.Equal(t,
				RoomCreatedEvent{RoomID: 5},
				resp,
			)
		}
	}

	select {
	case <-time.After(time.Second * 3):
		{
			t.Error("server did not close in time")
		}
	case <-serverDoneCh:
		{
		}
	}
}

func TestSignalingJoinRoom(t *testing.T) {
	testApiKey := "ABC-123-DEF"

	expectedFlow := []mockMsgFlow{
		{
			waitFor: &sm.Message{
				MsgType: sm.JoinRoom,
				Payload: sm.ToRawMessagePayload(sm.JoinRoomPayload{
					RoomID: uint64(5),
				}),
			},
			toSend: &sm.Message{
				MsgType: sm.RoomJoined,
				Payload: sm.ToRawMessagePayload(sm.RoomJoinedPayload{
					RoomID:        uint64(10),
					ClientsInRoom: []uint64{3, 6, 9},
				}),
			},
		},
	}
	wsEndpoint, serverDoneCh := startMockServer(t, testApiKey, expectedFlow)

	eventOutCh := make(chan Event, 8)
	mngr, err := newSignalingManager(wsEndpoint, testApiKey, nil, eventOutCh)
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), mngr.wsClientID)
	t.Log("signaling manager created")

	mngr.wsSendCh <- sm.Message{MsgType: sm.JoinRoom, Payload: sm.ToRawMessagePayload(sm.JoinRoomPayload{RoomID: 5})}

	select {
	case <-time.After(time.Second):
		{
			t.Error("server did not respond in time")
		}
	case resp := <-eventOutCh:
		{
			assert.Equal(t,
				RoomJoinedEvent{
					RoomID:        uint64(10),
					ClientsInRoom: []uint64{3, 6, 9},
				},
				resp,
			)
		}
	}

	select {
	case <-time.After(time.Second * 3):
		{
			t.Error("server did not close in time")
		}
	case <-serverDoneCh:
		{
		}
	}
}

type mockMsgFlow struct {
	waitFor *sm.Message
	toSend  *sm.Message
}

func startMockServer(t *testing.T, apiKey string, msgFlow []mockMsgFlow) (string, chan struct{}) {
	wsUpgrader := websocket.Upgrader{}
	doneCh := make(chan struct{}, 1)

	tcpListener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	})
	assert.NoError(t, err, "failed to start TCP listener")
	addr := tcpListener.Addr().String()
	t.Logf("TCP listener started on: %s", addr)

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		requestAPIKey := r.Header.Get("X-Api-Key")
		if requestAPIKey != apiKey {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Println(w, "unauthorized api key")
			return
		}

		conn, err := wsUpgrader.Upgrade(w, r, http.Header{"X-Client-ID": []string{"100"}})
		if err != nil {
			t.Fatalf("failed to upgrade HTTP connection to WS: %v", err)
		}

		go func() {
			for msgNum, step := range msgFlow {
				if step.waitFor == nil {
					writeErr := conn.WriteJSON(&step.toSend)
					if !assert.NoError(t, writeErr, "server failed to respond to the client") {
						break
					}
					continue
				}

				t.Logf("waiting for client msg: %d", msgNum)
				conn.SetReadDeadline(time.Now().Add(time.Second))

				var msg sm.Message
				readErr := conn.ReadJSON(&msg)
				if !assert.NoErrorf(t, readErr, "the server failed to read WS message from client") {
					break
				}
				t.Logf("got message with type '%d' from the client", msg.MsgType)

				if !assert.Equal(t, step.waitFor.MsgType, msg.MsgType, "the server received an unexpected message from the client") {
					break
				}

				if step.toSend != nil {
					writeErr := conn.WriteJSON(&step.toSend)
					if !assert.NoError(t, writeErr, "server failed to respond to the client") {
						break
					}
				}
			}

			doneCh <- struct{}{}
		}()
	})

	go func() {
		t.Log("starting HTTP server")
		http.Serve(tcpListener, httpMux)
	}()

	wsEndpoint := fmt.Sprintf("ws://%s/ws", addr)
	return wsEndpoint, doneCh
}
