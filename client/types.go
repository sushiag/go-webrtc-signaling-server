package client

import (
	"sync"

	"github.com/pion/webrtc/v4"

	sm "signaling-msgs"
)

type Client struct {
	signalingMngr *signalingManager
	eventsCh      <-chan Event
}

type WebRTCMsg struct {
	from uint64
	msg  string
}

// Handles WebRTC Peers
type peerManager struct {
	connections  map[uint64]*pendingPeerConnection
	sdpCh        chan sendSDP
	iceCh        chan sendICECandidate
	msgOutCh     chan WebRTCMsg
	peerEventsCh chan<- Event
}

type pendingPeerConnection struct {
	conn              *webrtc.PeerConnection
	pendingCandidates []*webrtc.ICECandidate
	dataChannel       *webrtc.DataChannel
	candidatesMux     sync.Mutex
}

type signalingManager struct {
	clients    map[uint64]*peerManager
	wsClientID uint64
	// Channel for sending commands to the server
	wsSendCh chan<- sm.Message
}

type sendICECandidate struct {
	to           uint64
	iceCandidate *webrtc.ICECandidate
}

type sendSDP struct {
	to  uint64
	sdp webrtc.SessionDescription
}

var defaultWebRTCConfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{URLs: []string{"stun:stun.l.google.com:19302"}},
	},
}
