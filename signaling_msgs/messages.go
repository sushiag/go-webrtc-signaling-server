package ws_messages

import (
	"encoding/json"

	"github.com/pion/webrtc/v4"
)

type MessageType uint8

type MessageAnyPayload struct {
	MsgType MessageType `json:"type"`
	Payload any         `json:"payload,omitempty"`
}

type MessageRawJSONPayload struct {
	MsgType MessageType     `json:"type"`
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
	SDP  webrtc.SessionDescription `json:"sdp"`
	From uint64                    `json:"from,omitempty"`
	To   uint64                    `json:"to,omitempty"`
}

type ICECandidatePayload struct {
	ICE  *webrtc.ICECandidate `json:"ice"`
	From uint64               `json:"from,omitempty"`
	To   uint64               `json:"to,omitempty"`
}
