package webrtc

import (
	"log"
)

// GracefulShutdown stops all peer connections cleanly and clears internal state.
func (pm *PeerManager) GracefulShutdown() {
	log.Println("[SHUTDOWN] Initiating graceful shutdown...")
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	for id, peer := range pm.Peers {
		log.Printf("[SHUTDOWN] Closing peer %d", id)
		peer.Cancel()
	}
	pm.Peers = make(map[uint64]*Peer)
	log.Println("[SHUTDOWN] All peers closed.")
}
