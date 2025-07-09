package ws_messages

import (
	"encoding/json"
	"fmt"
	"log"
)

// Helper function for serializing message payloads to json.RawMessage
//
// NOTE: only use this for tests since this will panic if serialization fails!
func ToRawMessagePayload(payload any) json.RawMessage {
	msg, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("failed to marshal WS message")
	}

	return msg
}

func (ty MessageType) AsString() string {
	switch ty {
	case Ping:
		return "ping"
	case Pong:
		return "pong"
	case CreateRoom:
		return "create-room"
	case RoomCreated:
		return "room-created"
	case JoinRoom:
		return "join-room"
	case RoomJoined:
		return "room-joined"
	case LeaveRoom:
		return "leave-room"
	case SDP:
		return "sdp"
	case ICECandidate:
		return "ice-candidate"
	default:
		return fmt.Sprintf("unknown (%d)", ty)
	}
}
