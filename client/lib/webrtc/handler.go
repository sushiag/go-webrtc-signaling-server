package webrtc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/pion/webrtc/v4"
	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
	ws "github.com/sushiag/go-webrtc-signaling-server/client/lib/websocket"
)

func NewPeerManager(userID uint64, msgOutCh chan common.WebRTCMessage) *PeerManager {
	fmt.Println("[DEBUG] NewPeerManager called for user:", userID)
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	pm := &PeerManager{
		userID:                userID,
		peers:                 make(map[uint64]*Peer),
		config:                config,
		managerQueue:          make(chan func(), 100),
		iceCandidateBuffer:    make(map[uint64][]webrtc.ICECandidateInit),
		pmEventCh:             make(chan pmEvent),
		processingLoopStarted: false,
		msgOutCh:              msgOutCh,
		PeerEventsCh:          make(chan common.PeerEvent),
	}

	pm.startProcessingEvents()

	return pm
}

func (sm *SignalingMessage) Decode(r io.Reader) error {
	return json.NewDecoder(r).Decode(sm)
}

// Process all incoming commands
func (pm *PeerManager) startProcessingEvents() {
	// guard so it's impossible to call this multiple times
	if pm.processingLoopStarted {
		return
	}

	// NOTE: this is how we serialize access to the PeerManager struct
	//
	// The struct SHOULD only be mutated through this goroutine loop so
	// there wouldn't be any concurrent read/writes.
	//
	// If a function in this loop were to spawn a goroutine,
	// it shouldn't be mutating the PeerManger.
	//
	// To send events to the peer manager, use the `pmEventCh`.
	go func() {
		for event := range pm.pmEventCh {
			switch event := event.(type) {
			case pmGetPeerIDs:
				{
					peerIDs := pm.getPeerIDsSwitch()
					event.resultCh <- peerIDs
				}
			case pmSendDataToPeer:
				{
					err := pm.sendBytesToPeerSwitch(event.peerID, event.data)
					event.resultCh <- err
				}
			case pmSendJSONToPeer:
				{
					err := pm.sendJSONToPeerSwitch(event.peerID, event.payload)
					event.resultCh <- err
				}
			case pmHandleIncomingMsg:
				{
					pm.handleIncomingMessageSwitch(event)
				}
			case pmRemovePeer:
				{
					pm.removePeerSwitch(event.peerID, event.responseCh)
				}
			default:
				{
					log.Printf("[ERROR] received an invalid peer manager event\n")
				}
			}
		}
	}()

	pm.processingLoopStarted = true
}

func (pm *PeerManager) handleIncomingMessageSwitch(event pmHandleIncomingMsg) {
	log.Printf("[DEBUG] Dispatching signaling message: type=%s from=%d to=%d\n", event.msg.Type, event.msg.Sender, event.msg.Target)

	if event.msg.Type < 0 {
		log.Println("[ERROR] Invalid message type; ignoring.")
		return
	}

	switch event.msg.Type {
	case common.MessageTypePeerList:
		{
			if pm.userID == pm.hostID {
				log.Println("[WEBRTC SIGNALING] Host detected; skipping peer-list processing.")
				return
			}
			for _, peerID := range event.msg.Users {
				if peerID == pm.userID || pm.peers[peerID] != nil {
					continue
				}
				log.Printf("[WEBRTC SIGNALING: %d] Initiating connection to peer %d\n", pm.userID, peerID)

				pm.createAndSendOfferSwitch(peerID, event.responseCh)
			}
		}

	case common.MessageTypeOffer:
		{
			if pm.userID == pm.hostID {
				log.Println("[WEBRTC SIGNALING] Host should not respond to offers. Skipping.")
				return
			}
			pm.handleOfferSwitch(event.msg, event.responseCh)
		}

	case common.MessageTypeAnswer:
		{
			peer, exists := pm.peers[event.msg.Sender]
			if !exists {
				log.Printf("[WEBRTC SIGNALING] Answer from unknown peer %d; ignoring.\n", event.msg.Sender)
				return
			}
			if peer.Connection.RemoteDescription() != nil {
				log.Printf("[WEBRTC SIGNALING] Remote description already set for peer %d; skipping answer.\n", event.msg.Sender)
				return
			}
			pm.handleAnswer(event.msg, event.responseCh)
		}

	case common.MessageTypeICECandidate:
		{
			if err := pm.handleICECandidateSwitch(event.msg); err != nil {
				log.Printf("[WEBRTC SIGNALING] Error handling ICE candidate from %d: %v\n", event.msg.Sender, err)
			}

		}

	case common.MessageTypeHostChanged:
		{
			log.Printf("[WEBRTC SIGNALING] Host changed to: %d\n", event.msg.Sender)
			pm.hostID = event.msg.Sender
		}

	case common.MessageTypeStartSession:
		{
			log.Printf("[WEBRTC SIGNALING] Start session triggered.")
			if err := pm.checkAllConnectedAndDisconnectSwitch(); err != nil {
				log.Printf("[WEBRTC SIGNALING] Error in full P2P session start: %v\n", err)
			}
		}

	case common.MessageTypeSendMessage:
		{
			if event.msg.Text == "" && event.msg.Payload.Data == nil {
				log.Printf("[WEBRTC SIGNALING] Empty message received from %d; ignoring.\n", event.msg.Sender)
				return
			}
			pm.sendBytesToPeerSwitch(event.msg.Target, []byte(event.msg.Text))
		}

	default:
		{
			log.Printf("[WEBRTC SIGNALING] Unknown message type: %s\n", event.msg.Type)
		}
	}
}

func (pm *PeerManager) createAndSendOfferSwitch(peerID uint64, responseCh chan ws.Message) {
	log.Printf("[DEBUG] creating new peer connection for %d\n", peerID)
	pc, err := webrtc.NewPeerConnection(pm.config)
	if err != nil {
		log.Printf("[ERROR] failed to create new peer connection for %d: %v\n", peerID, err)
		return
	}

	log.Printf("[DEBUG] creating new data channel for %d\n", peerID)
	dc, err := pc.CreateDataChannel("data", nil)
	if err != nil {
		log.Printf("[ERROR] failed to create new peer data channel for %d: %v\n", peerID, err)
		return
	}

	dc.OnOpen(func() {
		pm.PeerEventsCh <- common.PeerDataChOpened{PeerID: peerID}
		log.Printf("[INFO] data channel opened for %d\n", peerID)
	})

	ctx, cancel := context.WithCancel(context.Background())
	sendChan := make(chan string, 10)

	peer := &Peer{
		ID:          peerID,
		Connection:  pc,
		DataChannel: dc,
		ctx:         ctx,
		cancel:      cancel,
		sendChan:    sendChan,
	}

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("Received raw data from %d: %v", peerID, msg.Data)
		if msg.IsString {
			log.Printf("As string: %s", string(msg.Data))
		}
		pm.msgOutCh <- common.WebRTCMessage{
			From: peer.ID,
			Data: msg.Data,
		}
	})

	pm.peers[peerID] = peer

	if buffered, ok := pm.iceCandidateBuffer[peerID]; ok {
		log.Printf("[ICE] Flushing %d buffered candidates for peer %d", len(buffered), peerID)
		for _, c := range buffered {
			if err := peer.Connection.AddICECandidate(c); err != nil {
				log.Printf("[ICE] Failed to add buffered candidate to peer %d: %v", peerID, err)
			}
		}
		delete(pm.iceCandidateBuffer, peerID)
	}

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		init := c.ToJSON()
		responseCh <- ws.Message{
			Type:      common.MessageTypeICECandidate,
			Sender:    pm.userID,
			Target:    peerID,
			Candidate: init.Candidate,
		}
		if err != nil {
			log.Printf("[SIGNALING] Failed to send ICE candidate to %d: %v", peerID, err)
		}
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("Peer connection with %d changed to %s", peerID, state)

		select {
		case peer.sendChan <- state.String():
		default:
			log.Printf("[WARN] Channel is full for peer %d, state: %s", peerID, state)
		}
	})

	log.Printf("[DEBUG] creating offer for %d", peerID)
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		log.Printf("[ERROR] failed to create new peer offer for %d: %v\n", peerID, err)
		return
	}

	log.Printf("[DEBUG] setting local description for %d", peerID)
	if err := pc.SetLocalDescription(offer); err != nil {
		log.Printf("[ERROR] failed to set local description for %d: %v\n", peerID, err)
		return
	}
	log.Printf("[SIGNALING: %d] Sending offer to %d", pm.userID, peerID)

	responseCh <- ws.Message{
		Type:   common.MessageTypeOffer,
		Sender: pm.userID,
		Target: peerID,
		SDP:    offer.SDP,
	}
}

func (pm *PeerManager) handleOfferSwitch(msg SignalingMessage, responseCh chan ws.Message) {
	pc, err := webrtc.NewPeerConnection(pm.config)
	if err != nil {
		log.Printf("[ERROR] failed to create new peer connection while handling offer: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	sendChan := make(chan string, 10)

	peer := &Peer{
		ID:          msg.Sender,
		Connection:  pc,
		DataChannel: nil,
		ctx:         ctx,
		cancel:      cancel,
		sendChan:    sendChan,
	}

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		peer.DataChannel = dc

		dc.OnOpen(func() {
			pm.PeerEventsCh <- common.PeerDataChOpened{PeerID: peer.ID}
			log.Printf("[DATA: %d] Channel opened with %d", pm.userID, msg.Sender)
		})

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("Received raw data: %v", msg.Data)
			if msg.IsString {
				log.Printf("As string: %s", string(msg.Data))
			}
			pm.msgOutCh <- common.WebRTCMessage{From: peer.ID, Data: msg.Data}
		})
	})

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		init := c.ToJSON()
		if peer.remoteDescriptionSet {
			log.Printf("[DEBUG] remote description set for %d, sending ICE candidates", peer.ID)

			iceMsg := ws.Message{
				Type:      common.MessageTypeICECandidate,
				Sender:    pm.userID,
				Target:    msg.Sender,
				Candidate: init.Candidate,
			}
			responseCh <- iceMsg
		} else {
			peer.bufferedICECandidates = append(peer.bufferedICECandidates, init)
		}
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("Connection with %d state changed to %s", msg.Sender, state)
		switch state {
		case webrtc.PeerConnectionStateDisconnected,
			webrtc.PeerConnectionStateFailed,
			webrtc.PeerConnectionStateClosed:

			log.Printf("[PEER] Connection to %d is %s. Cleaning up.", msg.Sender, state)
			peer.cancel()
			pm.removePeerSwitch(msg.Sender, responseCh)
			pm.checkAllConnectedAndDisconnectSwitch()
		}
	})

	pm.peers[msg.Sender] = peer

	if buffered, ok := pm.iceCandidateBuffer[msg.Sender]; ok {
		log.Printf("[HANDLE OFFER] Flushing %d buffered candidates for peer %d", len(buffered), msg.Sender)
		for _, c := range buffered {
			if err := peer.Connection.AddICECandidate(c); err != nil {
				log.Printf("[HANDLE OFFER] Failed to add buffered candidate to peer %d: %v", msg.Sender, err)
			}
		}
		delete(pm.iceCandidateBuffer, msg.Sender)
	}
	// end

	if err := pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  msg.SDP,
	}); err != nil {
		log.Printf("[ERROR] failed to set remote description for %d: %v\n", msg.Sender, err)
	}
	peer.onRemoteDescriptionSet(pm.userID, responseCh)

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		log.Printf("[ERROR] failed to create answer for %d: %v\n", msg.Sender, err)
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		log.Printf("[ERROR] failed to set local description for %d: %v\n", msg.Sender, err)
	}

	responseCh <- ws.Message{
		Type:   common.MessageTypeAnswer,
		Sender: pm.userID,
		Target: msg.Sender,
		SDP:    answer.SDP,
	}
}

func (pm *PeerManager) handleAnswer(msg SignalingMessage, responseCh chan ws.Message) {
	peer, ok := pm.peers[msg.Sender]
	if !ok {
		log.Printf("[ERROR] failed to handle answer: peer %d not found", msg.Sender)
	}

	log.Printf("[SIGNALING] Setting remote description for answer from %d", msg.Sender)
	err := peer.Connection.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  msg.SDP,
	})
	if err != nil {
		log.Printf("[ERROR] failed to handle answer: could not set remote description for %d", msg.Sender)
		return
	}

	peer.onRemoteDescriptionSet(pm.userID, responseCh)
}

func (pm *PeerManager) handleICECandidateSwitch(msg SignalingMessage) error {
	peer, ok := pm.peers[msg.Sender]
	candidate := webrtc.ICECandidateInit{Candidate: msg.Candidate}

	if !ok {
		log.Printf("[ICE] Peer %d not ready yet. Buffering candidate.\n", msg.Sender)
		pm.iceCandidateBuffer[msg.Sender] = append(pm.iceCandidateBuffer[msg.Sender], candidate)
		return nil
	}

	log.Printf("[ICE] Handling ICE candidate from peer %d\n", msg.Sender)
	err := peer.Connection.AddICECandidate(candidate)
	if err != nil {
		return fmt.Errorf("[ICE] Failed to add ICE candidate from peer %d: %v\n", msg.Sender, err)
	}

	return nil
}

func (peer *Peer) onRemoteDescriptionSet(senderID uint64, responseCh chan ws.Message) {
	peer.remoteDescriptionSet = true
	log.Printf("[ICE] Remote description set for peer %d. Sending %d buffered candidates.", peer.ID, len(peer.bufferedICECandidates))
	for _, c := range peer.bufferedICECandidates {
		response := ws.Message{
			Type:      common.MessageTypeICECandidate,
			Sender:    senderID,
			Target:    peer.ID,
			Candidate: c.Candidate,
		}
		log.Printf("[INFO] Sending ICE candidate to %d", peer.ID)
		responseCh <- response
	}
	peer.bufferedICECandidates = nil

	select {
	case peer.sendChan <- "check_connected":
	default:
		log.Printf("[WARN] sendChan full for peer %d", peer.ID)
	}
}
