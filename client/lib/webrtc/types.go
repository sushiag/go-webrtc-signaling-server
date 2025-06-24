package webrtc

import (
	"context"
	"encoding/json"

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
	iceCandidateBuffer map[uint64][]webrtc.ICECandidateInit
	outgoingMessages   chan SignalingMessage
}

type SignalingMessage struct {
	Type      MessageType `json:"type"`
	Content   string      `json:"content,omitempty"`
	RoomID    uint64      `json:"room_id,omitempty"`
	Sender    uint64      `json:"from,omitempty"`
	Target    uint64      `json:"to,omitempty"`
	Candidate string      `json:"candidate,omitempty"`
	SDP       string      `json:"sdp,omitempty"`
	Users     []uint64    `json:"users,omitempty"`
	Text      string      `json:"text,omitempty"`
	Payload   Payload     `json:"payload,omitempty"`
}

type Payload struct {
	DataType string `json:"data_type"`
	Data     []byte `json:"data"`
}

type MessageType int

const (
	MessageTypeOffer MessageType = iota
	MessageTypeAnswer
	MessageTypeICECandidate
	MessageTypePeerJoined
	MessageTypeDisconnect
	MessageTypeSendMessage
	MessageTypePeerList
	MessageTypeHostChanged
	MessageTypeStartSession
)

func (mt MessageType) String() string {
	switch mt {
	case MessageTypeOffer:
		return "offer"
	case MessageTypeAnswer:
		return "answer"
	case MessageTypeICECandidate:
		return "ice-candidate"
	case MessageTypePeerJoined:
		return "peer-joined"
	case MessageTypeDisconnect:
		return "disconnect"
	case MessageTypeSendMessage:
		return "send-message"
	case MessageTypePeerList:
		return "peer-list"
	case MessageTypeHostChanged:
		return "host-changed"
	case MessageTypeStartSession:
		return "start-session"
	default:
		return "unknown"
	}
}

func (mt *MessageType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	*mt = ParseMessageType(s)
	return nil
}

func (mt MessageType) MarshalJSON() ([]byte, error) {
	return json.Marshal(mt.String())
}

func ParseMessageType(s string) MessageType {
	switch s {
	case "offer":
		return MessageTypeOffer
	case "answer":
		return MessageTypeAnswer
	case "ice-candidate":
		return MessageTypeICECandidate
	case "peer-joined":
		return MessageTypePeerJoined
	case "disconnect":
		return MessageTypeDisconnect
	case "send-message":
		return MessageTypeSendMessage
	case "peer-list":
		return MessageTypePeerList
	case "host-changed":
		return MessageTypeHostChanged
	case "start-session":
		return MessageTypeStartSession
	default:
		return -1
	}
}
