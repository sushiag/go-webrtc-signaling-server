package main

import (
	"log"

	"github.com/pion/webrtc/v4"
)

type PeerManager struct {
	peers map[string]*webrtc.PeerConnection
}

func NewPeerManager() *PeerManager {
	return &PeerManager{peers: make(map[string]*webrtc.PeerConnection)}
}

func (pm *PeerManager) Add(peerID string, peerConnection *webrtc.PeerConnection) {
	pm.peers[peerID] = peerConnection
}

func (pm *PeerManager) Get(peerID string) (*webrtc.PeerConnection, bool) {
	peerConnection, exists := pm.peers[peerID]
	return peerConnection, exists
}

func (pm *PeerManager) Remove(peerID string) {
	delete(pm.peers, peerID)
}

func NewPeerConnection() (*webrtc.PeerConnection, error) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Println("[ICE STATE]", state.String())
	})

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		log.Printf("[DATA CHANNEL] Incoming: %s", dc.Label())

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("[DATA CHANNEL] Received %d bytes", len(msg.Data))
			// Here’s where you’d write the file to disk or stream
		})
	})

	return pc, nil
}
