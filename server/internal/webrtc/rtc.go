package webrtc

import (
	"encoding/json"
	"log"
	"server/internal/websocket"

	"github.com/pion/webrtc/v4"
)

// peer to peer connection for client
func InitializePeerConnection(wm *websocket.WebSocketManager, roomID uint64, userID uint64) (*webrtc.PeerConnection, error) {
	// Define the STUN server to be used
	stunServer := "stun:stun.1.google.com:19302"
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{stunServer}}, // default stun server
		},
	}

	// new peer connection with the config
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	// ice candidates
	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			// Marshal ICE candidate to JSON
			candidateJSON, err := json.Marshal(c.ToJSON())
			if err != nil {
				log.Printf("[ERROR] Failed to marshal ICE candidate: %v", err)
				return
			}

			//  send message to websocket
			msg := websocket.Message{
				Type:    "ice-candidate",
				RoomID:  roomID,
				Sender:  userID,
				Content: string(candidateJSON),
			}

			// sending ice candidate to all users in the same room
			wm.SendToRoom(roomID, userID, msg)
		}
	})

	// negotiation needed event
	peerConnection.OnNegotiationNeeded(func() {
		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			log.Println("[ERROR] Failed to create offer:", err)
			return
		}

		// local description for the offer
		err = peerConnection.SetLocalDescription(offer)
		if err != nil {
			log.Println("[ERROR] Failed to set local description:", err)
			return
		}

		// offer to all users in the room
		msg := websocket.Message{
			Type:    "offer",
			RoomID:  roomID,
			Sender:  userID,
			Content: offer.SDP,
		}
		wm.SendToRoom(roomID, userID, msg)
	})

	// this handles any type of track
	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("New track received: %s", track.Kind())
	})

	// for state of change
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[ICE] Connection state changed to: %s", state.String())
		if state == webrtc.ICEConnectionStateFailed {
			log.Println("[ERROR] ICE connection failed, consider restarting the connection.")
		}
	})

	// return int peer connection
	return peerConnection, nil
}
