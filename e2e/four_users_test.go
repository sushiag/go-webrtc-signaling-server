package e2e_test

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	client "github.com/sushiag/go-webrtc-signaling-server/client/lib"
	server "github.com/sushiag/go-webrtc-signaling-server/server/lib/server"
)

func TestEndToEndSignalingFourUsers(t *testing.T) {
	server, serverUrl := server.StartServer("0")
	defer server.Close()

	apiKeyA, apiKeyB, apiKeyC, apiKeyD := "valid-api-key-1", "valid-api-key-2", "valid-api-key-3", "valid-api-key-4"

	clientA := client.NewClient(serverUrl)
	clientB := client.NewClient(serverUrl)
	clientC := client.NewClient(serverUrl)
	clientD := client.NewClient(serverUrl)

	clientA.Websocket.ApiKey = apiKeyA
	clientB.Websocket.ApiKey = apiKeyB
	clientC.Websocket.ApiKey = apiKeyC
	clientD.Websocket.ApiKey = apiKeyD

	clients := []*client.Client{clientA, clientB, clientC, clientD}

	for i, c := range clients {
		err := c.Connect()
		assert.NoError(t, err, "Client %d failed to connect", i)
	}

	err := clientA.CreateRoom()
	assert.NoError(t, err, "Client A failed to create room")

	roomID := strconv.FormatUint(clientA.Websocket.RoomID, 10)

	err = clientB.JoinRoom(roomID)
	assert.NoError(t, err, "Client B failed to join room")

	err = clientC.JoinRoom(roomID)
	assert.NoError(t, err, "Client C failed to join room")

	err = clientD.JoinRoom(roomID)
	assert.NoError(t, err, "Client D failed to join room")

	t.Logf("---- Waiting for data channels to open ----")
	readyDataChannels := 0
	totalPeers := 4
	totalDataChannels := (totalPeers - 1) * totalPeers
	dataChDeadline := time.After(3 * time.Second)

	for readyDataChannels < totalDataChannels {
		select {
		case <-clientA.PeerManager.PeerEventsCh:
			readyDataChannels += 1
		case <-clientB.PeerManager.PeerEventsCh:
			readyDataChannels += 1
		case <-clientC.PeerManager.PeerEventsCh:
			readyDataChannels += 1
		case <-clientD.PeerManager.PeerEventsCh:
			readyDataChannels += 1

		case <-dataChDeadline:
			{
				t.Fatalf("clients took longer than 3 secs to open their data channels, only opened: %d", readyDataChannels)
			}
		}
	}
	t.Logf("---- Data channels open! ----")

	rounds := 1
	msgsSent := atomic.Uint32{}
	msgsReceived := atomic.Uint32{}
	msgToSendPerUser := totalPeers - 1
	totalMsgsToSend := uint32(totalPeers * msgToSendPerUser * rounds)
	msgsToReceive := uint32(totalMsgsToSend)
	t.Logf("total messages to send: %d", totalMsgsToSend)
	wg := sync.WaitGroup{}

	for round := 1; round <= 1; round++ {
		t.Logf("---- Round %d ----", round)

		for _, sender := range clients {
			wg.Add(1)

			go func() {
				senderID := strconv.FormatUint(sender.Websocket.UserID, 10)

				peerIDs := sender.PeerManager.GetPeerIDs()
				for _, peerID := range peerIDs {
					receiverID := strconv.FormatUint(peerID, 10)
					message := "Round " + strconv.Itoa(round) +
						" | from client " + senderID +
						" | to client " + receiverID

					t.Logf("%d sending message to %d", sender.Websocket.UserID, peerID)
					err := sender.SendMessageToPeer(peerID, message)
					assert.NoErrorf(t, err, "failed to send message from client %s to peer %s", senderID, receiverID)
					msgsSent.Add(1)
				}

				for range msgToSendPerUser {
					select {
					case <-sender.MsgOutCh:
						msgsReceived.Add(1)
					case <-time.After(3 * time.Second):
						t.Errorf("client %s waited too long for the message", senderID)
						return
					}
				}

				wg.Done()
			}()
		}
	}

	wg.Wait()

	assert.Equal(t, totalMsgsToSend, msgsSent.Load(), "didn't send all messages")
	assert.Equal(t, msgsToReceive, msgsReceived.Load(), "didn't receive all messages")

	t.Logf("All clients successfully exchanged messages")

}
