package webrtc

import (
	"fmt"
	"log"
	"time"
)

func (pm *PeerManager) CloseAll() {
	pm.Peers.Range(func(key, value interface{}) bool {
		peer := value.(*Peer)

		peer.cmdChan <- PeerCommand{
			Action: func(p *Peer) {
				if p.Connection != nil {
					_ = p.Connection.Close()
				}
				p.Cancel()
				log.Printf("[PEER] Closed connection to peer %d", p.ID)
			},
		}

		pm.Peers.Delete(key)
		return true
	})
}

func (pm *PeerManager) GetPeerIDs() []uint64 {
	ids := make([]uint64, 0)

	pm.Peers.Range(func(_, value interface{}) bool {
		if peer, ok := value.(*Peer); ok {
			ids = append(ids, peer.ID)
		}
		return true
	})

	return ids
}

func (pm *PeerManager) CheckAllConnectedAndDisconnect() error {
	allConnected := true

	pm.Peers.Range(func(_, value interface{}) bool {
		peer := value.(*Peer)
		resultChan := make(chan bool)

		peer.cmdChan <- PeerCommand{
			Action: func(p *Peer) {
				resultChan <- p.isConnected
			},
		}

		connected := <-resultChan
		close(resultChan)

		if !connected {
			allConnected = false
			return false
		}
		return true
	})

	if !allConnected {
		return fmt.Errorf("not all peers connected")
	}

	log.Println("[SIGNALING] All peers connected. Closing signaling client for full P2P.")
	pm.Close()
	return nil
}

func (pm *PeerManager) Close() {
	pm.Peers.Range(func(_, value interface{}) bool {
		if peer, ok := value.(*Peer); ok && peer.Connection != nil {
			_ = peer.Connection.Close()
		}
		return true
	})
	log.Println("[SIGNALING] PeerManager closed.")
}

func (pm *PeerManager) WaitForDataChannel(peerID uint64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {

		peer, ok := pm.Peers.Load(peerID)
		if !ok {
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for DataChannel for peer %d", peerID)
			}
			time.Sleep(50 * time.Millisecond)
			continue
		}

		p, ok := peer.(*Peer)
		if !ok {
			return fmt.Errorf("invalid peer data for %d", peerID)
		}

		if p.DataChannel != nil {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for DataChannel for peer %d", peerID)
		}

		time.Sleep(50 * time.Millisecond)
	}
}
