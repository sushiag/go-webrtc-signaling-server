package webrtc

import (
	"log"
)

func (pm *PeerManager) GracefulShutdown() {
	log.Println("[SHUTDOWN] Initiating graceful shutdown...")

	var keys []uint64
	pm.Peers.Range(func(key, _ any) bool {
		keys = append(keys, key.(uint64))
		return true
	})
	for _, id := range keys {
		value, _ := pm.Peers.Load(id)
		peer := value.(*Peer)
		peer.Cancel()
		pm.Peers.Delete(id)
	}

	log.Println("[SHUTDOWN] All peers closed.")
}
