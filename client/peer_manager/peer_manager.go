package peer_manager

import (
	"fmt"

	"github.com/pion/webrtc/v4"

	smsg "signaling-msgs"
)

type PeerManager struct {
	peers        map[uint64]*peer
	signalingOut chan<- smsg.MessageAnyPayload
	dataChOpened chan uint64
	peerData     chan PeerDataMsg
}

type PeerDataMsg struct {
	From uint64
	Data []byte
}

type peer struct {
	conn   *webrtc.PeerConnection
	dataCh *webrtc.DataChannel
}

type sendICE struct {
	forPeer uint64
	ice     *webrtc.ICECandidate
}

var defaultWebRTCConfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{URLs: []string{"stun:stun.l.google.com:19302"}},
	},
}

// signalingIn:		source of signaling messsages
// signalingOut:	output for signaling messsages
func NewPeerManager(signalingIn <-chan smsg.MessageRawJSONPayload, signalingOut chan<- smsg.MessageAnyPayload) *PeerManager {
	client := &PeerManager{
		peers:        make(map[uint64]*peer),
		signalingOut: signalingOut,
		dataChOpened: make(chan uint64, 4),
		peerData:     make(chan PeerDataMsg, 32),
	}

	go client.signalingLoop(signalingIn)

	return client
}

func (pm *PeerManager) SendDataToPeer(peerID uint64, data []byte) error {
	conn, exists := pm.peers[peerID]
	if !exists {
		return fmt.Errorf("tried to send data to unknown peer: %d", peerID)
	}

	return conn.dataCh.Send(data)
}

func (pm *PeerManager) GetDataChOpenedCh() <-chan uint64 {
	return pm.dataChOpened
}

func (pm *PeerManager) GetPeerDataMsgCh() <-chan PeerDataMsg {
	return pm.peerData
}
