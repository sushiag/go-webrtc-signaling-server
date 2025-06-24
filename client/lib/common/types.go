package common

import (
	"encoding/json"
)

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
	MessageTypeRoomCreated
	MessageTypeCreateRoom
	MessageTypeJoinRoom
	MessageTypePeerListReq
	MessageTypeRoomJoined
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
	case MessageTypeRoomCreated:
		return "room-created"
	case MessageTypeCreateRoom:
		return "create-room"
	case MessageTypeJoinRoom:
		return "join-room"
	case MessageTypePeerListReq:
		return "peer-list-req"
	case MessageTypeRoomJoined:
		return "room-joined"
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
	case "room-created":
		return MessageTypeRoomCreated
	case "create-room":
		return MessageTypeCreateRoom
	case "join-room":
		return MessageTypeJoinRoom
	case "peer-list-req":
		return MessageTypePeerListReq
	case "room-joined":
		return MessageTypeRoomJoined
	default:
		return -1
	}
}
