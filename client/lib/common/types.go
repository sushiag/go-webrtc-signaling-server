package common

import (
	"encoding/json"
	"fmt"
)

type WebRTCMessage struct {
	From uint64
	Data []byte
}

type PeerEvent any

type PeerDataChOpened struct {
	PeerID uint64
}

type MessageType int

const (
	MessageTypeOffer MessageType = iota
	MessageTypeAnswer
	MessageTypeICECandidate
	MessageTypePeerJoined
	MessageTypePeerLeft
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
	MessageTypeSetUserID
)

func (mt MessageType) ToString() (string, error) {
	var convertedType string
	var err error

	switch mt {
	case MessageTypeOffer:
		convertedType = "offer"
	case MessageTypeAnswer:
		convertedType = "answer"
	case MessageTypeICECandidate:
		convertedType = "ice-candidate"
	case MessageTypePeerJoined:
		convertedType = "peer-joined"
	case MessageTypePeerLeft:
		convertedType = "peer-left"
	case MessageTypeDisconnect:
		convertedType = "disconnect"
	case MessageTypeSendMessage:
		convertedType = "send-message"
	case MessageTypePeerList:
		convertedType = "peer-list"
	case MessageTypeHostChanged:
		convertedType = "host-changed"
	case MessageTypeStartSession:
		convertedType = "start-session"
	case MessageTypeRoomCreated:
		convertedType = "room-created"
	case MessageTypeCreateRoom:
		convertedType = "create-room"
	case MessageTypeJoinRoom:
		convertedType = "join-room"
	case MessageTypePeerListReq:
		convertedType = "peer-list-req"
	case MessageTypeRoomJoined:
		convertedType = "room-joined"
	case MessageTypeSetUserID:
		convertedType = "set-user-id"
	default:
		convertedType = fmt.Sprintf("unknown (%d)", mt)
		err = fmt.Errorf(convertedType)
	}

	return convertedType, err
}

func (mt MessageType) MarshalJSON() ([]byte, error) {
	stringMt, err := mt.ToString()
	if err != nil {
		return nil, err
	}

	return json.Marshal(stringMt)
}

// TODO: We're overloading the normal UnmarshalJSON function here but once the server
// also starts using ints as a message type, this will no longer be necessary
func (mt *MessageType) UnmarshalJSON(b []byte) error {
	var parsed string
	var err error

	if err = json.Unmarshal(b, &parsed); err != nil {
		return err
	}

	var converted MessageType
	converted, err = ParseMessageType(parsed)
	if err != nil {
		return err
	}

	*mt = converted
	return err
}

func ParseMessageType(s string) (MessageType, error) {
	var parsedType MessageType
	var err error
	switch s {
	case "create-room":
		parsedType = MessageTypeCreateRoom
	case "join-room":
		parsedType = MessageTypeJoinRoom
	case "offer":
		parsedType = MessageTypeOffer
	case "answer":
		parsedType = MessageTypeAnswer
	case "ice-candidate":
		parsedType = MessageTypeICECandidate
	case "disconnect":
		parsedType = MessageTypeDisconnect
	case "peer-joined":
		parsedType = MessageTypePeerJoined
	case "room-created":
		parsedType = MessageTypeRoomCreated
	case "peer-list":
		parsedType = MessageTypePeerList
	case "start-session":
		parsedType = MessageTypeStartSession
	case "peer-list-request":
		parsedType = MessageTypePeerListReq
	case "host-changed":
		parsedType = MessageTypeHostChanged
	case "send-message":
		parsedType = MessageTypeSendMessage
	case "peer-left":
		parsedType = MessageTypePeerLeft
	case "set-user-id":
		parsedType = MessageTypeSetUserID
	case "room-joined":
		parsedType = MessageTypeRoomJoined
	default:
		err = fmt.Errorf("invalid message type: '%s'", s)
	}

	return parsedType, err
}
