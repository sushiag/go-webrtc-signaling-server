package client

import (
	"sync"

	"github.com/pion/webrtc/v4"
)

type WebRTCMsg struct {
	from uint64
	msg  string
}

type webRTCPeerManager struct {
	clientID     uint64
	connections  map[uint64]*pendingPeerConnection
	sdpCh        chan<- sdpSignalingRequest
	iceCh        chan<- iceSignalingRequest
	dataChOpened chan uint64
	msgOutCh     chan WebRTCMsg
}

type pendingPeerConnection struct {
	conn              *webrtc.PeerConnection
	pendingCandidates []*webrtc.ICECandidate
	dataChannel       *webrtc.DataChannel
	candidatesMux     sync.Mutex
}

type signalingManager struct {
	clients          map[uint64]*webRTCPeerManager
	sdpSignalingCh   chan sdpSignalingRequest
	iceSignalingCh   chan iceSignalingRequest
	wsClientID       uint64
	wsSendCh         chan<- WSMessage
	signalingEventCh <-chan SignalingEvent
}

type iceSignalingRequest struct {
	to           uint64
	iceCandidate *webrtc.ICECandidate
}

type sdpSignalingRequest struct {
	to  uint64
	sdp webrtc.SessionDescription
}

var defaultWebRTCConfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{URLs: []string{"stun:stun.l.google.com:19302"}},
	},
}
