package client

import (
	"fmt"
	"log"
	"sync"

	"github.com/pion/webrtc/v4"
)

func newPeerManager(
	sdpSignalingCh chan<- sdpSignalingRequest,
	iceSignalingCh chan<- iceSignalingRequest,
	clientID uint64,
) *webRTCPeerManager {
	return &webRTCPeerManager{
		clientID:     clientID,
		connections:  make(map[uint64]*pendingPeerConnection, 4),
		sdpCh:        sdpSignalingCh,
		iceCh:        iceSignalingCh,
		dataChOpened: make(chan uint64, 4),
		msgOutCh:     make(chan WebRTCMsg, 30),
	}
}

func (pm *webRTCPeerManager) newPeerOffer(peerID uint64) {
	log.Printf("[DEBUG] peer %d: creating SDP offer for %d", pm.clientID, peerID)

	conn, err := preparePeerConnCallbacks(pm.clientID, peerID, pm.iceCh)
	if err != nil {
		log.Fatalf("[ERROR] client %d: failed to create a new peer connection for %d: %v", pm.clientID, peerID, err)
	}

	// Prepare data channel with callbacks
	dataCh, err := conn.CreateDataChannel("data", nil)
	if err != nil {
		log.Fatalf("[ERROR] client %d: failed to prepare the data channel for %d: %v", pm.clientID, peerID, err)
	}
	prepareDataChCallbacks(pm.clientID, peerID, dataCh, pm.dataChOpened, pm.msgOutCh)

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
		log.Fatalf("[ERROR] client %d: failed to create SDP offer for %d: %v", pm.clientID, peerID, err)
	}

	// NOTE: this will start the UDP listeners and gathering of ICE candidates
	if err := conn.SetLocalDescription(offer); err != nil {
		log.Fatalf("[ERROR] client %d: failed to set local description for %d: %v", pm.clientID, peerID, err)
	}

	pm.sdpCh <- sdpSignalingRequest{
		to:  peerID,
		sdp: offer,
	}
}

func (pm *webRTCPeerManager) sendMsgToPeer(peerID uint64, msg string) error {
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

func (pm *webRTCPeerManager) handleSDPOffer(peerID uint64, offer webrtc.SessionDescription) {
	log.Printf("[INFO] client %d: handling SDP offer from %d", pm.clientID, peerID)

	conn, err := preparePeerConnCallbacks(pm.clientID, peerID, pm.iceCh)
	if err != nil {
		log.Fatalf("[ERROR] client %d: failed to create new peer connection for %d: %v", pm.clientID, peerID, err)
	}

	if err != nil {
		log.Fatalf("[ERROR] client %d: failed to create a new data channel for %d: %v", pm.clientID, peerID, err)
	}

	pendingConn := &pendingPeerConnection{

		conn,
		make([]*webrtc.ICECandidate, 3),
		nil,
		sync.Mutex{},
	}

	conn.OnDataChannel(func(dataCh *webrtc.DataChannel) {
		prepareDataChCallbacks(pm.clientID, peerID, dataCh, pm.dataChOpened, pm.msgOutCh)
		pendingConn.dataChannel = dataCh
	})

	if err := conn.SetRemoteDescription(offer); err != nil {
		log.Fatalf("[ERROR] client %d: failed to set remote description for %d: %v", pm.clientID, peerID, err)
	}

	answer, err := conn.CreateAnswer(nil)
	if err != nil {
		log.Fatalf("[ERROR] client %d: failed to create SDP answer for %d: %v", pm.clientID, peerID, err)
	}

	if err := conn.SetLocalDescription(answer); err != nil {
		log.Fatalf("[ERROR] client %d: failed to set local description for %d: %v", pm.clientID, peerID, err)
	}

	pm.sdpCh <- sdpSignalingRequest{
		peerID,
		answer,
	}

	pm.connections[peerID] = pendingConn
}

func preparePeerConnCallbacks(clientID uint64, peerID uint64, iceSignalingCh chan<- iceSignalingRequest) (*webrtc.PeerConnection, error) {
	conn, err := webrtc.NewPeerConnection(defaultWebRTCConfig)
	if err != nil {
		return conn, err
	}

	conn.OnICEGatheringStateChange(func(state webrtc.ICEGatheringState) {

		log.Printf("[INFO] client %d: ice gathering state for %d changed to: %s", clientID, peerID, state.String())
	})

	conn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("[INFO] client %d: connection state for %d changed to: %s", clientID, peerID, state.String())
	})

	conn.OnICECandidate(func(iceCandidate *webrtc.ICECandidate) {
		if iceCandidate == nil {
			return
		}

		log.Printf("[INFO] client %d: ice candidate for %d acquired", clientID, peerID)

		iceSignalingCh <- iceSignalingRequest{
			to:           peerID,
			iceCandidate: iceCandidate,
		}
	})

	return conn, nil
}

func prepareDataChCallbacks(clientID uint64, peerID uint64, dataCh *webrtc.DataChannel, dataChOpened chan<- uint64, dataChOut chan<- WebRTCMsg) {
	dataCh.OnOpen(func() {
		log.Printf("[INFO] client %d: data channel for %d opened", clientID, peerID)
		dataChOpened <- peerID
	})

	dataCh.OnClose(func() {
		log.Printf("[INFO] client %d: data channel from %d was closed", clientID, peerID)
	})

	dataCh.OnError(func(err error) {
		log.Printf("[ERROR] client %d: failed to read data from %d", clientID, peerID)

	})

	dataCh.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("[INFO] client %d: received data from %d: %s", clientID, peerID, string(msg.Data))
		dataChOut <- WebRTCMsg{
			from: peerID,
			msg:  string(msg.Data),
		}
	})
}
