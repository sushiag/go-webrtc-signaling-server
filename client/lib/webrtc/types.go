package webrtc

import (
	"context"
	"sync"

	"github.com/pion/webrtc/v4"
)

type Peer struct {
	ID                    uint64
	Connection            *webrtc.PeerConnection
	DataChannel           *webrtc.DataChannel
	bufferedICECandidates []webrtc.ICECandidateInit
	remoteDescriptionSet  bool

	ctx        context.Context
	cancel     context.CancelFunc
	cancelOnce sync.Once
	sendChan   chan string
}

type PeerManager struct {
	UserID           uint64
	HostID           uint64
	Peers            sync.Map
	Config           webrtc.Configuration
	SignalingMessage SignalingMessage
	onPeerCreated    func(*Peer, SignalingMessage)
	managerQueue     chan func()
	sendSignalFunc   func(SignalingMessage) error
}

type SignalingMessage struct {
	Type      string   // type of message
	Content   string   // content
	RoomID    uint64   // room id
	Sender    uint64   // sender user id
	Target    uint64   // target user id
	Candidate string   // ice-candidate string
	SDP       string   // session description
	Users     []uint64 // list of user ids
	Text      string   // for send messages
	Payload   Payload  // "file", "text", "image"
}
type Payload struct {
	DataType string `json:"data_type"`
	Data     []byte `json:"data"`
}
