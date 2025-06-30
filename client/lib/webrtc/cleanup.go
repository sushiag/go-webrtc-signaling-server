package webrtc

import (
	"fmt"
	"log"
	"time"

	"github.com/pion/webrtc/v4"
)

func (pm *PeerManager) closeAllSwitch() {
	for id, peer := range pm.Peers {
		if peer.Connection != nil {
			_ = peer.Connection.Close()
		}

		delete(pm.Peers, id)
		log.Printf("[PEER] Closed connection to peer %d\n", id)
	}
	log.Println("[SIGNALING] PeerManager closed.")
}

func (pm *PeerManager) CloseAll() {
	// TODO: convert to pmEvent, struct ok, switch ok
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

func (pm *PeerManager) getPeerIDsSwitch() []uint64 {
	peerIDs := make([]uint64, 0, len(pm.Peers))
	for i, peer := range pm.Peers {
		peerIDs[i] = peer.ID
	}
	return peerIDs
}

func (pm *PeerManager) GetPeerIDs() []uint64 {
	result := make(chan []uint64, 1)
	// TODO: convert to pmEvent, struct ok, switch ok
	pm.managerQueue <- func() {
		ids := make([]uint64, 0, len(pm.Peers))
		for id := range pm.Peers {
			ids = append(ids, id)
		}
		result <- ids
	}
	return <-result
}

// NOTE: what is this for?
func (pm *PeerManager) checkAllConnectedAndDisconnectSwitch() error {
	allConnected := true

	// NOTE: what is happening?
	for _, peer := range pm.Peers {
		if peer.DataChannel == nil || peer.DataChannel.ReadyState() != webrtc.DataChannelStateOpen {
			allConnected = false
			break
		}
	}

	if allConnected {
		log.Println("[SIGNALING] All peers connected. Closing signaling client for full P2P.")
		go pm.Close()
		return nil
	} else {
		log.Println("[SIGNALING] Not all peers connected.")
		return fmt.Errorf("not all peers connected")
	}
}

func (pm *PeerManager) CheckAllConnectedAndDisconnect() error {
	result := make(chan error, 1)

	// TODO: convert to pmEvent, struct ok, switch ok
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
	pm.pmEventCh <- pmCloseAll{}
	pm.CloseAll()
	log.Println("[SIGNALING] PeerManager closed.")
}

// NOTE: what is this for?
func (pm *PeerManager) waitForDataChannelSwitch(peerID uint64, timeout time.Duration) error {
	peer, ok := pm.Peers[peerID]
	if !ok {
		return fmt.Errorf("peer %d not found", peerID)
	}

	if peer.DataChannel != nil && peer.DataChannel.ReadyState() == webrtc.DataChannelStateOpen {
		return nil
	} else {
		return fmt.Errorf("data channel not open for peer %d", peerID)
	}
}

func (pm *PeerManager) WaitForDataChannel(peerID uint64, timeout time.Duration) error {
	result := make(chan error, 1)

	// TODO: convert to pmEvent, struct ok, switch ok
	pm.managerQueue <- func() {
		peer, ok := pm.Peers[peerID]
		if !ok {
			result <- fmt.Errorf("peer %d not found", peerID)
			return
		}
		if peer.DataChannel != nil && peer.DataChannel.ReadyState() == webrtc.DataChannelStateOpen {
			result <- nil
		} else {
			result <- fmt.Errorf("data channel not open for peer %d", peerID)
		}
	}

	select {
	case err := <-result:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for DataChannel for peer %d", peerID)
	}
}
