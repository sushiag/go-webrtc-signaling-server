package webrtc

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/pion/webrtc/v4"
)

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

func (pm *PeerManager) SendDataToPeer(peerID uint64, data []byte) error {
	result := make(chan error, 1)
	pm.managerQueue <- func() {
		peer, ok := pm.Peers[peerID]
		if !ok {
			result <- fmt.Errorf("peer %d not found", peerID)
			return
		}
		if peer.DataChannel == nil || peer.DataChannel.ReadyState() != webrtc.DataChannelStateOpen {
			result <- fmt.Errorf("data channel not open for peer %d", peerID)
			return
		}
		err := peer.DataChannel.Send(data)
		result <- err
	}
	return <-result
}

func (pm *PeerManager) SendPayloadToPeer(peerID uint64, payload Payload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return pm.SendDataToPeer(peerID, data)
}

func (pm *PeerManager) RemovePeer(peerID uint64, sendFunc func(SignalingMessage) error) {
	pm.managerQueue <- func() {
		peer, ok := pm.Peers[peerID]
		if !ok {
			log.Printf("[REMOVE ERROR] Peer %d not found for removal", peerID)
			return
		}
		delete(pm.Peers, peerID)

		if peer.Connection != nil {
			peer.Connection.Close()
		}

		log.Printf("[PEER] Peer %d removed successfully", peerID)

		if peerID == pm.HostID {
			newHostID := pm.findNextHost()
			if newHostID != 0 {
				pm.HostID = newHostID
				log.Printf("[HOST] Host reassigned to %d", pm.HostID)
				hostChangeMsg := SignalingMessage{
					Type:   "host-changed",
					Sender: pm.UserID,
					Target: 0,
					Users:  pm.GetPeerIDs(),
				}
				if err := sendFunc(hostChangeMsg); err != nil {
					log.Printf("[SIGNALING] Failed to send host-changed message: %v", err)
				}
			}
		}
	}
}
