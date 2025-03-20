package webrtc

import (
	"fmt"

	"github.com/sushiag/go-webrtc-signaling-server/internal/webrtc"

	"os"

	"github.com/joho/godotenv"
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
	config := []webrtc.Configuration{
		ICEServer: []webrtc.ICEServer{
			{URLs: []string{stunServer}},
		},
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %v", err)
	}
	return peerConnection, nil
}
