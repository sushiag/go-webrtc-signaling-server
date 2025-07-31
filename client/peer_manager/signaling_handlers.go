package peer_manager

import (
	"fmt"
	"log"

	"github.com/pion/webrtc/v4"

	smsg "signaling-msgs"
)

// This initates an offer for a connection to active peers in a room and creates channels
// This generates and send an SDP offer to active peer in the room.
func (pm *PeerManager) newPeerOffer(peerID uint64) error {

	// This preventss any duplications
	if _, exists := pm.peers[peerID]; exists {
		return fmt.Errorf("the given peer ID is already taken: %d", peerID)
	}

	// This creates peer connection to another peer
	conn, createConnErr := webrtc.NewPeerConnection(defaultWebRTCConfig)
	if createConnErr != nil {
		return fmt.Errorf("failed to initialize peer connection for peer %d: %v", peerID, createConnErr)
	}

	// This create a data channel on an existing connection
	dataCh, createDataChErr := conn.CreateDataChannel("data", nil)
	if createDataChErr != nil {
		return fmt.Errorf("failed to initialize data channel for peer %d: %v", peerID, createConnErr)
	}

	// This sends the ICE candidate
	conn.OnICECandidate(func(iceCandidate *webrtc.ICECandidate) {
		if iceCandidate == nil {
			return
		}

		pm.signalingOut <- smsg.MessageAnyPayload{
			MsgType: smsg.ICECandidate,
			To:      peerID,
			Payload: smsg.ICECandidatePayload{ICE: iceCandidate.ToJSON()},
		}
	})

	// This notify Peer if the data channel is open
	dataCh.OnOpen(func() {
		pm.dataChOpened <- peerID
	})

	// This sends messages over the data channel
	dataCh.OnMessage(func(msg webrtc.DataChannelMessage) {
		pm.peerData <- PeerDataMsg{
			From: peerID,
			Data: msg.Data,
		}
	})

	// This creates the SDP offer
	offer, createOfferErr := conn.CreateOffer(nil)
	if createOfferErr != nil {
		return fmt.Errorf("failed to create SDP offer for peer %d: %v", peerID, createConnErr)
	}

	if err := conn.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("failed to set local description offer for peer %d: %v", peerID, createConnErr)
	}

	// This Register the peer connection
	pm.peers[peerID] = &peer{
		conn:   conn,
		dataCh: dataCh,
	}

	// This sends the SDP to a remote peer
	pm.signalingOut <- smsg.MessageAnyPayload{
		MsgType: smsg.SDP,
		To:      peerID,
		Payload: smsg.SDPPayload{
			SDP: offer,
		},
	}

	return nil
}

// This handles an incoming SDP offer from a remote peer.
// Creates a peer connection then sets the remote description to create an response then sends it back to the peer.
func (pm *PeerManager) handlePeerOffer(peerID uint64, offer webrtc.SessionDescription) error {

	if _, exists := pm.peers[peerID]; exists {
		return fmt.Errorf("the given peer ID is already taken: %d", peerID)
	}

	// This creates a new peer connection
	conn, createConnErr := webrtc.NewPeerConnection(defaultWebRTCConfig)
	if createConnErr != nil {
		return fmt.Errorf("failed to initialize peer connection for peer %d: %v", peerID, createConnErr)
	}

	// This handles the ICE candidates being sent over
	conn.OnICECandidate(func(iceCandidate *webrtc.ICECandidate) {
		if iceCandidate == nil {
			return
		}

		pm.signalingOut <- smsg.MessageAnyPayload{
			MsgType: smsg.ICECandidate,
			To:      peerID,
			Payload: smsg.ICECandidatePayload{ICE: iceCandidate.ToJSON()},
		}
	})

	// This handles the ICE Candites
	conn.OnDataChannel(func(dataCh *webrtc.DataChannel) {
		peer, exists := pm.peers[peerID]
		if !exists {
			log.Printf("[ERROR] data channel for a peer without a opened: %d", peerID)
		}

		dataCh.OnOpen(func() {
			pm.dataChOpened <- peerID
		})

		dataCh.OnMessage(func(msg webrtc.DataChannelMessage) {
			pm.peerData <- PeerDataMsg{
				From: peerID,
				Data: msg.Data,
			}
		})

		peer.dataCh = dataCh
	})

	// This creates a reponse for the remote peer
	if err := conn.SetRemoteDescription(offer); err != nil {
		return fmt.Errorf("failed to set remote description offer for peer %d: %v", peerID, createConnErr)
	}

	// This creates a response for the local description of a peer
	answer, createAnswerErr := conn.CreateAnswer(nil)
	if createAnswerErr != nil {
		return fmt.Errorf("failed to create SDP answer for peer %d: %v", peerID, createConnErr)
	}

	if err := conn.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("failed to set local description for peer %d: %v", peerID, createConnErr)
	}

	// This register the peer connection on the data channel
	pm.peers[peerID] = &peer{
		conn:   conn,
		dataCh: nil,
	}

	// This sends back the response to the offerer peer
	pm.signalingOut <- smsg.MessageAnyPayload{
		MsgType: smsg.SDP,
		To:      peerID,
		Payload: smsg.SDPPayload{
			SDP: answer,
		},
	}

	return nil
}

// This handles the peer response that will be received from the offered peeer
// This is used after a peer sent an offer and is receiving from the peer
func (pm *PeerManager) handlePeerAnswer(peerID uint64, answer webrtc.SessionDescription) error {
	conn, exists := pm.peers[peerID]
	if !exists {
		return fmt.Errorf("tried to set remote SDP for nonexistent peer: %d", peerID)
	}

	return conn.conn.SetRemoteDescription(answer)
}

// This applies a remote ICE candidate that will be receive from the offered peer
func (pm *PeerManager) addIceCandidate(peerID uint64, iceCandidate webrtc.ICECandidateInit) error {
	conn, exists := pm.peers[peerID]
	if !exists {
		return fmt.Errorf("tried to add ICE candidate for nonexistent peer: %d", peerID)
	}

	return conn.conn.AddICECandidate(iceCandidate)
}
