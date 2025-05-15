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
	cmdChan               chan PeerCommand

	ctx         context.Context
	cancel      context.CancelFunc
	cancelOnce  sync.Once
	sendChan    chan string
	isConnected bool
}

type PeerManager struct {
	UserID           uint64
	HostID           uint64
	Peers            sync.Map
	Config           webrtc.Configuration
	Mutex            sync.Mutex
	SignalingMessage SignalingMessage
	onPeerCreated    func(*Peer, SignalingMessage)
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

type PeerCommand struct {
	Command string
	Data    interface{}
	Action  func(p *Peer)
}

/*****
func (p *Peer) run() {
	for {
		select {
		case cmd := <-p.cmdChan:
			cmd.Action(p)
		case <-p.ctx.Done():
			return
		}
	}
}***/
