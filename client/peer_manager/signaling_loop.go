package peer_manager

import (
	"encoding/json"
	"log"

	"github.com/pion/webrtc/v4"

	smsg "signaling-msgs"
)

// This handles the incoming signaling messages and routes it in the respectively
func (pm *PeerManager) signalingLoop(signalingIn <-chan smsg.MessageRawJSONPayload) {
	for msg := range signalingIn {
		switch msg.MsgType {
		// This handles the client when joining a room and then creates an SDP offer for each of the client active in the rooom
		case smsg.RoomJoined:
			{
				var payload smsg.RoomJoinedPayload
				if err := json.Unmarshal(msg.Payload, &payload); err != nil {
					log.Printf("[ERROR] failed to unmarshal room joined payload")
					continue
				}

				for _, clientID := range payload.ClientsInRoom {
					if err := pm.newPeerOffer(clientID); err != nil {
						log.Printf("[ERROR] failed to create SDP offer for %d: %v", clientID, err)
						continue
					}
				}
			}
		case smsg.SDP:
			{
				// This handles the incoming SDP message (offer/answer)
				var payload smsg.SDPPayload
				if err := json.Unmarshal(msg.Payload, &payload); err != nil {
					log.Printf("[ERROR] failed to unmarshal SDP payload")
					continue
				}

				switch payload.SDP.Type {
				case webrtc.SDPTypeOffer:
					{
						err := pm.handlePeerOffer(msg.From, payload.SDP)
						if err != nil {
							log.Printf("[ERROR] failed to handle peer offer: %v", err)
							break
						}
					}
				case webrtc.SDPTypeAnswer:
					{
						if err := pm.handlePeerAnswer(msg.From, payload.SDP); err != nil {
							log.Printf("[ERROR] failed to handle peer answer: %v", err)
						}
					}
				default:
					{
						log.Printf("[WARN] unhandled SDP type: %s", payload.SDP.Type.String())
					}
				}
			}
		case smsg.ICECandidate:
			{
				// This handle the incoming ICE Candites and add ii to the eixsting/offered peer connection
				var payload smsg.ICECandidatePayload
				if err := json.Unmarshal(msg.Payload, &payload); err != nil {
					log.Printf("[ERROR] failed to unmarshal ICE candidate payload")
					continue
				}

				pm.addIceCandidate(msg.From, payload.ICE)
			}
		}
	}
}
