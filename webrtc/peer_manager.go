package webrtc

import (
	"log"
)

func (pm *PeerManager) GracefulShutdown() {
	log.Println("[SHUTDOWN] Initiating graceful shutdown...")

	pm.Peers.Range(func(key, value any) bool {
		id := key.(uint64)
		peer := value.(*Peer)

		log.Printf("[SHUTDOWN] Closing peer %d", id)
		peer.Cancel()

		pm.Peers.Delete(id)
		return true // continue iteration
	})

	log.Println("[SHUTDOWN] All peers closed.")
}
