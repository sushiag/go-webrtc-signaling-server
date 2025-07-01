package webrtc

import (
	"log"
)

func (pm *PeerManager) GracefulShutdown() {
	log.Println("[SHUTDOWN] Initiating graceful shutdown...")
	for id := range pm.peers {
		log.Printf("[SHUTDOWN] Closing peer %d", id)
		delete(pm.peers, id)
	}
	log.Println("[SHUTDOWN] All peers closed.")
}

func (pm *PeerManager) SetInitialHost(peerIDs []uint64) {
	if len(peerIDs) == 0 {
		return
	}

	minID := peerIDs[0]
	for _, id := range peerIDs[1:] {
		if id < minID {
			minID = id
		}
	}
	pm.hostID = minID
	log.Printf("[HOST] Initial host set to: %d", pm.hostID)
}

func (pm *PeerManager) findNextHost() uint64 {
	var nextHostID uint64 = 0
	for id := range pm.peers {
		if nextHostID == 0 || id < nextHostID {
			nextHostID = id
		}
	}
	return nextHostID
}
