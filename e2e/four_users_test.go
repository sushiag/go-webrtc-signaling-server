package e2e_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	client "github.com/sushiag/go-webrtc-signaling-server/client"
	server "github.com/sushiag/go-webrtc-signaling-server/server/lib/server"
	"github.com/sushiag/go-webrtc-signaling-server/server/lib/server/db"
)

func TestEndToEndSignalingFourUsers(t *testing.T) {
	server, serverURL := server.StartServer("0", &db.Queries{})
	defer server.Close()
	wsEndpoint := fmt.Sprintf("ws://%s/ws", serverURL)

	apiKeys := []string{
		"valid-api-key-1",
		"valid-api-key-2",
		"valid-api-key-3",
		"valid-api-key-4",
	}
	nClients := len(apiKeys)

	clients := make([]*client.Client, nClients)
	for i := range nClients {
		newClient, err := client.NewClientWithKey(wsEndpoint, apiKeys[i])
		require.NoErrorf(t, err, "failed to initialize client %d", i)
		clients[i] = newClient
	}
	t.Log("clients initialized")

	roomID, err := clients[0].CreateRoom()
	require.NoError(t, err)

	for i := 1; i < nClients; i++ {
		_, err := clients[i].JoinRoom(roomID)
		require.NoErrorf(t, err, "client %d failed to join room", clients[i].GetClientID())
	}
	t.Log("clients joined room")

	t.Logf("waiting for data channels to open...")
	connectedPeers := sync.Map{} // connected peerIDs for each client: map[clientID][]peerID
	dataChannelsOk := make(chan uint64, 4)
	for _, client := range clients {
		clientID := client.GetClientID()
		connectedPeers.Store(clientID, []uint64{})

		go func() {
			dataChOpened := client.GetDataChOpened()

			for range nClients - 1 {
				newPeerID := <-dataChOpened

				// Check if the new data channel doesn't have a duplicate
				peersAny, ok := connectedPeers.Load(clientID)
				require.True(t, ok)
				peers, ok := peersAny.([]uint64) // type cast
				require.NotContains(t, peers, newPeerID)

				// Add the new peerID then update the sync.Map
				peers = append(peers, newPeerID)
				connectedPeers.Store(clientID, peers)
			}

			dataChannelsOk <- clientID
		}()
	}

	clientsReady := 0
	dataChDeadline := time.After(3 * time.Second)
	for clientsReady < nClients {
		select {
		case clientID := <-dataChannelsOk:
			t.Logf("client %d finished waiting for data channels to open", clientID)
			clientsReady += 1
		case <-dataChDeadline:
			{
				t.Fatalf("clients took longer than 3 secs to open their data channels, only opened: %d", clientsReady)
			}
		}
	}
	t.Logf("all data channels opened sucessfully!")

	rounds := 1
	msgsSent := atomic.Uint32{}
	msgsReceived := atomic.Uint32{}
	msgToSendPerUser := nClients - 1
	totalMsgsToSend := uint32(nClients * msgToSendPerUser * rounds)
	msgsToReceive := uint32(totalMsgsToSend)
	t.Logf("total messages to send: %d", totalMsgsToSend)
	wg := sync.WaitGroup{}

	for round := 1; round <= 1; round++ {
		t.Logf("---- Round %d ----", round)

		for _, sender := range clients {
			wg.Add(1)

			go func() {
				senderID := sender.GetClientID()
				peersAny, ok := connectedPeers.Load(senderID)
				require.True(t, ok)

				peers := peersAny.([]uint64)
				for _, receipientID := range peers {
					msg := fmt.Sprintf("hello client %d from client %d", receipientID, senderID)
					t.Logf("sending data %d->%d", senderID, receipientID)
					sender.SendDataToPeer(uint64(receipientID), []byte(msg))
					msgsSent.Add(1)
				}

				msgOutCh := sender.GetPeerDataMsgCh()
				for range msgToSendPerUser {
					select {
					case <-msgOutCh:
						msgsReceived.Add(1)
					case <-time.After(3 * time.Second):
						t.Errorf("client %d waited too long for the message", senderID)
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
