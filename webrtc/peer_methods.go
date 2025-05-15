package webrtc

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/pion/webrtc/v4"
)

func (p *Peer) StartCommandLoop() {
	go func() {
		for cmd := range p.cmdChan {
			cmd.Action(p)
		}
	}()
}

func (p *Peer) OnRemoteDescriptionSet(senderID uint64, sendFunc func(SignalingMessage) error) {
	done := make(chan struct{})
	p.cmdChan <- PeerCommand{
		Action: func(peer *Peer) {
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
			close(done)
		},
	}
	<-done // Optional: wait for operation to complete
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
	peerInterface, ok := pm.Peers.Load(peerID)
	if !ok {
		log.Printf("[SEND ERROR] Peer %d not found", peerID)
		return fmt.Errorf("peer %d not found", peerID)
	}

	peer := peerInterface.(*Peer)

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
	peerInterface, ok := pm.Peers.Load(peerID)
	if !ok {
		log.Printf("[REMOVE ERROR] Peer %d not found for removal", peerID)
		return
	}

	peer := peerInterface.(*Peer)

	peer.Cancel()
	peer.Connection.Close()
	pm.Peers.Delete(peerID)
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
	pm.Peers.Range(func(key, value interface{}) bool {
		id := key.(uint64)
		if nextHost == 0 || id < nextHost {
			nextHost = id
		}
		return true // continue iterating
	})
	return nextHost
}

func (pm *PeerManager) HasPeers() bool {
	has := false
	pm.Peers.Range(func(_, _ any) bool {
		has = true
		return false
	})
	return has
}
