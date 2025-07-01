package webrtc

import ws "github.com/sushiag/go-webrtc-signaling-server/client/lib/websocket"

// NOTE: I just moved the public functions to this file for easier visibility

func (pm *PeerManager) GetPeerIDs() []uint64 {
	resultCh := make(chan []uint64, 1)
	pm.pmEventCh <- pmGetPeerIDs{resultCh}
	return <-resultCh
}

func (pm *PeerManager) SendBytesToPeer(peerID uint64, data []byte) error {
	respCh := make(chan error, 1)
	pm.pmEventCh <- pmSendDataToPeer{peerID, data, respCh}
	return <-respCh
}

func (pm *PeerManager) SendJSONToPeer(peerID uint64, payload Payload) error {
	respCh := make(chan error, 1)
	pm.pmEventCh <- pmSendJSONToPeer{peerID, payload, respCh}
	return <-respCh
}

func (pm *PeerManager) HandleIncomingMessage(msg SignalingMessage, responseCh chan ws.Message) {
	pm.pmEventCh <- pmHandleIncomingMsg{msg, responseCh}
}

func (pm *PeerManager) RemovePeer(peerID uint64, responseCh chan ws.Message) {
	pm.pmEventCh <- pmRemovePeer{peerID, responseCh}
}
