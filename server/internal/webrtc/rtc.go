package webrtc

import (
	"encoding/json"
	"log"

	"server/internal/websocket" // Correct import path for your websocket package

	"github.com/pion/webrtc/v4"
)

type Message struct {
	Type    string `json:"type"`
	RoomID  string `json:"roomID"`
	Content string `json:"content"`
}

// InitializePeerConnection sets up a WebRTC peer connection for a client
func InitializePeerConnection(conn *websocket.Conn, roomID string) (*webrtc.PeerConnection, error) {
	stunServer := "stun:stun.1.google.com:19302"
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{stunServer}},
		},
	}

	// Create a new peer connection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	// Handle ICE candidates
	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			// Convert ICE candidate to JSON
			candidateJSON, err := json.Marshal(c.ToJSON())
			if err != nil {
				log.Printf("[ERROR] Failed to marshal ICE candidate: %v", err)
				return
			}

			// Create message to send to WebSocket
			msg := Message{
				Type:    "ice-candidate",
				RoomID:  roomID,
				Content: string(candidateJSON),
			}

			// Send message over WebSocket
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("[ERROR] Failed to send ICE candidate: %v", err)
			}
		}
	})

	// Handle negotiation needed event
	peerConnection.OnNegotiationNeeded(func() {
		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			log.Println("[ERROR] Failed to create offer:", err)
			return
		}

		err = peerConnection.SetLocalDescription(offer)
		if err != nil {
			log.Println("[ERROR] Failed to set local description", err)
			return
		}

		// Send the offer via WebSocket
		msg := Message{
			Type:    "offer",
			RoomID:  roomID,
			Content: offer.SDP,
		}
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("[ERROR] Failed to send offer: %v", err)
		}
	})

	// Handle incoming tracks (e.g., video or audio)
	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("New track received: %s", track.Kind())
	})

	// Handle ICE connection state changes
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[ICE] Connection state changed to: %s", state.String())
	})

	return peerConnection, nil
}
