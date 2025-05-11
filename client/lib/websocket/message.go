package websocket

type Payload struct {
	DataType string `json:"data_type"`
	Data     []byte `json:"data"`
}

type Message struct {
	Type      string   `json:"type"`
	Content   string   `json:"content,omitempty"`
	RoomID    uint64   `json:"roomid,omitempty"`
	Sender    uint64   `json:"from,omitempty"`
	Target    uint64   `json:"to,omitempty"`
	Candidate string   `json:"candidate,omitempty"`
	SDP       string   `json:"sdp,omitempty"`
	Users     []uint64 `json:"users,omitempty"`
	Text      string   `json:"text,omitempty"`
	Payload   Payload  `json:"Payload,omitempty"`
}

const (
	MessageTypeCreateRoom   = "create-room"
	MessageTypeRoomCreated  = "room-created"
	MessageTypeJoinRoom     = "join-room"
	MessageTypeRoomJoined   = "room-joined"
	MessageTypeOffer        = "offer"
	MessageTypeAnswer       = "answer"
	MessageTypeICECandidate = "ice-candidate"
	MessageTypeDisconnect   = "disconnect"
	MessageTypePeerJoined   = "peer-joined"
	MessageTypePeerListReq  = "peer-list-request"
	MessageTypePeerList     = "peer-list"
	MessageTypeStartSession = "start-session"
	MessageTypeSendMessage  = "send-message"
	MessageTypeHostChanged  = "host-changed"
)
