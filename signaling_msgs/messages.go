package ws_messages

import (
	"encoding/json"
)

type MessageType uint8

type Message struct {
	MsgType MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
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
	RoomID uint64 `json:"room_id"`
}

type SDPPayload struct {
	SDP string `json:"sdp"`
	For uint64 `json:"for"`
}

type ICECandidatePayload struct {
	ICE string `json:"ice"`
	For uint64 `json:"for"`
}
