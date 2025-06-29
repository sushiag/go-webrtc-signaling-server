package webrtc

import (
	"encoding/json"
	"log"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
)

func (p *Peer) handleSendLoop() {
	for {
		select {
		case msg := <-p.sendChan:
			log.Printf("[Peer %d] Outgoing message: %s", p.ID, msg)
		case <-p.ctx.Done():
			log.Printf("[Peer %d] sendLoop shutdown", p.ID)
			return
		}
	}
}

func (pm *PeerManager) SendDataToPeer(peerID uint64, data []byte) error {
	// TODO(chee): fix this
	pm.managerQueue <- func() {
		peer, ok := pm.Peers[peerID]
		if !ok || peer.DataChannel == nil {
			log.Printf("[SendDataToPeer] Peer %d not found or no data channel", peerID)
			return
		}
		if err := peer.DataChannel.Send(data); err != nil {
			log.Printf("[SendDataToPeer] Failed sending to peer %d: %v", peerID, err)
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
	// TODO(chee): fix this
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
					Type:   common.MessageTypeHostChanged,
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
