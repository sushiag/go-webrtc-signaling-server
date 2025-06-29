package webrtc

import (
	"fmt"
	"log"
	"time"

	"github.com/pion/webrtc/v4"
)

func (pm *PeerManager) CloseAll() {
	// TODO(chee): fix this
	pm.managerQueue <- func() {
		for id, peer := range pm.Peers {
			if peer.Connection != nil {
				_ = peer.Connection.Close()
			}

			delete(pm.Peers, id)
			log.Printf("[PEER] Closed connection to peer %d", id)
		}
	}
}

func (pm *PeerManager) GetPeerIDs() []uint64 {
	result := make(chan []uint64, 1)
	// TODO(chee): fix this
	pm.managerQueue <- func() {
		ids := make([]uint64, 0, len(pm.Peers))
		for id := range pm.Peers {
			ids = append(ids, id)
		}
		result <- ids
	}
	return <-result
}

func (pm *PeerManager) CheckAllConnectedAndDisconnect() error {
	result := make(chan error, 1)

	// TODO(chee): fix this
	pm.managerQueue <- func() {
		allConnected := true
		for _, peer := range pm.Peers {
			if peer.DataChannel == nil || peer.DataChannel.ReadyState() != webrtc.DataChannelStateOpen {
				allConnected = false
				break
			}
		}

		if allConnected {
			log.Println("[SIGNALING] All peers connected. Closing signaling client for full P2P.")
			go pm.Close()
			result <- nil
		} else {
			log.Println("[SIGNALING] Not all peers connected.")
			result <- fmt.Errorf("not all peers connected")
		}
	}

	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		return fmt.Errorf("timeout while verifying peer connections")
	}
}

func (pm *PeerManager) Close() {
	pm.CloseAll()
	log.Println("[SIGNALING] PeerManager closed.")
}
