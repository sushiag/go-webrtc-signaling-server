package client

import "github.com/pion/webrtc/v4"

type Event any

type ServerPingEvent struct{}

type ServerPongEvent struct{}

type RoomCreatedEvent struct {
	RoomID uint64
}

type RoomJoinedEvent struct {
	RoomID        uint64
	ClientsInRoom []uint64
}

type PeerConnectionStateChangedEvent struct {
	NewState webrtc.PeerConnectionState
}

type PeerDataChOpenedEvent struct {
	PeerID uint64
}

type PeerDataChClosedEvent struct {
	PeerID uint64
}
