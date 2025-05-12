package webrtc

import (
	"errors"
	"fmt"
	"log"
	"time"
)

func (pm *PeerManager) CloseAll() {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	for id, peer := range pm.Peers {
		if peer.Connection != nil {
			peer.Connection.Close()
		}
		peer.Cancel()
		delete(pm.Peers, id)
		log.Printf("[PEER] Closed connection to peer %d", id)
	}
}

func (pm *PeerManager) GetPeerIDs() []uint64 {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	ids := make([]uint64, 0, len(pm.Peers))
	for id := range pm.Peers {
		ids = append(ids, id)
	}
	return ids
}

func (pm *PeerManager) CheckAllConnectedAndDisconnect() error {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	for _, p := range pm.Peers {
		p.mutex.Lock()
		connected := p.isConnected
		p.mutex.Unlock()

		if !connected {
			return errors.New("not all peers connected")
		}
	}

	log.Println("[SIGNALING] All peers connected. Closing signaling client for full P2P.")
	pm.Close()
	return nil
}

func (pm *PeerManager) Close() {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	// Close all peer connections
	for _, peer := range pm.Peers {
		if peer.Connection != nil {
			peer.Connection.Close()
		}
	}

	// Any other cleanup you need
	log.Println("[SIGNALING] PeerManager closed.")
}

func (pm *PeerManager) WaitForDataChannel(peerID uint64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		pm.Mutex.Lock()
		peer, ok := pm.Peers[peerID]
		pm.Mutex.Unlock()

		if ok && peer.DataChannel != nil {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for DataChannel for peer %d", peerID)
		}

		time.Sleep(50 * time.Millisecond)
	}
}
