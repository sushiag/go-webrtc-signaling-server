package webrtc

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/pion/webrtc/v4"
)

func NewPeerManager(userID uint64) *PeerManager {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
	return &PeerManager{
		UserID: userID,
		Peers:  make(map[uint64]*Peer),
		Config: config,
	}
}
func (pm *PeerManager) SetInitialHost(peerIDs []uint64) {
	if len(peerIDs) == 0 {
		return
	}
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	minID := peerIDs[0]
	for _, id := range peerIDs {
		if id < minID {
			minID = id
		}
	}
	pm.HostID = minID
	log.Printf("[HOST] Initial host set to: %d", pm.HostID)
}

func (peer *Peer) OnRemoteDescriptionSet(senderID uint64, sendFunc func(SignalingMessage) error) {
	peer.mutex.Lock()
	defer peer.mutex.Unlock()

	peer.remoteDescriptionSet = true
	for _, c := range peer.bufferedICECandidates {
		_ = sendFunc(SignalingMessage{
			Type:      "ice-candidate",
			Sender:    senderID,
			Target:    peer.ID,
			Candidate: c.Candidate,
		})
	}
	peer.bufferedICECandidates = nil
}

func (peer *Peer) handleSendLoop() {
	for {
		select {
		case <-peer.ctx.Done():
			return
		case msg := <-peer.sendChan:
			if peer.Connection.ConnectionState() == webrtc.PeerConnectionStateConnected {
				_ = peer.DataChannel.SendText(msg)
			}
		}
	}
}

func (p *Peer) Cancel() {
	p.cancelOnce.Do(func() {
		p.cancel()
	})
}
func (pm *PeerManager) SendDataToPeer(peerID uint64, data []byte) error {
	pm.Mutex.Lock()
	peer, ok := pm.Peers[peerID]
	pm.Mutex.Unlock()

	if !ok {
		log.Printf("[SEND ERROR] Peer %d not found", peerID)
		return fmt.Errorf("peer %d not found", peerID)
	}

	// Wait until the data channel is available
	if peer.DataChannel == nil {
		log.Printf("[SEND ERROR] Peer %d has no DataChannel", peerID)
		return fmt.Errorf("peer %d has no DataChannel", peerID)
	}

	// Make sure the DataChannel is open
	if peer.DataChannel.ReadyState() != webrtc.DataChannelStateOpen {
		log.Printf("[SEND ERROR] DataChannel for peer %d is not open", peerID)
		return fmt.Errorf("data channel for peer %d is not open", peerID)
	}

	return peer.DataChannel.Send(data)
}

func (pm *PeerManager) SendPayloadToPeer(peerID uint64, payload Payload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return pm.SendDataToPeer(peerID, data)
}

func (pm *PeerManager) OnPeerCreated(f func(*Peer, SignalingMessage)) {
	pm.onPeerCreated = f
}

func (pm *PeerManager) RemovePeer(peerID uint64, sendFunc func(SignalingMessage) error) {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	peer, ok := pm.Peers[peerID]
	if !ok {
		log.Printf("[REMOVE ERROR] Peer %d not found for removal", peerID)
		return
	}

	peer.Cancel()
	peer.Connection.Close()
	delete(pm.Peers, peerID)
	log.Printf("[PEER] Peer %d removed successfully", peerID)

	// Host reassignment logic
	if peerID == pm.HostID {
		newHostID := pm.findNextHost()
		if newHostID != 0 {
			pm.HostID = pm.findNextHost()
			log.Printf("[HOST] Host reassigned to %d", pm.HostID)

			// Notify others about the host change
			hostChangeMsg := SignalingMessage{
				Type:   "host-changed",
				Sender: pm.UserID,
				Target: 0, // Broadcast to all peers
				Users:  pm.GetPeerIDs(),
			}
			if err := sendFunc(hostChangeMsg); err != nil {
				log.Printf("[SIGNALING] Failed to send host-changed message: %v", err)
			}
		}
	}
}

func (pm *PeerManager) findNextHost() uint64 {
	var nextHost uint64 = 0
	for id := range pm.Peers {
		if nextHost == 0 || id < nextHost {
			nextHost = id
		}
	}
	return nextHost
}
