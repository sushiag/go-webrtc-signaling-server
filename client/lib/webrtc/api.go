package webrtc

import (
	"log"
)

// NOTE: only these functions are public in this package

func (pm *PeerManager) GetPeerIDs() []uint64 {
	resultCh := make(chan []uint64, 1)
	pm.pmEventCh <- pmGetPeerIDs{}
	return <-resultCh
}

func (pm *PeerManager) SendBytesToPeer(peerID uint64, data []byte) error {
	respCh := make(chan error, 1)
	log.Println("CALLED SEND BYTES")
	pm.pmEventCh <- pmSendDataToPeer{peerID, data, respCh}
	return <-respCh
}

func (pm *PeerManager) SendJSONToPeer(peerID uint64, payload Payload) error {
	respCh := make(chan error, 1)
	pm.pmEventCh <- pmSendJSONToPeer{peerID, payload, respCh}
	return <-respCh
}

func (pm *PeerManager) HandleIncomingMessage(msg SignalingMessage, sendFunc func(SignalingMessage) error) {
	pm.pmEventCh <- pmHandleIncomingMsg{msg, sendFunc}
}

func (pm *PeerManager) OutgoingMessages() <-chan SignalingMessage {
	return pm.outgoingMessages
}

func (pm *PeerManager) RemovePeer(peerID uint64, sendFunc func(SignalingMessage) error) {
	pm.pmEventCh <- pmRemovePeer{peerID, sendFunc}
}
