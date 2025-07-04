package webrtc

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
	ws "github.com/sushiag/go-webrtc-signaling-server/client/lib/websocket"
)

func (pm *PeerManager) sendBytesToPeerSwitch(peerID uint64, data []byte) error {
	peer, peerExists := pm.peers[peerID]
	if !peerExists {
		return fmt.Errorf("peer %d not found", peerID)
	}

	if peer.DataChannel == nil {
		return fmt.Errorf("no data channel initialized for peer %d", peerID)
	}

	if err := peer.DataChannel.Send(data); err != nil {
		return fmt.Errorf("failed sending to peer %d: %v", peerID, err)
	}

	return nil
}

func (pm *PeerManager) sendJSONToPeerSwitch(peerID uint64, payload Payload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload to JSON")
	}

	return pm.sendBytesToPeerSwitch(peerID, data)
}

func (pm *PeerManager) removePeerSwitch(peerID uint64, responseCh chan ws.Message) {
	peer, exists := pm.peers[peerID]
	if !exists {
		log.Printf("[REMOVE ERROR] Peer %d not found for removal\n", peerID)
		return
	}
	delete(pm.peers, peerID)

	if peer.Connection != nil {
		peer.Connection.Close()
	}

	log.Printf("[PEER] Peer %d removed successfully\n", peerID)

	if peerID == pm.hostID {
		newHostID := pm.findNextHost()
		if newHostID != 0 {
			pm.hostID = newHostID
			log.Printf("[HOST] Host reassigned to %d\n", pm.hostID)

			hostChangeMsg := ws.Message{
				Type:   common.MessageTypeHostChanged,
				Sender: pm.UserID,
				Target: 0,
				Users:  pm.getPeerIDsSwitch(),
			}
			log.Println("[INFO] Sending host-changed message")
			responseCh <- hostChangeMsg
		}
	}
}
