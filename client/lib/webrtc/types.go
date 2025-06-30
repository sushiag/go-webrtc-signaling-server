package webrtc

import (
	"context"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
)

type Peer struct {
	ID                    uint64
	Connection            *webrtc.PeerConnection
	DataChannel           *webrtc.DataChannel
	bufferedICECandidates []webrtc.ICECandidateInit
	remoteDescriptionSet  bool

	ctx    context.Context
	cancel context.CancelFunc

	sendChan chan string
}

type pmEvent any

type pmCloseAll struct{}
type pmGetPeerIDs struct {
	resultCh chan []uint64
}
type pmCheckAllConnectedAndDisconnect struct {
	resultCh chan error
}
type pmWaitForDataChannel struct {
	peerID   uint64
	timeout  time.Duration
	resultCh chan error
}
type pmSendDataToPeer struct {
	peerID   uint64
	data     []byte
	resultCh chan error
}
type pmSendJSONToPeer struct {
	peerID   uint64
	payload  Payload
	resultCh chan error
}
type pmHandleIncomingMsg struct {
	msg      SignalingMessage
	sendFunc func(SignalingMessage) error // TODO: maybe not send a func here
}
type pmRemovePeer struct {
	peerID   uint64
	sendFunc func(SignalingMessage) error // TODO: maybe not send a func here
}
type pmHandleICECandidate struct {
	msg      SignalingMessage
	resultCh chan error
}
type pmCreateAndSendOffer struct {
	peerID   uint64
	sendFunc func(SignalingMessage) error // TODO: maybe not send a func here
	resultCh chan error
}
type pmHandleOffer struct {
	msg      SignalingMessage
	sendFunc func(SignalingMessage) error // TODO: maybe not send a func here
	resultCh chan error
}

type PeerManager struct {
	userID                uint64
	hostID                uint64
	peers                 map[uint64]*Peer
	config                webrtc.Configuration
	signalingMessage      SignalingMessage
	onPeerCreated         func(*Peer, SignalingMessage)
	managerQueue          chan func()
	sendSignalFunc        func(SignalingMessage) error
	iceCandidateBuffer    map[uint64][]webrtc.ICECandidateInit
	outgoingMessages      chan SignalingMessage
	pmEventCh             chan pmEvent
	processingLoopStarted bool
}

type SignalingMessage struct {
	Type      common.MessageType `json:"type"`
	Content   string             `json:"content,omitempty"`
	RoomID    uint64             `json:"room_id,omitempty"`
	Sender    uint64             `json:"from,omitempty"`
	Target    uint64             `json:"to,omitempty"`
	Candidate string             `json:"candidate,omitempty"`
	SDP       string             `json:"sdp,omitempty"`
	Users     []uint64           `json:"users,omitempty"`
	Text      string             `json:"text,omitempty"`
	Payload   Payload            `json:"payload,omitempty"`
}

type Payload struct {
	DataType string `json:"data_type"`
	Data     []byte `json:"data"`
}
