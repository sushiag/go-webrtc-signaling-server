package websocket

import (
	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
)

type Payload struct {
	DataType string `json:"data_type"`
	Data     []byte `json:"data"`
}

type Message struct {
	Type      common.MessageType `json:"type"`
	Content   string             `json:"content,omitempty"`
	RoomID    uint64             `json:"roomid,omitempty"`
	Sender    uint64             `json:"from,omitempty"`
	Target    uint64             `json:"to,omitempty"`
	Candidate string             `json:"candidate,omitempty"`
	SDP       string             `json:"sdp,omitempty"`
	Users     []uint64           `json:"users,omitempty"`
	Text      string             `json:"text,omitempty"`
	Payload   Payload            `json:"payload"`
	UserID    uint64             `json:"userid,omitempty"`
}
