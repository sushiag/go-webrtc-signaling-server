package webrtc

import (
	"encoding/json"
	"fmt"
	"log"

	"os"

	"github.com/joho/godotenv"
	"github.com/pion/webrtc/v4"
)

func LoadSTUNServer() string {
	_ = godotenv.Load() // loads the env variables
	stunServer := os.Getenv("STUN_SERVER")
	if stunServer == "" {
		stunServer = "stun:stun.1.google.com:19302" // default google stun server
	}
	return stunServer
}

func InitializePeerConnection() (*webrtc.PeerConnection, error) {
	stunServer := LoadSTUNServer()
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{stunServer}},
		},
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %v", err)
	}

	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			candidate := c.ToJSON()
			candidateJSON, _ := json.Marshal(candidate)
			log.Println("[ICE] New candidate:", string(candidateJSON))
		}
	})

	peerConnection.OnNegotiationNeeded(func() {
		log.Println("[SDP] Negotiation needed")
	})

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Println("[WebRTC] New track received:", track.Kind())
	})

	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("[STATE] Connection state changed to: %s", state.String())
	})

	return peerConnection, nil
}
