package peer_manager

import (
	"fmt"

	"github.com/pion/webrtc/v4"

	smsg "signaling-msgs"
)

// This represents the PeerManger, a main component that maintains the active WebRTC peers in a map.
// This sends/receives signaling messages (SDP AND ICE) -- as well as manage evevens for the data channels and icoming data from peers.
type PeerManager struct {
	peers        map[uint64]*peer
	signalingOut chan<- smsg.MessageAnyPayload
	dataChOpened chan uint64
	peerData     chan PeerDataMsg
}

// This represents the data messages from peers, which incluudes the peer sender's ID.
type PeerDataMsg struct {
	From uint64
	Data []byte
}

// This represents a single peer connection that includes both the PeerConnection and DataChannel.
type peer struct {
	conn   *webrtc.PeerConnection
	dataCh *webrtc.DataChannel
}

// This represents the internal tracker for the ICE candidates before sending to peers.
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

// This handles the sending for data from a peer to another peeer
func (pm *PeerManager) SendDataToPeer(peerID uint64, data []byte) error {
	conn, exists := pm.peers[peerID]
	if !exists {
		return fmt.Errorf("tried to send data to unknown peer: %d", peerID)
	}

	return conn.dataCh.Send(data)
}

// This handles the current status of a channel open for sending data/sdp/ICE
func (pm *PeerManager) GetDataChOpenedCh() <-chan uint64 {
	return pm.dataChOpened
}

// This handles the message sent from a peer to another peer.
func (pm *PeerManager) GetPeerDataMsgCh() <-chan PeerDataMsg {
	return pm.peerData
}
