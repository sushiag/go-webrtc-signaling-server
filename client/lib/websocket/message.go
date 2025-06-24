package websocket

import (
	"encoding/json"
)

type Payload struct {
	DataType string `json:"data_type"`
	Data     []byte `json:"data"`
}

type Message struct {
	Type      MessageType `json:"type"`
	Content   string      `json:"content,omitempty"`
	RoomID    uint64      `json:"roomid,omitempty"`
	Sender    uint64      `json:"from,omitempty"`
	Target    uint64      `json:"to,omitempty"`
	Candidate string      `json:"candidate,omitempty"`
	SDP       string      `json:"sdp,omitempty"`
	Users     []uint64    `json:"users,omitempty"`
	Text      string      `json:"text,omitempty"`
	Payload   Payload     `json:"Payload,omitempty"`
}

type MessageType int

const (
	MessageTypeInvalid MessageType = iota
	MessageTypeCreateRoom
	MessageTypeRoomCreated
	MessageTypeJoinRoom
	MessageTypeRoomJoined
	MessageTypeOffer
	MessageTypeAnswer
	MessageTypeICECandidate
	MessageTypeDisconnect
	MessageTypePeerJoined
	MessageTypePeerListRequest
	MessageTypePeerList
	MessageTypeStartSession
	MessageTypeSendMessage
	MessageTypeHostChanged
)

func (t MessageType) String() string {
	switch t {
	case MessageTypeCreateRoom:
		return "create-room"
	case MessageTypeRoomCreated:
		return "room-created"
	case MessageTypeJoinRoom:
		return "join-room"
	case MessageTypeRoomJoined:
		return "room-joined"
	case MessageTypeOffer:
		return "offer"
	case MessageTypeAnswer:
		return "answer"
	case MessageTypeICECandidate:
		return "ice-candidate"
	case MessageTypeDisconnect:
		return "disconnect"
	case MessageTypePeerJoined:
		return "peer-joined"
	case MessageTypePeerListRequest:
		return "peer-list-request"
	case MessageTypePeerList:
		return "peer-list"
	case MessageTypeStartSession:
		return "start-session"
	case MessageTypeSendMessage:
		return "send-message"
	case MessageTypeHostChanged:
		return "host-changed"
	default:
		return "invalid"
	}
}

func ParseMessageType(s string) MessageType {
	switch s {
	case "create-room":
		return MessageTypeCreateRoom
	case "room-created":
		return MessageTypeRoomCreated
	case "join-room":
		return MessageTypeJoinRoom
	case "room-joined":
		return MessageTypeRoomJoined
	case "offer":
		return MessageTypeOffer
	case "answer":
		return MessageTypeAnswer
	case "ice-candidate":
		return MessageTypeICECandidate
	case "disconnect":
		return MessageTypeDisconnect
	case "peer-joined":
		return MessageTypePeerJoined
	case "peer-list-request":
		return MessageTypePeerListRequest
	case "peer-list":
		return MessageTypePeerList
	case "start-session":
		return MessageTypeStartSession
	case "send-message":
		return MessageTypeSendMessage
	case "host-changed":
		return MessageTypeHostChanged

	default:
		return MessageTypeInvalid
	}
}

// Marshal as string to JSON
func (t MessageType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// Unmarshal from string to enum
func (t *MessageType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*t = ParseMessageType(s)
	return nil
}
