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
	clients        map[uint64]*webRTCPeerManager
	sdpSignalingCh chan sdpSignalingRequest
	iceSignalingCh chan iceSignalingRequest
}

var mockSignaler = &signalingManager{
	clients:        make(map[uint64]*webRTCPeerManager, 2),
	sdpSignalingCh: make(chan sdpSignalingRequest, 10),
	iceSignalingCh: make(chan iceSignalingRequest, 10),
}

func TestPeerManager(t *testing.T) {
	mockSignalingMngr := newMockSignalingManager(t)

	pm1 := newPeerManager(mockSignalingMngr, 1)
	pm2 := newPeerManager(mockSignalingMngr, 2)

	mockSignalingMngr.clients[pm1.clientID] = pm1
	mockSignalingMngr.clients[pm2.clientID] = pm2

	pm1.newPeerOffer(pm2.clientID)

	openedCh1 := <-pm1.dataChOpened
	t.Logf("data channel for %d opened", openedCh1)
	openedCh2 := <-pm2.dataChOpened
	t.Logf("data channel for %d opened", openedCh2)

	wg := sync.WaitGroup{}
	wg.Add(2)

	msgsToSend := 5

	// Sending from pm1 -> pm2
	go func() {
		t.Logf("client %d: loop started", pm1.clientID)
		msgsReceived := make([]string, 0)
		msgsSent := 0

		isSendingMsgs := true
		isReceivingMsgs := true
		for isSendingMsgs || isReceivingMsgs {
			select {
			// Message receiving
			case msg := <-pm1.msgOutCh:
				{
					assert.Equal(t, msg.from, openedCh1, "the 'from' field of the received message by client %d is wrong", pm1.clientID)
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
							t.Logf("[ERROR] client %d: failed to send message to %d: %v", pm1.clientID, openedCh1, err)
						} else {
							t.Logf("client %d: sent message to %d", pm1.clientID, openedCh1)
						}
						msgsSent += 1

						isSendingMsgs = msgsSent < 5
					}
				}
			}

			time.Sleep(time.Millisecond * 15)
		}

		expectedMsgs := []string{"5", "4", "3", "2", "1"}
		assert.Equal(t, expectedMsgs, msgsReceived, "the received messages by client %d were wrong", pm1.clientID)
		assert.Equal(t, msgsToSend, msgsSent, "client %d did not send enough messages", pm1.clientID)

		t.Logf("client %d: loop ended", pm1.clientID)
		wg.Done()
	}()

	// Sending from pm2 -> pm1
	go func() {
		t.Logf("client %d: loop started", pm2.clientID)
		msgsReceived := make([]string, 0)
		msgsSent := 0

		isSendingMsgs := true
		isReceivingMsgs := true
		for isSendingMsgs || isReceivingMsgs {

			select {
			// Message receiving
			case msg := <-pm2.msgOutCh:
				{
					assert.Equal(t, openedCh2, msg.from, "the 'from' field of the received message by client %d is wrong", pm2.clientID)
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
							t.Errorf("client %d: failed to send message to %d: %v", pm2.clientID, openedCh2, err)
						} else {
							t.Logf("client %d: sent message to %d", pm1.clientID, openedCh2)
						}
						msgsSent += 1

						isSendingMsgs = msgsSent < 5
					}
				}
			}

			time.Sleep(time.Millisecond * 15)
		}

		expectedMsgs := []string{"1", "2", "3", "4", "5"}
		assert.Equal(t, expectedMsgs, msgsReceived, "the received messages by client %d were wrong", pm2.clientID)
		assert.Equal(t, msgsToSend, msgsSent, "client %d did not send enough messages", pm2.clientID)

		t.Logf("client %d: loop ended", pm2.clientID)
		wg.Done()
	}()

	wg.Wait()
}

func newMockSignalingManager(t *testing.T) *signalingManager {
	mngr := &signalingManager{
		clients:        make(map[uint64]*webRTCPeerManager, 2),
		sdpSignalingCh: make(chan sdpSignalingRequest, 10),
		iceSignalingCh: make(chan iceSignalingRequest, 10),
	}

	go func() {
		for {
			select {
			case sdpReq := <-mngr.sdpSignalingCh:
				{
					t.Logf("signaling SDP from %d to %d", sdpReq.from, sdpReq.to)

					switch sdpReq.sdp.Type {
					case webrtc.SDPTypeOffer:
						{
							targetClient, exists := mngr.clients[sdpReq.to]
							if !exists {
								t.Errorf("client %d tried to send SDP offer to nonexistent client: %d", sdpReq.from, sdpReq.to)
							}

							targetClient.handleSDPOffer(sdpReq.from, sdpReq.sdp)
						}
					case webrtc.SDPTypeAnswer:
						{
							targetClient, exists := mngr.clients[sdpReq.to]
							if !exists {
								t.Errorf("client %d tried to send signaling message to nonexistent client: %d", sdpReq.from, sdpReq.to)
							}

							targetConn, exists := targetClient.connections[sdpReq.from]
							if !exists {
								t.Errorf("client %d tried to send signaling message to client %d but they were not expecting a message", sdpReq.from, sdpReq.to)
							}

							targetConn.conn.SetRemoteDescription(sdpReq.sdp)
						}
					}

				}

			case iceReq := <-mngr.iceSignalingCh:
				{
					t.Logf("signaling ICE candidate from %d to %d", iceReq.from, iceReq.to)
					candidate := webrtc.ICECandidateInit{Candidate: iceReq.iceCandidate.ToJSON().Candidate}

					targetClient, exists := mngr.clients[iceReq.to]

					if !exists {
						t.Logf("client %d tried to send an ICE candidate to a nonexistent client: %d", iceReq.from, iceReq.to)
					}

					targetConn, exists := targetClient.connections[iceReq.from]
					if !exists {
						t.Errorf("client %d tried to send an ICE candidate to client %d but they were not expecting one", iceReq.from, iceReq.to)
					}
					targetConn.conn.AddICECandidate(candidate)
				}
			}
		}
	}()

	return mngr
}
