package webrtc

import (
	"encoding/json"
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

func (peer *Peer) Cancel() {
	peer.cancelOnce.Do(func() {
		peer.cancel()
	})
}

func (pm *PeerManager) SendDataToPeer(peerID uint64, data []byte) error {
	pm.managerQueue <- func() {
		value, ok := pm.Peers.Load(peerID)
		if !ok {
			log.Printf("[SEND ERROR] Peer %d not found", peerID)
			return
		}
		peer, ok := value.(*Peer)
		if !ok {
			log.Printf("[SEND ERROR] Peer %d has invalid type", peerID)
			return
		}

		if peer.DataChannel == nil || peer.DataChannel.ReadyState() != webrtc.DataChannelStateOpen {
			log.Printf("[SEND ERROR] DataChannel for peer %d is not open", peerID)
			return
		}

		err := peer.DataChannel.Send(data)
		if err != nil {
			log.Printf("[SEND ERROR] Failed to send data: %v", err)
		}
	}
	return nil
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
		value, ok := pm.Peers.LoadAndDelete(peerID)
		if !ok {
			log.Printf("[REMOVE ERROR] Peer %d not found for removal", peerID)
			return
		}
		peer, ok := value.(*Peer)
		if !ok {
			log.Printf("[REMOVE ERROR] Peer %d has invalid type", peerID)
			return
		}

		peer.Cancel()
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
