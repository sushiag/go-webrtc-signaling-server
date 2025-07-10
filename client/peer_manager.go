package client

import (
	"fmt"
	"log"
	"sync"

	"github.com/pion/webrtc/v4"
)

func newPeerManager(eventOutCh chan<- Event) *peerManager {
	pm := &peerManager{
		connections:  make(map[uint64]*pendingPeerConnection, 4),
		sdpCh:        make(chan sendSDP, 32),
		iceCh:        make(chan sendICECandidate, 32),
		msgOutCh:     make(chan WebRTCMsg, 30),
		peerEventsCh: eventOutCh,
	}

	return pm
}

func (pm *peerManager) newPeerOffer(peerID uint64) {
	log.Printf("[DEBUG] creating SDP offer for %d", peerID)

	conn, err := preparePeerConnCallbacks(peerID, pm.iceCh, pm.peerEventsCh)
	if err != nil {
		log.Fatalf("[ERROR] failed to create a new peer connection for %d: %v", peerID, err)
	}

	// Prepare data channel with callbacks
	dataCh, err := conn.CreateDataChannel("data", nil)
	if err != nil {
		log.Fatalf("[ERROR] failed to prepare the data channel for %d: %v", peerID, err)
	}
	prepareDataChCallbacks(peerID, dataCh, pm.peerEventsCh, pm.msgOutCh)

	pendingConn := &pendingPeerConnection{
		conn,
		make([]*webrtc.ICECandidate, 3),
		dataCh,
		sync.Mutex{},
	}
	pm.connections[peerID] = pendingConn

	// Send Offer
	offer, err := conn.CreateOffer(nil)
	if err != nil {
		log.Fatalf("[ERROR] failed to create SDP offer for %d: %v", peerID, err)
	}

	// NOTE: this will start the UDP listeners and gathering of ICE candidates
	if err := conn.SetLocalDescription(offer); err != nil {
		log.Fatalf("[ERROR] failed to set local description for %d: %v", peerID, err)
	}

	pm.sdpCh <- sendSDP{
		to:  peerID,
		sdp: offer,
	}
}

func (pm *peerManager) sendMsgToPeer(peerID uint64, msg string) error {
	conn, exists := pm.connections[peerID]
	if !exists {
		return fmt.Errorf("tried to send message to an unknown peer")
	}

	if conn.dataChannel == nil {
		return fmt.Errorf("data channel for peer %d is not yet initialized", peerID)
	}

	if conn.dataChannel.ReadyState() != webrtc.DataChannelStateOpen {
		return fmt.Errorf("data channel for peer %d is not yet open", peerID)
	}

	conn.dataChannel.SendText(msg)

	return nil
}

func (pm *peerManager) handleSDPOffer(peerID uint64, offer webrtc.SessionDescription) {
	log.Printf("[DEBUG] handling SDP offer from %d", peerID)

	conn, err := preparePeerConnCallbacks(peerID, pm.iceCh, pm.peerEventsCh)
	if err != nil {
		log.Fatalf("[ERROR] failed to create new peer connection for %d: %v", peerID, err)
	}

	if err != nil {
		log.Fatalf("[ERROR] failed to create a new data channel for %d: %v", peerID, err)
	}

	pendingConn := &pendingPeerConnection{

		conn,
		make([]*webrtc.ICECandidate, 3),
		nil,
		sync.Mutex{},
	}

	conn.OnDataChannel(func(dataCh *webrtc.DataChannel) {
		prepareDataChCallbacks(peerID, dataCh, pm.peerEventsCh, pm.msgOutCh)
		pendingConn.dataChannel = dataCh
	})

	if err := conn.SetRemoteDescription(offer); err != nil {
		log.Fatalf("[ERROR] failed to set remote description for %d: %v", peerID, err)
	}

	answer, err := conn.CreateAnswer(nil)
	if err != nil {
		log.Fatalf("[ERROR] failed to create SDP answer for %d: %v", peerID, err)
	}

	if err := conn.SetLocalDescription(answer); err != nil {
		log.Fatalf("[ERROR] failed to set local description for %d: %v", peerID, err)
	}

	pm.sdpCh <- sendSDP{
		peerID,
		answer,
	}

	pm.connections[peerID] = pendingConn
}

func preparePeerConnCallbacks(peerID uint64, iceSignalingCh chan<- sendICECandidate, peerEventCh chan<- Event) (*webrtc.PeerConnection, error) {
	conn, err := webrtc.NewPeerConnection(defaultWebRTCConfig)
	if err != nil {
		return conn, err
	}

	conn.OnICEGatheringStateChange(func(state webrtc.ICEGatheringState) {
		log.Printf("[DEBUG] ice gathering state for %d changed to: %s", peerID, state.String())
	})

	conn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("[DEBUG] connection state for %d changed to: %s", peerID, state.String())
		peerEventCh <- PeerConnectionStateChangedEvent{state}
	})

	conn.OnICECandidate(func(iceCandidate *webrtc.ICECandidate) {
		if iceCandidate == nil {
			return
		}

		log.Printf("[DEBUG] ice candidate for %d acquired", peerID)

		iceSignalingCh <- sendICECandidate{
			to:           peerID,
			iceCandidate: iceCandidate,
		}
	})

	return conn, nil
}

func prepareDataChCallbacks(peerID uint64, dataCh *webrtc.DataChannel, peerEventsCh chan<- Event, dataChOut chan<- WebRTCMsg) {
	dataCh.OnOpen(func() {
		log.Printf("[DEBUG] data channel for %d opened", peerID)
		peerEventsCh <- PeerDataChOpenedEvent{peerID}
	})

	dataCh.OnClose(func() {
		log.Printf("[DEBUG] data channel from %d was closed", peerID)
		peerEventsCh <- PeerDataChClosedEvent{peerID}
	})

	dataCh.OnError(func(err error) {
		log.Printf("[ERROR] failed to read data from %d", peerID)
	})

	dataCh.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("[DEBUG] received data from %d: %s", peerID, string(msg.Data))
		dataChOut <- WebRTCMsg{
			from: peerID,
			msg:  string(msg.Data),
		}
	})
}
