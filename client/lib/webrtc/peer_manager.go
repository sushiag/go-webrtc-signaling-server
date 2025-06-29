package webrtc

import (
	"log"
)

func (pm *PeerManager) GracefulShutdown() {

	log.Println("[SHUTDOWN] Initiating graceful shutdown...")
	for id := range pm.Peers {
		log.Printf("[SHUTDOWN] Closing peer %d", id)
		delete(pm.Peers, id)
	}
	log.Println("[SHUTDOWN] All peers closed.")

}
func (pm *PeerManager) SetInitialHost(peerIDs []uint64) {
	if len(peerIDs) == 0 {
		return
	}

	done := make(chan struct{})

	minID := peerIDs[0]
	for _, id := range peerIDs[1:] {
		if id < minID {
			minID = id
		}
	}
	pm.HostID = minID
	log.Printf("[HOST] Initial host set to: %d", pm.HostID)
	close(done)

	<-done // wait until HostID is set before continuing
}

func (pm *PeerManager) findNextHost() uint64 {
	var nextHostID uint64 = 0
	for id := range pm.Peers {
		if nextHostID == 0 || id < nextHostID {
			nextHostID = id
		}
	}
	return nextHostID
}

func (pm *PeerManager) OnPeerCreated(f func(*Peer, SignalingMessage)) {
	pm.onPeerCreated = f
}
