package ws_messages

import (
	"encoding/json"

	"github.com/pion/webrtc/v4"
)

type MessageType uint8

type MessageAnyPayload struct {
	MsgType MessageType `json:"type"`
	To      uint64      `json:"to,omitempty"`
	From    uint64      `json:"from,omitempty"`
	Payload any         `json:"payload,omitempty"`
}

type MessageRawJSONPayload struct {
	MsgType MessageType     `json:"type"`
	To      uint64          `json:"to,omitempty"`
	From    uint64          `json:"from,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

const (
	Ping MessageType = iota
	Pong
	CreateRoom
	RoomCreated
	JoinRoom
	RoomJoined
	LeaveRoom
	SDP
	ICECandidate
)

type RoomCreatedPayload struct {
	RoomID uint64 `json:"room_id"`
}

type JoinRoomPayload struct {
	RoomID uint64 `json:"room_id"`
}

type RoomJoinedPayload struct {
	RoomID        uint64   `json:"room_id"`
	ClientsInRoom []uint64 `json:"clients"`
}

type SDPPayload struct {
	SDP webrtc.SessionDescription `json:"sdp"`
}

type ICECandidatePayload struct {
	ICE webrtc.ICECandidateInit `json:"ice"`
}
