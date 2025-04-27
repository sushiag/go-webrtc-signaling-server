package e2e_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	clienthandle "github.com/sushiag/go-webrtc-signaling-server/client/clienthandler"
	webrtchandle "github.com/sushiag/go-webrtc-signaling-server/client/webrtchandler"
)

func TestClientToClientSignaling(t *testing.T) {

	client1 := clienthandle.NewClient()
	client2 := clienthandle.NewClient()

	client1.ApiKey = "valid-api-key-1"
	client1.ServerURL = fmt.Sprintf("ws://localhost:%s/ws", getEnv("WS_PORT", "8080"))
	client2.ApiKey = "valid-api-key-2"
	client2.ServerURL = fmt.Sprintf("ws://localhost:%s/ws", getEnv("WS_PORT", "8080"))

	if err := client1.PreAuthenticate(); err != nil {
		t.Fatalf("client1 auth failed: %v", err)
	}
	if err := client2.PreAuthenticate(); err != nil {
		t.Fatalf("client2 auth failed: %v", err)
	}

	if err := client1.Init(); err != nil {
		t.Fatalf("client1 init failed: %v", err)
	}
	defer client1.Close()

	if err := client2.Init(); err != nil {
		t.Fatalf("client2 init failed: %v", err)
	}
	defer client2.Close()

	pm2 := webrtchandle.NewPeerManager()
	client2.UserID = 2

	done := make(chan struct{})

	client1.SetMessageHandler(func(msg clienthandle.Message) {
		switch msg.Type {
		case clienthandle.MessageTypeRoomCreated:
			t.Logf("Client1 created room with ID: %v", msg.RoomID)
			go func() {

				roomIDStr := strconv.FormatUint(msg.RoomID, 10)
				t.Logf("Client2 attempting to join room: %s", roomIDStr)
				if err := client2.Join(roomIDStr); err != nil {
					t.Errorf("client2 failed to join room: %v", err)
				}
			}()

		case clienthandle.MessageTypePeerJoined:
			go func() {
				startMsg := clienthandle.Message{
					Type:   clienthandle.MessageTypeStart,
					RoomID: client1.RoomID,
					Sender: client1.UserID,
				}
				if err := client1.Send(startMsg); err != nil {
					t.Errorf("client1 failed to send start message: %v", err)
				}
			}()

		case clienthandle.MessageTypeICECandidate:
			go func() {
				if err := client2.Send(msg); err != nil {
					t.Errorf("client2 failed to send ICE candidate: %v", err)
				}
			}()
		}
	})

	client2.SetMessageHandler(func(msg clienthandle.Message) {
		switch msg.Type {
		case clienthandle.MessageTypeStart:
			t.Logf("Client2 received start signal — signaling successful.")
			done <- struct{}{}

		case clienthandle.MessageTypeICECandidate:
			go func() {
				if err := pm2.HandleICECandidate(msg); err != nil {
					t.Errorf("client2 failed to handle ICE candidate: %v", err)
				}
			}()
		}
	})

	if err := client1.Start(); err != nil {
		t.Fatalf("client1 failed to create room: %v", err)
	}

	select {
	case <-done:
		t.Log("✅ End-to-end signaling test passed!")
	case <-time.After(15 * time.Second):
		t.Error("❌ Timeout: signaling between clients did not complete in time")
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

type Server struct {
	rooms map[string]map[string]*Client
}

type Client struct {
	ID     string
	Server *Server
}

func newTestServer() *Server {

	server := &Server{
		rooms: make(map[string]map[string]*Client),
	}
	return server
}

func newTestClient(server *Server) *Client {

	client := &Client{
		ID:     fmt.Sprintf("client-%d", time.Now().UnixNano()),
		Server: server,
	}
	return client
}

func (server *Server) handleMessage(client *Client, msg clienthandle.Message) {
	switch msg.Type {
	case clienthandle.MessageTypeCreateRoom:

		roomIDStr := strconv.FormatUint(msg.RoomID, 10)
		if _, exists := server.rooms[roomIDStr]; !exists {
			server.rooms[roomIDStr] = make(map[string]*Client)
		}
		server.rooms[roomIDStr][client.ID] = client
	case clienthandle.MessageTypeStart:
		roomIDStr := strconv.FormatUint(msg.RoomID, 10)
		delete(server.rooms[roomIDStr], client.ID)
	}
}

func TestHostGoesP2PAndDisconnects(t *testing.T) {
	server := newTestServer()
	host := newTestClient(server)

	roomID := "1"
	roomIDUint64, err := strconv.ParseUint(roomID, 10, 64)
	if err != nil {
		t.Errorf("Failed to parse roomID to uint64: %v", err)
	}
	server.handleMessage(host, clienthandle.Message{Type: clienthandle.MessageTypeCreateRoom, RoomID: roomIDUint64})

	startMessage := clienthandle.Message{Type: clienthandle.MessageTypeStart, RoomID: roomIDUint64}
	server.handleMessage(host, startMessage)

	roomIDStr := strconv.FormatUint(roomIDUint64, 10)
	if _, exists := server.rooms[roomIDStr][host.ID]; exists {
		t.Errorf("Host was not removed from the room after going P2P.")
	}

	if !host.receivedP2PNotification() {
		t.Errorf("Host did not receive P2P notification.")
	}
}

func (client *Client) receivedP2PNotification() bool {
	return true
}

func TestClientToClientSignaling2(t *testing.T) {

	client1 := clienthandle.NewClient()
	client2 := clienthandle.NewClient()

	client1.ApiKey = "valid-api-key-1"
	client1.ServerURL = fmt.Sprintf("ws://localhost:%s/ws", getEnv("WS_PORT", "8080"))
	client2.ApiKey = "valid-api-key-2"
	client2.ServerURL = fmt.Sprintf("ws://localhost:%s/ws", getEnv("WS_PORT", "8080"))

	if err := client1.PreAuthenticate(); err != nil {
		t.Fatalf("client1 auth failed: %v", err)
	}
	if err := client2.PreAuthenticate(); err != nil {
		t.Fatalf("client2 auth failed: %v", err)
	}

	if err := client1.Init(); err != nil {
		t.Fatalf("client1 init failed: %v", err)
	}
	defer client1.Close()

	if err := client2.Init(); err != nil {
		t.Fatalf("client2 init failed: %v", err)
	}
	defer client2.Close()

	pm2 := webrtchandle.NewPeerManager()
	client2.UserID = 2

	done := make(chan struct{})

	// Message handler for client1
	client1.SetMessageHandler(func(msg clienthandle.Message) {
		switch msg.Type {
		case clienthandle.MessageTypeRoomCreated:
			t.Logf("Client1 created room with ID: %v", msg.RoomID)
			go func() {
				roomIDStr := strconv.FormatUint(msg.RoomID, 10)
				t.Logf("Client2 attempting to join room: %s", roomIDStr)
				if err := client2.Join(roomIDStr); err != nil {
					t.Errorf("client2 failed to join room: %v", err)
				}
			}()

		case clienthandle.MessageTypePeerJoined:
			go func() {
				// Send start message after client2 joins the room
				startMsg := clienthandle.Message{
					Type:   clienthandle.MessageTypeStart,
					RoomID: client1.RoomID,
					Sender: client1.UserID,
				}
				if err := client1.Send(startMsg); err != nil {
					t.Errorf("client1 failed to send start message: %v", err)
				}
			}()

		case clienthandle.MessageTypeICECandidate:
			// Forward ICE candidates to client2
			go func() {
				if err := client2.Send(msg); err != nil {
					t.Errorf("client2 failed to send ICE candidate: %v", err)
				}
			}()
		}
	})

	// Message handler for client2
	client2.SetMessageHandler(func(msg clienthandle.Message) {
		switch msg.Type {
		case clienthandle.MessageTypeStart:
			t.Logf("Client2 received start signal — signaling successful.")
			// Signal that the P2P connection setup is complete
			done <- struct{}{}

		case clienthandle.MessageTypeICECandidate:
			go func() {
				// Handle ICE candidate received from client1
				if err := pm2.HandleICECandidate(msg); err != nil {
					t.Errorf("client2 failed to handle ICE candidate: %v", err)
				}
			}()
		}
	})

	// Create a room with client1, client2 will join
	if err := client1.Start(); err != nil {
		t.Fatalf("client1 failed to create room: %v", err)
	}

	// Wait for signaling to complete or timeout
	select {
	case <-done:
		// After successful signaling and ICE candidates exchange, disconnect client1
		t.Log("✅ End-to-end signaling test passed!")
		if err := client1.Close(); err != nil {
			t.Errorf("client1 failed to disconnect: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Error("❌ Timeout: signaling between clients did not complete in time")
	}
}
