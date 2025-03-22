package webrtc

import (
	"encoding/json"
	"fmt"
	"log"

	"os"

	"github.com/joho/godotenv"
	"github.com/pion/webrtc/v4"
	"github.com/sushiag/go-webrtc-signaling-server/internal/websocket"
)

func LoadSTUNServer() string {
	_ = godotenv.Load() // loads the env variables
	stunServer := os.Getenv("STUN_SERVER")
	if stunServer == "" {
		stunServer = "stun:stun.1.google.com:19302" // default google stun server
	}
	return stunServer
}

func InitializePeerConnection(wm *websocket.WebSocketManager, roomID, clientID string) (*webrtc.PeerConnection, error) {
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

			message := websocket.Message{
				Type:    "ice-candidate",
				RoomID:  roomID,
				Sender:  clientID,
				Content: string(candidateJSON),
			}
			wm.SendToRoom(roomID, clientID, message)
		}
	})

	peerConnection.OnNegotiationNeeded(func() {
		log.Println("[SDP] Negotiation needed")
	})

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Println("[WebRTC] New track received:", track.Kind())
	})

	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[ICE] Connection state changed to: %s", state.String())

		switch state {
		case webrtc.ICEConnectionStateFailed:
			log.Println("[ICE] connection failed. restart attemping:")
			offer, err := peerConnection.CreateOffer(&webrtc.OfferOptions{ICERestart: true})
			if err != nil {
				log.Println("[ICE] failed to create restart offer:", err)
				return
			}
			peerConnection.SetLocalDescription(offer)
			wm.SendToRoom(roomID, clientID, websocket.Message{
				Type:    "offer",
				RoomID:  roomID,
				Sender:  clientID,
				Content: string(offer.SDP),
			})
		case webrtc.ICEConnectionStateDisconnected:
			log.Println("[ICE] Disconnected, checking if reconnection is possible..")

		}

	})

	return peerConnection, nil
}
