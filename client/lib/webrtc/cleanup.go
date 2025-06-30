package webrtc

import (
	"fmt"
	"log"

	"github.com/pion/webrtc/v4"
)

func (pm *PeerManager) closeAllSwitch() {
	for id, peer := range pm.peers {
		if peer.Connection != nil {
			_ = peer.Connection.Close()
		}

		_, exists := pm.peers[id]
		if exists {
			delete(pm.peers, id)
			log.Printf("[PEER] Closed connection to peer %d\n", id)
		} else {
			log.Printf("[WARN] Tried to close connection to an unknown peer %d\n", id)
		}
	}
}

func (pm *PeerManager) getPeerIDsSwitch() []uint64 {
	peerIDs := make([]uint64, len(pm.peers))

	i := 0
	for _, peer := range pm.peers {
		peerIDs[i] = peer.ID
		i++
	}

	return peerIDs
}

// NOTE: what is this for?
func (pm *PeerManager) checkAllConnectedAndDisconnectSwitch() error {
	allConnected := true

	// NOTE: what is happening?
	for _, peer := range pm.peers {
		if peer.DataChannel == nil || peer.DataChannel.ReadyState() != webrtc.DataChannelStateOpen {
			allConnected = false
			break
		}
	}

	if allConnected {
		log.Println("[SIGNALING] All peers connected. Closing signaling client for full P2P.")
		pm.closeAllSwitch()
		log.Println("[SIGNALING] PeerManager closed")
		return nil
	} else {
		log.Println("[SIGNALING] Not all peers connected.")
		return fmt.Errorf("not all peers connected")
	}
}
