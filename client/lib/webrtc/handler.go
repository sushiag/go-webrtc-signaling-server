package webrtc

import (
	"context"
	"fmt"
	"log"

	"github.com/pion/webrtc/v4"
)

func (pm *PeerManager) managerWorker() {
	for fn := range pm.managerQueue {
		fn()
	}
}

func NewPeerManager(userID uint64) *PeerManager {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	pm := &PeerManager{
		UserID:             userID,
		Peers:              make(map[uint64]*Peer),
		Config:             config,
		managerQueue:       make(chan func(), 100),
		iceCandidateBuffer: make(map[uint64][]webrtc.ICECandidateInit),
	}

	go pm.managerWorker()

	return pm
}

func (pm *PeerManager) HandleSignalingMessage(msg SignalingMessage, sendFunc func(SignalingMessage) error) {
	pm.managerQueue <- func() {
		if pm.sendSignalFunc == nil {
			pm.sendSignalFunc = sendFunc
		}
		switch msg.Type {
		case "peer-list":
			for _, peerID := range msg.Users {
				if peerID == pm.UserID {
					continue
				}
				if _, exists := pm.Peers[peerID]; exists {
					log.Printf("[WEBRTC SIGNALING] Already connected to %d, skipping", peerID)
					continue
				}
				log.Printf("[WEBRTC SIGNALING] Connecting to peer: %d", peerID)
				if err := pm.CreateAndSendOffer(peerID, sendFunc); err != nil {
					log.Printf("[WEBRTC SIGNALING] Offer to %d failed: %v", peerID, err)
				}
			}

		case "offer":
			log.Printf("[WEBRTC SIGNALING] Received offer from %d", msg.Sender)
			if err := pm.HandleOffer(msg, sendFunc); err != nil {
				log.Printf("[WEBRTC SIGNALING] Handle offer error: %v", err)
			}

		case "answer":
			log.Printf("[WEBRTC SIGNALING] Received answer from %d", msg.Sender)
			if err := pm.HandleAnswer(msg, sendFunc); err != nil {
				log.Printf("[WEBRTC SIGNALING] Handle answer error: %v", err)
			}
		case "host-changed":
			log.Printf("[WEBRTC SIGNALING] Received host change from %d", msg.Sender)
			if newHostID := pm.findNextHost(); newHostID != 0 {
				pm.HostID = newHostID
				log.Printf("[HOST] Host reassigned to %d", pm.HostID)

				if pm.sendSignalFunc != nil {
					hostChangeMsg := SignalingMessage{
						Type:   "host-changed",
						Sender: pm.UserID,
						Target: 0,
						Users:  pm.GetPeerIDs(),
					}
					if err := pm.sendSignalFunc(hostChangeMsg); err != nil {
						log.Printf("[SIGNALING] Failed to send host-changed message: %v", err)
					}
				}
			} else {
				log.Printf("[WEBRTC SIGNALING] No new host found.")
			}

		case "ice-candidate":
			log.Printf("[WEBRTC SIGNALING] Received ICE candidate from %d", msg.Sender)
			if err := pm.HandleICECandidate(msg, sendFunc); err != nil {
				log.Printf("[WEBRTC SIGNALING] Failed to handle candidate from %d: %v", msg.Sender, err)
			}

		case "send-message":
			if msg.Text == "" && msg.Payload.Data == nil {
				log.Printf("Empty message received from %d, ignoring", msg.Sender)
				return
			}

			if msg.Text != "" {
				log.Printf("[WEBRTC SIGNALING] Received text message from %d to %d: %s", msg.Sender, msg.Target, msg.Text)
			}

			if msg.Payload.Data != nil {
				log.Printf("[WEBRTC SIGNALING] Received %s data from %d", msg.Payload.DataType, msg.Sender)
			}

			data := []byte(msg.Text)
			if err := pm.SendDataToPeer(msg.Target, data); err != nil {
				log.Printf("Failed to send message to %d: %v", msg.Target, err)
			}
		case "start-session":
			log.Printf("[WEBRTC SIGNALING] Received start command. Going Full peer to peer.")
			for peerID := range pm.Peers {
				if peerID == pm.UserID {
					continue
				}
				if err := pm.CheckAllConnectedAndDisconnect(); err != nil {
					log.Printf("Offer to peer %d failed: %v", peerID, err)
				}
			}

		}
	}
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

		// Flush buffered ICE candidates
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
			Type:      "ice-candidate",
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
			Type:   "offer",
			Sender: pm.UserID,
			Target: peerID,
			SDP:    offer.SDP,
		}); err != nil {
			log.Printf("Peer %d failed to offer :%v", peerID, err)
		}
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
				Type:      "ice-candidate",
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

	// Enqueue peer insertion to the manager queue to be safe.
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

	// Set remote description before marking remoteDescriptionSet and sending candidates.
	if err := pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  msg.SDP,
	}); err != nil {
		return err
	}

	// Now that remote description is set, notify peer and flush buffered ICE candidates.
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
			Type:   "answer",
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

func (pm *PeerManager) HandleICECandidate(msg SignalingMessage, sendFunc func(SignalingMessage) error) error {
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
			Type:      "ice-candidate",
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
