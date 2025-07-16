package peer_manager

import (
	"fmt"
	"log"

	"github.com/pion/webrtc/v4"

	smsg "signaling-msgs"
)

func (pm *PeerManager) newPeerOffer(peerID uint64) error {
	if _, exists := pm.peers[peerID]; exists {
		return fmt.Errorf("the given peer ID is already taken: %d", peerID)
	}

	conn, createConnErr := webrtc.NewPeerConnection(defaultWebRTCConfig)
	if createConnErr != nil {
		return fmt.Errorf("failed to initialize peer connection for peer %d: %v", peerID, createConnErr)
	}

	dataCh, createDataChErr := conn.CreateDataChannel("data", nil)
	if createDataChErr != nil {
		return fmt.Errorf("failed to initialize data channel for peer %d: %v", peerID, createConnErr)
	}

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

	dataCh.OnOpen(func() {
		pm.dataChOpened <- peerID
	})

	dataCh.OnMessage(func(msg webrtc.DataChannelMessage) {
		pm.peerData <- PeerDataMsg{
			From: peerID,
			Data: msg.Data,
		}
	})

	offer, createOfferErr := conn.CreateOffer(nil)
	if createOfferErr != nil {
		return fmt.Errorf("failed to create SDP offer for peer %d: %v", peerID, createConnErr)
	}

	if err := conn.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("failed to set local description offer for peer %d: %v", peerID, createConnErr)
	}

	pm.peers[peerID] = &peer{
		conn:   conn,
		dataCh: dataCh,
	}

	pm.signalingOut <- smsg.MessageAnyPayload{
		MsgType: smsg.SDP,
		To:      peerID,
		Payload: smsg.SDPPayload{
			SDP: offer,
		},
	}

	return nil
}

func (pm *PeerManager) handlePeerOffer(peerID uint64, offer webrtc.SessionDescription) error {
	if _, exists := pm.peers[peerID]; exists {
		return fmt.Errorf("the given peer ID is already taken: %d", peerID)
	}

	conn, createConnErr := webrtc.NewPeerConnection(defaultWebRTCConfig)
	if createConnErr != nil {
		return fmt.Errorf("failed to initialize peer connection for peer %d: %v", peerID, createConnErr)
	}

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

	if err := conn.SetRemoteDescription(offer); err != nil {
		return fmt.Errorf("failed to set remote description offer for peer %d: %v", peerID, createConnErr)
	}

	answer, createAnswerErr := conn.CreateAnswer(nil)
	if createAnswerErr != nil {
		return fmt.Errorf("failed to create SDP answer for peer %d: %v", peerID, createConnErr)
	}

	if err := conn.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("failed to set local description for peer %d: %v", peerID, createConnErr)
	}

	pm.peers[peerID] = &peer{
		conn:   conn,
		dataCh: nil,
	}

	pm.signalingOut <- smsg.MessageAnyPayload{
		MsgType: smsg.SDP,
		To:      peerID,
		Payload: smsg.SDPPayload{
			SDP: answer,
		},
	}

	return nil
}

func (pm *PeerManager) handlePeerAnswer(peerID uint64, answer webrtc.SessionDescription) error {
	conn, exists := pm.peers[peerID]
	if !exists {
		return fmt.Errorf("tried to set remote SDP for nonexistent peer: %d", peerID)
	}

	return conn.conn.SetRemoteDescription(answer)
}

func (pm *PeerManager) addIceCandidate(peerID uint64, iceCandidate webrtc.ICECandidateInit) error {
	conn, exists := pm.peers[peerID]
	if !exists {
		return fmt.Errorf("tried to add ICE candidate for nonexistent peer: %d", peerID)
	}

	return conn.conn.AddICECandidate(iceCandidate)
}
