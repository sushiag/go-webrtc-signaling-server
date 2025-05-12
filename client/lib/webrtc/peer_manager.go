package webrtc

import (
	"log"
)

func (pm *PeerManager) GracefulShutdown() {
	pm.managerQueue <- func() {
		log.Println("[SHUTDOWN] Initiating graceful shutdown...")

		pm.Peers.Range(func(key, value any) bool {
			id, ok := key.(uint64)
			if !ok {
				log.Printf("[SHUTDOWN] Invalid peer ID type: %T", key)
				return true
			}
			peer, ok := value.(*Peer)
			if !ok {
				log.Printf("[SHUTDOWN] Invalid peer type for ID %d: %T", id, value)
				return true
			}

			log.Printf("[SHUTDOWN] Closing peer %d", id)
			peer.Cancel()
			pm.Peers.Delete(id)
			return true
		})

		log.Println("[SHUTDOWN] All peers closed.")
	}
}

func (pm *PeerManager) SetInitialHost(peerIDs []uint64) {
	if len(peerIDs) == 0 {
		return
	}

	pm.managerQueue <- func() {
		minID := peerIDs[0]
		for _, id := range peerIDs[1:] {
			if id < minID {
				minID = id
			}
		}
		pm.HostID = minID
		log.Printf("[HOST] Initial host set to: %d", pm.HostID)
	}
}

func (pm *PeerManager) findNextHost() uint64 {
	var nextHostID uint64 = 0
	pm.Peers.Range(func(key, _ interface{}) bool {
		id := key.(uint64)
		if nextHostID == 0 || id < nextHostID {
			nextHostID = id
		}
		return true
	})
	return nextHostID
}

func (pm *PeerManager) OnPeerCreated(f func(*Peer, SignalingMessage)) {
	pm.onPeerCreated = f
}
