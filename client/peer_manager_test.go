package client

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/assert"
)

type mockSignalingManager struct {
	clients        map[uint64]*peerManager
	sdpSignalingCh chan sendSDP
	iceSignalingCh chan sendICECandidate
}

var mockSignaler = &signalingManager{
	clients: make(map[uint64]*peerManager, 2),
}

func TestPeerManager(t *testing.T) {
	pm1EventsCh := make(chan Event, 8)
	pm2EventsCh := make(chan Event, 8)
	clientID1 := uint64(1)
	clientID2 := uint64(2)
	pm1 := newPeerManager(pm1EventsCh)
	pm2 := newPeerManager(pm2EventsCh)
	clients := map[uint64]*peerManager{
		clientID1: pm1,
		clientID2: pm2,
	}

	newMockSignalingServer(t, clients)

	pm1.newPeerOffer(clientID2)

	dataChannelsOpened := 0
	deadline := time.After(time.Second * 3)
	var openedCh1 uint64
	var openedCh2 uint64
	for dataChannelsOpened < 2 {
		select {
		case ev := <-pm1EventsCh:
			{
				switch v := ev.(type) {
				case PeerDataChOpenedEvent:
					{
						if assert.Equalf(t, clientID2, v.PeerID, "client %d should be opening a data channel for client %d", clientID1, clientID2) {
							dataChannelsOpened += 1
							t.Logf("data channel for %d opened", v.PeerID)
							openedCh1 = v.PeerID
						}
					}
				}
			}
		case ev := <-pm2EventsCh:
			{
				switch v := ev.(type) {
				case PeerDataChOpenedEvent:
					{
						if assert.Equalf(t, clientID1, v.PeerID, "client %d should be opening a data channel for client %d", clientID2, clientID1) {
							dataChannelsOpened += 1
							t.Logf("data channel for %d opened", v.PeerID)
							openedCh2 = v.PeerID
						}
					}
				}
			}
		case <-deadline:
			{
				t.Fatal("peers took too long to open their data channels")
			}
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	msgsToSend := 5

	// Sending from pm1 -> pm2
	go func() {
		t.Logf("client %d: loop started", clientID1)
		msgsReceived := make([]string, 0)
		msgsSent := 0

		isSendingMsgs := true
		isReceivingMsgs := true
		for isSendingMsgs || isReceivingMsgs {
			select {
			// Message receiving
			case msg := <-pm1.msgOutCh:
				{
					assert.Equal(t, msg.from, openedCh1, "the 'from' field of the received message by client %d is wrong", clientID1)
					msgsReceived = append(msgsReceived, msg.msg)

					isReceivingMsgs = len(msgsReceived) < msgsToSend
				}
			// Message sending
			default:
				{
					if !isSendingMsgs {
						break
					}

					if msgsSent < msgsToSend {
						msg := fmt.Sprintf("%d", msgsSent+1)
						err := pm1.sendMsgToPeer(openedCh1, msg)
						if err != nil {
							t.Logf("[ERROR] client %d: failed to send message to %d: %v", clientID1, openedCh1, err)
						} else {
							t.Logf("client %d: sent message to %d", clientID1, openedCh1)
						}
						msgsSent += 1

						isSendingMsgs = msgsSent < 5
					}
				}
			}

			time.Sleep(time.Millisecond * 15)
		}

		expectedMsgs := []string{"5", "4", "3", "2", "1"}
		assert.Equal(t, expectedMsgs, msgsReceived, "the received messages by client %d were wrong", clientID1)
		assert.Equal(t, msgsToSend, msgsSent, "client %d did not send enough messages", clientID1)

		t.Logf("client %d: loop ended", clientID1)
		wg.Done()
	}()

	// Sending from pm2 -> pm1
	go func() {
		t.Logf("client %d: loop started", clientID2)
		msgsReceived := make([]string, 0)
		msgsSent := 0

		isSendingMsgs := true
		isReceivingMsgs := true
		for isSendingMsgs || isReceivingMsgs {

			select {
			// Message receiving
			case msg := <-pm2.msgOutCh:
				{
					assert.Equal(t, openedCh2, msg.from, "the 'from' field of the received message by client %d is wrong", clientID2)
					msgsReceived = append(msgsReceived, msg.msg)

					isReceivingMsgs = len(msgsReceived) < msgsToSend
				}
			// Message sending
			default:
				{
					if !isSendingMsgs {
						break
					}

					if msgsSent < msgsToSend {
						msg := fmt.Sprintf("%d", msgsToSend-msgsSent)
						err := pm2.sendMsgToPeer(openedCh2, msg)
						if err != nil {
							t.Errorf("client %d: failed to send message to %d: %v", clientID2, openedCh2, err)
						} else {
							t.Logf("client %d: sent message to %d", clientID2, openedCh2)
						}
						msgsSent += 1

						isSendingMsgs = msgsSent < 5
					}
				}
			}

			time.Sleep(time.Millisecond * 15)
		}

		expectedMsgs := []string{"1", "2", "3", "4", "5"}
		assert.Equal(t, expectedMsgs, msgsReceived, "the received messages by client %d were wrong", clientID2)
		assert.Equal(t, msgsToSend, msgsSent, "client %d did not send enough messages", clientID2)

		t.Logf("client %d: loop ended", clientID2)
		wg.Done()
	}()

	wg.Wait()
}

type mockSignalingServer struct {
	clients map[uint64]*peerManager
}

func newMockSignalingServer(t *testing.T, clients map[uint64]*peerManager) *mockSignalingServer {
	server := &mockSignalingServer{
		clients: clients,
	}

	for clientID, client := range clients {
		t.Logf("preparing signaling loop for client %d", clientID)

		go func() {
			t.Logf("started signaling loop for client %d", clientID)
			for {
				select {
				case sdpReq := <-client.sdpCh:
					{
						t.Logf("signaling SDP from %d to %d", clientID, sdpReq.to)

						switch sdpReq.sdp.Type {
						case webrtc.SDPTypeOffer:
							{
								targetClient, exists := server.clients[sdpReq.to]
								if !exists {
									t.Errorf("client %d tried to send SDP offer to nonexistent client: %d", clientID, sdpReq.to)
								}

								targetClient.handleSDPOffer(clientID, sdpReq.sdp)
							}
						case webrtc.SDPTypeAnswer:
							{
								targetClient, exists := server.clients[sdpReq.to]
								if !exists {
									t.Errorf("client %d tried to send signaling message to nonexistent client: %d", clientID, sdpReq.to)
								}

								targetConn, exists := targetClient.connections[clientID]
								if !exists {
									t.Errorf("client %d tried to send signaling message to client %d but they were not expecting a message", clientID, sdpReq.to)
								}

								targetConn.conn.SetRemoteDescription(sdpReq.sdp)
							}
						}

					}

				case iceReq := <-client.iceCh:
					{
						t.Logf("signaling ICE candidate from %d to %d", clientID, iceReq.to)
						candidate := webrtc.ICECandidateInit{Candidate: iceReq.iceCandidate.ToJSON().Candidate}

						targetClient, exists := server.clients[iceReq.to]

						if !exists {
							t.Logf("client %d tried to send an ICE candidate to a nonexistent client: %d", clientID, iceReq.to)
						}

						targetConn, exists := targetClient.connections[clientID]
						if !exists {
							t.Errorf("client %d tried to send an ICE candidate to client %d but they were not expecting one", clientID, iceReq.to)
						}
						targetConn.conn.AddICECandidate(candidate)
					}
				}
			}
		}()
	}

	return server
}
