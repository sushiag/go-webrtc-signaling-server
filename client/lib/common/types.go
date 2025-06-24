package common

type MessageType int

const (
	MessageTypeCreateRoom MessageType = iota
	MessageTypeRoomCreated
	MessageTypeJoinRoom
	MessageTypeRoomJoined
	MessageTypeOffer
	MessageTypeAnswer
	MessageTypeICECandidate
	MessageTypeDisconnect
	MessageTypePeerJoined
	MessageTypePeerListReq
	MessageTypePeerList
	MessageTypeStartSession
	MessageTypeSendMessage
	MessageTypeHostChanged
)

func (m MessageType) String() string {
	switch m {
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
	case MessageTypePeerListReq:
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
		return "unknown"
	}
}
