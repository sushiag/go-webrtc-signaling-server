package webrtc

import (
	"context"

	"github.com/pion/webrtc/v4"
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

type PeerManager struct {
	UserID             uint64
	HostID             uint64
	Peers              map[uint64]*Peer
	Config             webrtc.Configuration
	SignalingMessage   SignalingMessage
	onPeerCreated      func(*Peer, SignalingMessage)
	managerQueue       chan func()
	sendSignalFunc     func(SignalingMessage) error
	iceCandidateBuffer map[uint64][]webrtc.ICECandidateInit // NEW
}

type SignalingMessage struct {
	Type      string   `json:"type"`
	Content   string   `json:"content,omitempty"`
	RoomID    uint64   `json:"room_id,omitempty"`
	Sender    uint64   `json:"from,omitempty"`
	Target    uint64   `json:"to,omitempty"`
	Candidate string   `json:"candidate,omitempty"`
	SDP       string   `json:"sdp,omitempty"`
	Users     []uint64 `json:"users,omitempty"`
	Text      string   `json:"text,omitempty"`
	Payload   Payload  `json:"payload,omitempty"`
}

type Payload struct {
	DataType string `json:"data_type"`
	Data     []byte `json:"data"`
}
