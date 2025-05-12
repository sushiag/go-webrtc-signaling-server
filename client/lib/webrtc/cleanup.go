package webrtc

import (
	"errors"
	"fmt"
	"log"
	"time"
)

func (pm *PeerManager) CloseAll() {
	pm.Peers.Range(func(key, value any) bool {
		id, ok := key.(uint64)
		if !ok {
			return true
		}
		peer, ok := value.(*Peer)
		if !ok {
			return true
		}

		go func(id uint64, peer *Peer) {
			peer.sendChan <- "close"

			if peer.Connection != nil {
				peer.Connection.Close()
			}
			peer.Cancel()

			pm.Peers.Delete(id)
			log.Printf("[PEER] Closed connection to peer %d", id)
		}(id, peer)

		return true
	})
}

func (pm *PeerManager) GetPeerIDs() []uint64 {
	ids := make([]uint64, 0)

	pm.Peers.Range(func(key, value any) bool {
		if id, ok := key.(uint64); ok {
			ids = append(ids, id)
		}
		return true
	})
	return ids
}

func (pm *PeerManager) CheckAllConnectedAndDisconnect() error {
	var count int
	pm.Peers.Range(func(_, _ any) bool {
		count++
		return true
	})
	errCh := make(chan error, count)

	pm.Peers.Range(func(_, value any) bool {
		peer, ok := value.(*Peer)
		if !ok {
			errCh <- errors.New("invalid peer type in map")
			return true
		}

		go func(peer *Peer) {
			peer.sendChan <- "start-session"

			select {
			case connected := <-peer.sendChan:
				if connected != "true" {
					errCh <- errors.New("not all peers connected")
					return
				}
				errCh <- nil
			case <-time.After(2 * time.Second):
				errCh <- errors.New("timeout waiting for peer connection status")
			}
		}(peer)

		return true
	})

	for i := 0; i < count; i++ {
		if err := <-errCh; err != nil {
			log.Println("[SIGNALING] Not all peers connected.")
			return err
		}
	}

	log.Println("[SIGNALING] All peers connected. Closing signaling client for full P2P.")
	pm.Close()
	return nil
}

func (pm *PeerManager) Close() {
	pm.Peers.Range(func(_, value any) bool {
		peer, ok := value.(*Peer)
		if !ok {
			return true
		}

		go func(peer *Peer) {
			peer.sendChan <- "close"
			if peer.Connection != nil {
				peer.Connection.Close()
			}
		}(peer)

		return true
	})

	log.Println("[SIGNALING] PeerManager closed.")
}

func (pm *PeerManager) WaitForDataChannel(peerID uint64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		value, ok := pm.Peers.Load(peerID)
		if !ok {
			log.Printf("peer %d not found", peerID)
			continue
		}
		peer, ok := value.(*Peer)
		if !ok {
			log.Printf("peer %d has invalid type", peerID)
			continue
		}

		peer.sendChan <- "check_data_channel"

		select {
		case <-peer.sendChan:
			return nil
		case <-time.After(time.Until(deadline)):
			return fmt.Errorf("timeout waiting for DataChannel for peer %d", peerID)
		}
	}
}
