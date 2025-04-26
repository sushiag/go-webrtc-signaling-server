package e2e_test

import (
	"testing"
	"time"

	clienthandle "github.com/sushiag/go-webrtc-signaling-server/client/clienthandler"
)

func TestClientToClientSignaling(t *testing.T) {
	client1 := clienthandle.NewClient()
	client2 := clienthandle.NewClient()

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

	done := make(chan bool)

	client1.SetMessageHandler(func(msg clienthandle.Message) {
		if msg.Type == clienthandle.MessageTypeRoomCreated {
			go func() {
				err := client2.Join((string)(msg.RoomID))
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
	})

	client2.SetMessageHandler(func(msg clienthandle.Message) {
		if msg.Type == clienthandle.MessageTypeStart {
			t.Logf("Received 'start' signal, test complete")
			done <- true
		}
	})

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
