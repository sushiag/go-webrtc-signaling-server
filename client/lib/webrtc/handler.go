package webrtc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/pion/webrtc/v4"
	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
)

func NewPeerManager(userID uint64) *PeerManager {
	fmt.Println("[DEBUG] NewPeerManager called for user:", userID)
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	pm := &PeerManager{
		UserID:                userID,
		Peers:                 make(map[uint64]*Peer),
		Config:                config,
		managerQueue:          make(chan func(), 100),
		iceCandidateBuffer:    make(map[uint64][]webrtc.ICECandidateInit),
		outgoingMessages:      make(chan SignalingMessage, 16),
		pmEventCh:             make(chan pmEvent),
		processingLoopStarted: false,
	}

	go func() {
		for fn := range pm.managerQueue {
			fn()
		}
	}()

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
	go func() {
		for event := range pm.pmEventCh {
			switch e := event.(type) {
			case pmCloseAll:
				{
					pm.closeAllSwitch()
				}
			case pmGetPeerIDs:
				{
					peerIDs := pm.getPeerIDsSwitch()
					e.resultCh <- peerIDs
				}
			case pmCheckAllConnectedAndDisconnect:
				{
					err := pm.checkAllConnectedAndDisconnectSwitch()
					e.resultCh <- err
				}
			case pmWaitForDataChannel:
				{
					err := pm.waitForDataChannelSwitch(e.peerID, e.timeout)
					e.resultCh <- err
				}
			case pmSendDataToPeer:
				{
					err := pm.sendDataToPeerSwitch(e.peerID, e.data)
					e.resultCh <- err
				}
			case pmHandleIncomingMsg:
				{
					pm.handleICECandidateSwitch(e.msg, e.sendFunc)
				}
			case pmRemovePeer:
				{
					pm.removePeerSwitch(e.peerID, e.sendFunc)
				}
			case pmHandleICECandidate:
				{
					err := pm.handleICECandidateSwitch(e.msg, e.sendFunc)
					e.resultCh <- err
				}
			case pmCreateAndSendOffer:
				{
					err := pm.createAndSendOfferSwitch(e.peerID, e.sendFunc)
					e.resultCh <- err
				}
			case pmHandleOffer:
				{
					err := pm.handleOfferSwitch(e.msg, e.sendFunc)
					e.resultCh <- err
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

func (pm *PeerManager) OutgoingMessages() <-chan SignalingMessage {
	return pm.outgoingMessages
}

func (pm *PeerManager) handleIncomingMessageSwitch(msg SignalingMessage, sendFunc func(SignalingMessage) error) {
	if pm.sendSignalFunc == nil {
		pm.sendSignalFunc = sendFunc
	}

	log.Printf("[DEBUG] Dispatching signaling message: type=%s from=%d to=%d", msg.Type, msg.Sender, msg.Target)

	if msg.Type < 0 {
		log.Println("[WARN] Invalid message type; ignoring.")
		return
	}

	switch msg.Type {
	case common.MessageTypePeerList:
		if pm.UserID == pm.HostID {
			log.Println("[WEBRTC SIGNALING] Host detected; skipping peer-list processing.")
			return
		}
		for _, peerID := range msg.Users {
			if peerID == pm.UserID || pm.Peers[peerID] != nil {
				continue
			}
			log.Printf("[WEBRTC SIGNALING] Initiating connection to peer %d", peerID)
			if err := pm.CreateAndSendOffer(peerID, sendFunc); err != nil {
				log.Printf("[WEBRTC SIGNALING] Failed to offer to %d: %v", peerID, err)
			}
		}

	case common.MessageTypeOffer:
		if pm.UserID == pm.HostID {
			log.Println("[WEBRTC SIGNALING] Host should not respond to offers. Skipping.")
			return
		}
		if err := pm.HandleOffer(msg, sendFunc); err != nil {
			log.Printf("[WEBRTC SIGNALING] Error handling offer from %d: %v", msg.Sender, err)
		}

	case common.MessageTypeAnswer:
		peer, exists := pm.Peers[msg.Sender]
		if !exists {
			log.Printf("[WEBRTC SIGNALING] Answer from unknown peer %d; ignoring.", msg.Sender)
			return
		}
		if peer.Connection.RemoteDescription() != nil {
			log.Printf("[WEBRTC SIGNALING] Remote description already set for peer %d; skipping answer.", msg.Sender)
			return
		}
		if err := pm.HandleAnswer(msg, sendFunc); err != nil {
			log.Printf("[WEBRTC SIGNALING] Error handling answer from %d: %v", msg.Sender, err)
		}

	case common.MessageTypeICECandidate:
		if err := pm.HandleICECandidate(msg, sendFunc); err != nil {
			log.Printf("[WEBRTC SIGNALING] Error handling ICE candidate from %d: %v", msg.Sender, err)
		}

	case common.MessageTypeHostChanged:
		log.Printf("[WEBRTC SIGNALING] Host changed to: %d", msg.Sender)
		pm.HostID = msg.Sender

	case common.MessageTypeStartSession:
		log.Printf("[WEBRTC SIGNALING] Start session triggered.")
		if err := pm.CheckAllConnectedAndDisconnect(); err != nil {
			log.Printf("[WEBRTC SIGNALING] Error in full P2P session start: %v", err)
		}

	case common.MessageTypeSendMessage:
		if msg.Text == "" && msg.Payload.Data == nil {
			log.Printf("[WEBRTC SIGNALING] Empty message received from %d; ignoring.", msg.Sender)
			return
		}
		if err := pm.SendDataToPeer(msg.Target, []byte(msg.Text)); err != nil {
			log.Printf("[WEBRTC SIGNALING] Failed to send message to %d: %v", msg.Target, err)
		}

	default:
		log.Printf("[WEBRTC SIGNALING] Unknown message type: %s", msg.Type)
	}
}

func (pm *PeerManager) HandleIncomingMessage(msg SignalingMessage, sendFunc func(SignalingMessage) error) {
	// TODO: convert to pmEvent, struct ok
	pm.managerQueue <- func() {
		if pm.sendSignalFunc == nil {
			pm.sendSignalFunc = sendFunc
		}

		log.Printf("[DEBUG] Dispatching signaling message: type=%s from=%d to=%d", msg.Type, msg.Sender, msg.Target)

		if msg.Type < 0 {
			log.Println("[WARN] Invalid message type; ignoring.")
			return
		}

		switch msg.Type {
		case common.MessageTypePeerList:
			if pm.UserID == pm.HostID {
				log.Println("[WEBRTC SIGNALING] Host detected; skipping peer-list processing.")
				return
			}
			for _, peerID := range msg.Users {
				if peerID == pm.UserID || pm.Peers[peerID] != nil {
					continue
				}
				log.Printf("[WEBRTC SIGNALING] Initiating connection to peer %d", peerID)
				if err := pm.CreateAndSendOffer(peerID, sendFunc); err != nil {
					log.Printf("[WEBRTC SIGNALING] Failed to offer to %d: %v", peerID, err)
				}
			}

		case common.MessageTypeOffer:
			if pm.UserID == pm.HostID {
				log.Println("[WEBRTC SIGNALING] Host should not respond to offers. Skipping.")
				return
			}
			if err := pm.HandleOffer(msg, sendFunc); err != nil {
				log.Printf("[WEBRTC SIGNALING] Error handling offer from %d: %v", msg.Sender, err)
			}

		case common.MessageTypeAnswer:
			peer, exists := pm.Peers[msg.Sender]
			if !exists {
				log.Printf("[WEBRTC SIGNALING] Answer from unknown peer %d; ignoring.", msg.Sender)
				return
			}
			if peer.Connection.RemoteDescription() != nil {
				log.Printf("[WEBRTC SIGNALING] Remote description already set for peer %d; skipping answer.", msg.Sender)
				return
			}
			if err := pm.HandleAnswer(msg, sendFunc); err != nil {
				log.Printf("[WEBRTC SIGNALING] Error handling answer from %d: %v", msg.Sender, err)
			}

		case common.MessageTypeICECandidate:
			if err := pm.HandleICECandidate(msg, sendFunc); err != nil {
				log.Printf("[WEBRTC SIGNALING] Error handling ICE candidate from %d: %v", msg.Sender, err)
			}

		case common.MessageTypeHostChanged:
			log.Printf("[WEBRTC SIGNALING] Host changed to: %d", msg.Sender)
			pm.HostID = msg.Sender

		case common.MessageTypeStartSession:
			log.Printf("[WEBRTC SIGNALING] Start session triggered.")
			if err := pm.CheckAllConnectedAndDisconnect(); err != nil {
				log.Printf("[WEBRTC SIGNALING] Error in full P2P session start: %v", err)
			}

		case common.MessageTypeSendMessage:
			if msg.Text == "" && msg.Payload.Data == nil {
				log.Printf("[WEBRTC SIGNALING] Empty message received from %d; ignoring.", msg.Sender)
				return
			}
			if err := pm.SendDataToPeer(msg.Target, []byte(msg.Text)); err != nil {
				log.Printf("[WEBRTC SIGNALING] Failed to send message to %d: %v", msg.Target, err)
			}

		default:
			log.Printf("[WEBRTC SIGNALING] Unknown message type: %s", msg.Type)
		}
	}
}

func (pm *PeerManager) createAndSendOfferSwitch(peerID uint64, sendFunc func(SignalingMessage) error) error {
	pc, err := webrtc.NewPeerConnection(pm.Config)
	if err != nil {
		return err
	}
	dc, err := pc.CreateDataChannel("data", nil)
	if err != nil {
		return fmt.Errorf("create data channel: %w", err)
	}

	dc.OnOpen(func() {
		fmt.Println("DataChannel opened")
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

	// TODO: checkout what's happening here
	go peer.handleSendLoop()

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("Received raw data: %v", msg.Data)
		if msg.IsString {
			log.Printf("As string: %s", string(msg.Data))
		}
	})

	// TODO: convert to pmEvent, struct ok, switch ok
	// NOTE: i only removed sending this section through a channel because this is already
	// being called in the processing loop
	// start
	pm.Peers[peerID] = peer

	if buffered, ok := pm.iceCandidateBuffer[peerID]; ok {
		log.Printf("[ICE] Flushing %d buffered candidates for peer %d", len(buffered), peerID)
		for _, c := range buffered {
			if err := peer.Connection.AddICECandidate(c); err != nil {
				log.Printf("[ICE] Failed to add buffered candidate to peer %d: %v", peerID, err)
			}
		}
		delete(pm.iceCandidateBuffer, peerID)
	}
	// end

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil || pm.sendSignalFunc == nil {
			return
		}
		init := c.ToJSON()
		err := pm.sendSignalFunc(SignalingMessage{
			Type:      common.MessageTypeICECandidate,
			Sender:    pm.UserID,
			Target:    peerID,
			Candidate: init.Candidate,
		})
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

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return err
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		return err
	}
	log.Printf("[SIGNALING] Sending offer to %d", peerID)

	if pm.sendSignalFunc != nil {
		if err := pm.sendSignalFunc(SignalingMessage{
			Type:   common.MessageTypeOffer,
			Sender: pm.UserID,
			Target: peerID,
			SDP:    offer.SDP,
		}); err != nil {
			log.Printf("Peer %d failed to offer :%v", peerID, err)
		}
	}

	return nil
}

func (pm *PeerManager) CreateAndSendOffer(peerID uint64, sendFunc func(SignalingMessage) error) error {
	pc, err := webrtc.NewPeerConnection(pm.Config)
	if err != nil {
		return err
	}
	dc, err := pc.CreateDataChannel("data", nil)
	if err != nil {
		return fmt.Errorf("create data channel: %w", err)
	}

	dc.OnOpen(func() {
		fmt.Println("DataChannel opened")
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

	go peer.handleSendLoop()

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("Received raw data: %v", msg.Data)
		if msg.IsString {
			log.Printf("As string: %s", string(msg.Data))
		}
	})

	pm.managerQueue <- func() {
		pm.Peers[peerID] = peer

		if buffered, ok := pm.iceCandidateBuffer[peerID]; ok {
			log.Printf("[ICE] Flushing %d buffered candidates for peer %d", len(buffered), peerID)
			for _, c := range buffered {
				if err := peer.Connection.AddICECandidate(c); err != nil {
					log.Printf("[ICE] Failed to add buffered candidate to peer %d: %v", peerID, err)
				}
			}
			delete(pm.iceCandidateBuffer, peerID)
		}
	}

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil || pm.sendSignalFunc == nil {
			return
		}
		init := c.ToJSON()
		err := pm.sendSignalFunc(SignalingMessage{
			Type:      common.MessageTypeICECandidate,
			Sender:    pm.UserID,
			Target:    peerID,
			Candidate: init.Candidate,
		})
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

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return err
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		return err
	}
	log.Printf("[SIGNALING] Sending offer to %d", peerID)

	if pm.sendSignalFunc != nil {
		if err := pm.sendSignalFunc(SignalingMessage{
			Type:   common.MessageTypeOffer,
			Sender: pm.UserID,
			Target: peerID,
			SDP:    offer.SDP,
		}); err != nil {
			log.Printf("Peer %d failed to offer :%v", peerID, err)
		}
	}
	return nil
}

func (pm *PeerManager) handleOfferSwitch(msg SignalingMessage, sendFunc func(SignalingMessage) error) error {
	pc, err := webrtc.NewPeerConnection(pm.Config)
	if err != nil {
		return err
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
			log.Printf("[DATA] Channel open with %d", msg.Sender)
		})
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("Received raw data: %v", msg.Data)
			if msg.IsString {
				log.Printf("As string: %s", string(msg.Data))
			}
		})
	})

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		init := c.ToJSON()
		if peer.remoteDescriptionSet {
			if err := sendFunc(SignalingMessage{
				Type:      common.MessageTypeICECandidate,
				Sender:    pm.UserID,
				Target:    msg.Sender,
				Candidate: init.Candidate,
			}); err != nil {
				log.Printf("[ICE] Failed to send ICE candidate: %v", err)
			}
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
			pm.RemovePeer(msg.Sender, sendFunc)
			go pm.CheckAllConnectedAndDisconnect()
		}
	})

	// TODO: convert to pmEvent, struct ok, switch ok

	// NOTE: i only removed sending this section through a channel since this is
	// already called within the processing loop
	// start
	pm.Peers[msg.Sender] = peer

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
		return err
	}
	peer.OnRemoteDescriptionSet(pm.UserID, sendFunc)

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return err
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		return err
	}

	if sendFunc != nil {
		return sendFunc(SignalingMessage{
			Type:   common.MessageTypeAnswer,
			Sender: pm.UserID,
			Target: msg.Sender,
			SDP:    answer.SDP,
		})
	}
	return nil
}

func (pm *PeerManager) HandleOffer(msg SignalingMessage, sendFunc func(SignalingMessage) error) error {
	pc, err := webrtc.NewPeerConnection(pm.Config)
	if err != nil {
		return err
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
			log.Printf("[DATA] Channel open with %d", msg.Sender)
		})
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("Received raw data: %v", msg.Data)
			if msg.IsString {
				log.Printf("As string: %s", string(msg.Data))
			}
		})
	})

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		init := c.ToJSON()
		if peer.remoteDescriptionSet {
			if err := sendFunc(SignalingMessage{
				Type:      common.MessageTypeICECandidate,
				Sender:    pm.UserID,
				Target:    msg.Sender,
				Candidate: init.Candidate,
			}); err != nil {
				log.Printf("[ICE] Failed to send ICE candidate: %v", err)
			}
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
			pm.RemovePeer(msg.Sender, sendFunc)
			go pm.CheckAllConnectedAndDisconnect()
		}
	})

	// TODO: convert to pmEvent, struct ok, switch ok
	pm.managerQueue <- func() {
		pm.Peers[msg.Sender] = peer

		if buffered, ok := pm.iceCandidateBuffer[msg.Sender]; ok {
			log.Printf("[HANDLE OFFER] Flushing %d buffered candidates for peer %d", len(buffered), msg.Sender)
			for _, c := range buffered {
				if err := peer.Connection.AddICECandidate(c); err != nil {
					log.Printf("[HANDLE OFFER] Failed to add buffered candidate to peer %d: %v", msg.Sender, err)
				}
			}
			delete(pm.iceCandidateBuffer, msg.Sender)
		}
	}

	if err := pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  msg.SDP,
	}); err != nil {
		return err
	}
	peer.OnRemoteDescriptionSet(pm.UserID, sendFunc)

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return err
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		return err
	}

	if sendFunc != nil {
		return sendFunc(SignalingMessage{
			Type:   common.MessageTypeAnswer,
			Sender: pm.UserID,
			Target: msg.Sender,
			SDP:    answer.SDP,
		})
	}
	return nil
}

func (pm *PeerManager) HandleAnswer(msg SignalingMessage, sendFunc func(SignalingMessage) error) error {
	peer, ok := pm.Peers[msg.Sender]
	if !ok {
		return fmt.Errorf("peer %d not found", msg.Sender)
	}

	log.Printf("[SIGNALING] Setting remote description for answer from %d", msg.Sender)
	err := peer.Connection.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  msg.SDP,
	})
	if err != nil {
		return err
	}

	peer.OnRemoteDescriptionSet(pm.UserID, sendFunc)
	return nil
}

func (pm *PeerManager) handleICECandidateSwitch(msg SignalingMessage, sendFunc func(SignalingMessage) error) error {
	peer, ok := pm.Peers[msg.Sender]
	candidate := webrtc.ICECandidateInit{Candidate: msg.Candidate}

	if !ok {
		log.Printf("[ICE] Peer %d not ready yet. Buffering candidate.\n", msg.Sender)
		pm.iceCandidateBuffer[msg.Sender] = append(pm.iceCandidateBuffer[msg.Sender], candidate)
		return nil
	}

	log.Printf("[ICE] Handling ICE candidate from peer %d", msg.Sender)
	err := peer.Connection.AddICECandidate(candidate)
	if err != nil {
		return fmt.Errorf("[ICE] Failed to add ICE candidate from peer %d: %v\n", msg.Sender, err)
	}

	return nil
}

func (pm *PeerManager) HandleICECandidate(msg SignalingMessage, sendFunc func(SignalingMessage) error) error {
	// TODO: convert to pmEvent, struct ok, switch ok
	pm.managerQueue <- func() {
		peer, ok := pm.Peers[msg.Sender]
		candidate := webrtc.ICECandidateInit{Candidate: msg.Candidate}

		if !ok {
			log.Printf("[ICE] Peer %d not ready yet. Buffering candidate.", msg.Sender)
			pm.iceCandidateBuffer[msg.Sender] = append(pm.iceCandidateBuffer[msg.Sender], candidate)
			return
		}

		log.Printf("[ICE] Handling ICE candidate from peer %d", msg.Sender)
		err := peer.Connection.AddICECandidate(candidate)
		if err != nil {
			log.Printf("[ICE] Failed to add ICE candidate from peer %d: %v", msg.Sender, err)
		}
	}
	return nil
}

func (peer *Peer) OnRemoteDescriptionSet(senderID uint64, sendFunc func(SignalingMessage) error) {
	peer.remoteDescriptionSet = true
	log.Printf("[ICE] Remote description set for peer %d. Sending %d buffered candidates.", peer.ID, len(peer.bufferedICECandidates))
	for _, c := range peer.bufferedICECandidates {
		if err := sendFunc(SignalingMessage{
			Type:      common.MessageTypeICECandidate,
			Sender:    senderID,
			Target:    peer.ID,
			Candidate: c.Candidate,
		}); err != nil {
			log.Printf("[ICE] Failed to send buffered candidate to %d: %v", peer.ID, err)
		}
	}
	peer.bufferedICECandidates = nil

	select {
	case peer.sendChan <- "check_connected":
	default:
		log.Printf("[WARN] sendChan full for peer %d", peer.ID)
	}
}
