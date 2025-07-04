package webrtc

import (
	"context"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
	ws "github.com/sushiag/go-webrtc-signaling-server/client/lib/websocket"
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

type pmGetPeerIDs struct {
	resultCh chan []uint64
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
	msg        SignalingMessage
	responseCh chan ws.Message
}
type pmRemovePeer struct {
	peerID     uint64
	responseCh chan ws.Message
}

type PeerManager struct {
	UserID                uint64
	hostID                uint64
	peers                 map[uint64]*Peer
	config                webrtc.Configuration
	signalingMessage      SignalingMessage
	onPeerCreated         func(*Peer, SignalingMessage)
	managerQueue          chan func()
	iceCandidateBuffer    map[uint64][]webrtc.ICECandidateInit
	pmEventCh             chan pmEvent
	processingLoopStarted bool
	msgOutCh              chan common.WebRTCMessage
	PeerEventsCh          chan common.PeerEvent
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
	UserID    uint64             `json:"userid,omitempty"`
}

type Payload struct {
	DataType string `json:"data_type"`
	Data     []byte `json:"data"`
}
