package e2e_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	client "github.com/sushiag/go-webrtc-signaling-server/client/lib"
	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
	server "github.com/sushiag/go-webrtc-signaling-server/server/lib/server"
)

func TestEndToEndSignaling(t *testing.T) {
	srv, serverURL := server.StartServer("0")
	defer srv.Close() // ensure server is closed after the test

	apiKeyA, apiKeyB := "valid-api-key-1", "valid-api-key-2"

	clientA := client.NewClient(serverURL)
	clientB := client.NewClient(serverURL)

	clientA.Websocket.ApiKey = apiKeyA
	clientB.Websocket.ApiKey = apiKeyB

	err := clientA.Connect()
	assert.NoError(t, err, "Client A failed to connect")
	t.Logf("[TEST] Client A connected")

	err = clientB.Connect()
	assert.NoError(t, err, "Client B failed to connect")
	t.Logf("[TEST] Client B connected")

	err = clientA.CreateRoom()
	assert.NoError(t, err, "Client A: failed to create room")
	t.Logf("[TEST] Client A Created Room")

	roomID := strconv.FormatUint(clientA.Websocket.RoomID, 10)

	err = clientB.JoinRoom(roomID)
	assert.NoError(t, err, "Client B failed to join room")
	t.Logf("[TEST] Client B Joined Room")

	clientAMsg := "Hello from Client A!"
	var clientAPeers []uint64
	var clientBPeers []uint64

	clientBMsg := "Wassup from Client B!"

	t.Logf("---- Waiting for data channels to open ----")
	readyDataChannels := 0
	dataChDeadline := time.After(5 * time.Second)
	for readyDataChannels < 2 {
		select {
		case ev := <-clientA.PeerManager.PeerEventsCh:
			{
				clientAPeers = clientA.PeerManager.GetPeerIDs()
				assert.Equal(t, clientAPeers, []uint64{1}, "wrong client A peers")
				t.Logf("Client A peers: %v", clientAPeers)
				assert.Equal(t, ev, common.PeerDataChOpened{PeerID: clientAPeers[0]})
				readyDataChannels += 1
			}

		case ev := <-clientB.PeerManager.PeerEventsCh:
			{
				clientBPeers = clientB.PeerManager.GetPeerIDs()
				assert.Equal(t, clientBPeers, []uint64{0}, "wrong client B peers")
				t.Logf("Client B peers: %v", clientBPeers)
				assert.Equal(t, ev, common.PeerDataChOpened{PeerID: clientBPeers[0]})
				readyDataChannels += 1
			}

		case <-dataChDeadline:
			{
				t.Fatal("clients took longer than 5 secs to open their data channels")
			}
		}
	}
	t.Logf("---- Data channels open! ----")

	t.Logf("---- Sending Messages Start ----")
	err = clientA.SendMessageToPeer(clientAPeers[0], clientAMsg)
	assert.NoErrorf(t, err, "Failed to send message from client A to peer %d", clientAPeers[0])

	err = clientB.SendMessageToPeer(clientBPeers[0], clientBMsg)
	assert.NoErrorf(t, err, "Failed to send message from client B to peer %d", clientBPeers[0])
	t.Logf("---- Sending Messages End ----")

	receivedMsgs := 0
	recvMsgDeadline := time.After(5 * time.Second)
	for receivedMsgs < 2 {
		select {
		case msg := <-clientA.MsgOutCh:
			{
				assert.Equal(t, msg.From, clientB.Websocket.UserID)
				assert.Equal(t, msg.Data, []byte(clientBMsg))
				receivedMsgs += 1
			}

		case msg := <-clientB.MsgOutCh:
			{
				assert.Equal(t, msg.From, clientA.Websocket.UserID)
				assert.Equal(t, msg.Data, []byte(clientAMsg))
				receivedMsgs += 1
			}

		case <-recvMsgDeadline:
			{
				t.Fatal("clients took longer than 5 secs to exchange messages")
			}
		}
	}

	t.Logf("All clients successfully exchanged messages for 1 round.")
}
