package client_test

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestClientEndToEnd(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(2)

	client1 := clienthandler.NewClient()
	client2 := clienthandler.NewClient()

	err := client1.PreAuthenticate()
	if err != nil {
		t.Fatalf("Client1 auth failed: %v", err)
	}

	err = client2.PreAuthenticate()
	if err != nil {
		t.Fatalf("Client2 auth failed: %v", err)
	}

	err = client1.Init()
	if err != nil {
		t.Fatalf("Client1 init failed: %v", err)
	}
	defer client1.Close()

	err = client2.Init()
	if err != nil {
		t.Fatalf("Client2 init failed: %v", err)
	}
	defer client2.Close()

	// Client 1 will create the room
	client1.SetMessageHandler(func(msg clienthandler.Message) {
		if msg.Type == clienthandler.MessageTypeRoomCreated {
			t.Logf("Room created by client1: %d", msg.RoomID)
			client2.SetMessageHandler(func(msg2 clienthandler.Message) {
				if msg2.Type == clienthandler.MessageTypeRoomJoined {
					t.Logf("Client2 joined room %d", msg2.RoomID)

					// Client1 sends "start" to initiate P2P
					err := client1.Send(clienthandler.Message{
						Type:   clienthandler.MessageTypeStart,
						RoomID: msg.RoomID,
						Sender: client1.UserID,
					})
					if err != nil {
						t.Errorf("Client1 failed to send start: %v", err)
					}
					wg.Done()
				}
			})

			if err := client2.Join((formatUint(msg.RoomID))); err != nil {
				t.Errorf("Client2 failed to join: %v", err)
			}
		}
	})

	err = client1.Start()
	if err != nil {
		t.Fatalf("Client1 failed to create room: %v", err)
	}

	// Give time for signaling to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("End-to-end test completed successfully")
	case <-time.After(15 * time.Second):
		t.Fatal("Test timed out")
	}
}

func formatUint(i uint64) string {
	return strconv.FormatUint(i, 10)
}
