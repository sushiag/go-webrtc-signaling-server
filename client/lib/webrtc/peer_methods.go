package webrtc

import (
	"encoding/json"
	"fmt"
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

func (pm *PeerManager) sendBytesToPeerSwitch(peerID uint64, data []byte) error {
	peer, ok := pm.peers[peerID]
	if !ok || peer.DataChannel == nil {
		log.Printf("[SendDataToPeer] Peer %d not found or no data channel", peerID)
		return nil
	}

	if err := peer.DataChannel.Send(data); err != nil {
		return fmt.Errorf("[SendDataToPeer] Failed sending to peer %d: %v", peerID, err)
	}

	return nil
}

func (pm *PeerManager) sendJSONToPeerSwitch(peerID uint64, payload Payload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return pm.sendBytesToPeerSwitch(peerID, data)
}

func (pm *PeerManager) removePeerSwitch(peerID uint64, sendFunc func(SignalingMessage) error) {
	peer, exists := pm.peers[peerID]
	if !exists {
		log.Printf("[REMOVE ERROR] Peer %d not found for removal", peerID)
		return
	}
	delete(pm.peers, peerID)

	if peer.Connection != nil {
		peer.Connection.Close()
	}

	log.Printf("[PEER] Peer %d removed successfully", peerID)

	if peerID == pm.hostID {
		newHostID := pm.findNextHost()
		if newHostID != 0 {
			pm.hostID = newHostID
			log.Printf("[HOST] Host reassigned to %d", pm.hostID)

			hostChangeMsg := SignalingMessage{
				Type:   common.MessageTypeHostChanged,
				Sender: pm.userID,
				Target: 0,
				Users:  pm.getPeerIDsSwitch(),
			}
			if err := sendFunc(hostChangeMsg); err != nil {
				log.Printf("[SIGNALING] Failed to send host-changed message: %v", err)
			}
		}
	}
}
