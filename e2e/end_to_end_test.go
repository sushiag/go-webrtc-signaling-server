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

// getEnv retrieves environment variables with a fallback to the default value if not set
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// TestClientToClientSignaling performs end-to-end signaling test between two clients
func TestClientToClientSignaling(t *testing.T) {
	// Manually set API key and port for each client
	client1 := clienthandle.NewClient()
	client2 := clienthandle.NewClient()

	// Set the API key and port
	client1.ApiKey = "valid-api-key-1"
	client1.ServerURL = fmt.Sprintf("ws://localhost:%s/ws", getEnv("WS_PORT", "8080"))

	client2.ApiKey = "valid-api-key-2"
	client2.ServerURL = fmt.Sprintf("ws://localhost:%s/ws", getEnv("WS_PORT", "8080"))

	// Perform authentication for both clients
	err := client1.PreAuthenticate()
	if err != nil {
		t.Fatalf("client1 auth failed: %v", err)
	}
	err = client2.PreAuthenticate()
	if err != nil {
		t.Fatalf("client2 auth failed: %v", err)
	}

	err = client1.Init()
	if err != nil {
		t.Fatalf("client1 init failed: %v", err)
	}
	defer client1.Close()

	err = client2.Init()
	if err != nil {
		t.Fatalf("client2 init failed: %v", err)
	}
	defer client2.Close()

	// Initialize the WebRTC handlers or PeerManager for both clients
	pm2 := webrtchandle.NewPeerManager()

	done := make(chan bool)

	// Handle messages for client1
	client1.SetMessageHandler(func(msg clienthandle.Message) {
		if msg.Type == clienthandle.MessageTypeRoomCreated {
			// Log the RoomID for debugging
			fmt.Printf("Received RoomID: %v\n", msg.RoomID)

			// Convert RoomID to string before passing it to client2.Join
			go func() {
				roomIDStr := strconv.FormatUint(msg.RoomID, 10)
				// Add more logging to check if client2 is joining
				fmt.Printf("Client2 joining room with ID: %s\n", roomIDStr)
				err := client2.Join(roomIDStr) // pass roomID as string
				if err != nil {
					t.Errorf("client2 failed to join: %v", err)
				}
			}()
		}

		if msg.Type == clienthandle.MessageTypePeerJoined {
			go func() {
				startMsg := clienthandle.Message{
					Type:   clienthandle.MessageTypeStart,
					RoomID: client1.RoomID,
					Sender: client1.UserID,
				}
				_ = client1.Send(startMsg)
			}()
		}

		// Handle ICE candidates and pass them to the other client
		if msg.Type == clienthandle.MessageTypeICECandidate {
			go func() {
				// Send the ICE candidate to the other client
				err := client2.Send(msg)
				if err != nil {
					t.Errorf("client2 failed to send ICE candidate: %v", err)
				}
			}()
		}
	})

	// Handle messages for client2
	client2.SetMessageHandler(func(msg clienthandle.Message) {
		if msg.Type == clienthandle.MessageTypeStart {
			t.Logf("Received 'start' signal, test complete")
			done <- true
		}

		// Handle ICE candidates and add them to the peer connection
		if msg.Type == clienthandle.MessageTypeICECandidate {
			go func() {
				// Use the WebRTC handler (or PeerManager) to add the ICE candidate for client2's peer connection
				err := pm2.HandleICECandidate(msg) // Update this based on your WebRTC package
				if err != nil {
					t.Errorf("client2 failed to add ICE candidate: %v", err)
				}
			}()
		}
	})

	// Start the room creation by client1
	err = client1.Start()
	if err != nil {
		t.Fatalf("client1 failed to create room: %v", err)
	}

	select {
	case <-done:
		t.Log("E2E signaling test passed.")
	case <-time.After(10 * time.Second):
		t.Error("Timeout: signaling between clients did not complete")
	}
}
