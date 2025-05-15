package webrtc

import (
	"context"
	"fmt"
	"log"

	"github.com/pion/webrtc/v4"
)

func NewPeerManager(userID uint64) *PeerManager {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
	return &PeerManager{
		UserID: userID,
		Config: config,
	}
}
func (pm *PeerManager) SetInitialHost(peerIDs []uint64) {
	var minID uint64
	first := true

	pm.Peers.Range(func(key, _ interface{}) bool {
		id := key.(uint64)
		if first || id < minID {
			minID = id
			first = false
		}
		return true
	})
	if !first {
		pm.HostID = minID
		log.Printf("[HOST] Initial host set to: %d", pm.HostID)
	}
}

func (pm *PeerManager) HandleSignalingMessage(msg SignalingMessage, sendFunc func(SignalingMessage) error) {
	switch msg.Type {
	case "peer-list":
		for _, peerID := range msg.Users {
			if peerID == pm.UserID {
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
		log.Printf("[WEBRTC SIGNALING] Received answer from %d", msg.Sender)
		if newHostID := pm.findNextHost(); newHostID != 0 {
			pm.HostID = newHostID
			log.Printf("[HOST] Host reassigned to %d", pm.HostID)

			hostChangeMsg := SignalingMessage{
				Type:   "host-changed",
				Sender: pm.UserID,
				Target: 0,
				Users:  pm.GetPeerIDs(),
			}
			if err := sendFunc(hostChangeMsg); err != nil {
				log.Printf("[SIGNALING] Failed to send host-changed message: %v", err)
			}
		} else {
			log.Printf("[WEBRTC SIGNALING] No new host found.")
		}

	case "ice-candidate":
		log.Printf("[WEBRTC SIGNALING] Received ICE candidate from %d", msg.Sender)
		if err := pm.HandleICECandidate(msg); err != nil {
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

		pm.Peers.Range(func(key, value interface{}) bool {
			peerID := key.(uint64)
			if peerID == pm.UserID {
				return true
			}
			if err := pm.CheckAllConnectedAndDisconnect(); err != nil {
				log.Printf("Offer to peer %d failed: %v", peerID, err)
			}
			return true
		})
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
		cmdChan:     make(chan PeerCommand, 10), // <-- REQUIRED
	}
	peer.StartCommandLoop()

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("Received raw data: %v", msg.Data)
		if msg.IsString {
			log.Printf("As string: %s", string(msg.Data))
		}
	})
	pm.Peers.Store(peerID, peer)

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		init := c.ToJSON()

		peer.cmdChan <- PeerCommand{
			Action: func(p *Peer) {
				if err := sendFunc(SignalingMessage{
					Type:      "ice-candidate",
					Sender:    pm.UserID,
					Target:    peerID,
					Candidate: init.Candidate,
				}); err != nil {
					log.Printf("[SIGNALING] Failed to send ICE candidate to %d: %v", peerID, err)
				}
			},
		}
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("Peer connection with %d changed to %s", peerID, state)

		if state == webrtc.PeerConnectionStateConnected {
			peer.cmdChan <- PeerCommand{
				Action: func(p *Peer) {
					p.isConnected = true
					log.Printf("[STATE] Peer %d connected successfully.", peerID)
				},
			}
		} else {
			log.Printf("[STATE] Peer %d connection state: %s", peerID, state)
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

	return sendFunc(SignalingMessage{
		Type:   "offer",
		Sender: pm.UserID,
		Target: peerID,
		SDP:    offer.SDP,
	})
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
		cmdChan:     make(chan PeerCommand, 10), // <-- REQUIRED
	}
	peer.StartCommandLoop()

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

		peer.cmdChan <- PeerCommand{
			Action: func(p *Peer) {
				if p.remoteDescriptionSet {
					err := sendFunc(SignalingMessage{
						Type:      "ice-candidate",
						Sender:    pm.UserID,
						Target:    msg.Sender,
						Candidate: init.Candidate,
					})
					if err != nil {
						log.Printf("[SIGNALING] Failed to send ICE candidate to %d: %v", msg.Sender, err)
					}
				} else {
					p.bufferedICECandidates = append(p.bufferedICECandidates, init)
				}
			},
		}
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("Connection with %d state changed to %s", msg.Sender, state)
		switch state {
		case webrtc.PeerConnectionStateDisconnected,
			webrtc.PeerConnectionStateFailed,
			webrtc.PeerConnectionStateClosed:

			log.Printf("[PEER] Connection to %d is %s. Cleaning up.", msg.Sender, state)
			peer.Cancel()
			pm.RemovePeer(msg.Sender, sendFunc)
			go pm.CheckAllConnectedAndDisconnect()
		}
	})

	pm.Peers.Store(msg.Sender, peer)

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

	return sendFunc(SignalingMessage{
		Type:   "answer",
		Sender: pm.UserID,
		Target: msg.Sender,
		SDP:    answer.SDP,
	})
}
func (pm *PeerManager) HandleAnswer(msg SignalingMessage, sendFunc func(SignalingMessage) error) error {
	value, ok := pm.Peers.Load(msg.Sender)
	if !ok {
		return fmt.Errorf("peer %d not found", msg.Sender)
	}
	peer := value.(*Peer)

	log.Printf("[SIGNALING] Setting remote description for answer from %d", msg.Sender)

	if err := peer.Connection.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  msg.SDP,
	}); err != nil {
		return err
	}

	peer.OnRemoteDescriptionSet(pm.UserID, sendFunc)
	return nil
}
func (pm *PeerManager) HandleICECandidate(msg SignalingMessage) error {
	value, ok := pm.Peers.Load(msg.Sender)
	if !ok {
		err := fmt.Errorf("peer %d not found", msg.Sender)
		log.Println("[ICE] Error:", err)
		return err
	}
	peer := value.(*Peer)

	log.Printf("[ICE] Handling ICE candidate from peer %d", msg.Sender)

	err := peer.Connection.AddICECandidate(webrtc.ICECandidateInit{Candidate: msg.Candidate})
	if err != nil {
		log.Printf("[ICE] Failed to add ICE candidate from peer %d: %v", msg.Sender, err)
	}
	return err
}
