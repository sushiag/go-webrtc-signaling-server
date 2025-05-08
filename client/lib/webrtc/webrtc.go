package webrtc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
)

type Peer struct {
	ID                    uint64
	Connection            *webrtc.PeerConnection
	DataChannel           *webrtc.DataChannel
	bufferedICECandidates []webrtc.ICECandidateInit
	remoteDescriptionSet  bool
	mutex                 sync.Mutex

	ctx         context.Context
	cancel      context.CancelFunc
	sendChan    chan string
	isConnected bool
}

type PeerManager struct {
	UserID           uint64
	Peers            map[uint64]*Peer
	Config           webrtc.Configuration
	Mutex            sync.Mutex
	SignalingMessage SignalingMessage
	onPeerCreated    func(*Peer, SignalingMessage)
}

type SignalingMessage struct {
	Type      string   // type of message
	Content   string   // content
	RoomID    uint64   // room id
	Sender    uint64   // sender user id
	Target    uint64   // target user id
	Candidate string   // ice-candidate string
	SDP       string   // session description
	Users     []uint64 // list of user ids
	Text      string   // for send messages
	Payload   Payload  // "file", "text", "image"
}
type Payload struct {
	DataType string `json:"data_type"`
	Data     []byte `json:"data"`
}

func NewPeerManager(userID uint64) *PeerManager {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
	return &PeerManager{
		UserID: userID,
		Peers:  make(map[uint64]*Peer),
		Config: config,
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
			switch msg.Payload.DataType {
			case "audio":
				log.Printf("Received audio data, size: %d bytes", len(msg.Payload.Data))
			case "video":
				log.Printf("Received video data, size: %d bytes", len(msg.Payload.Data))
			case "file":
				log.Printf("Received file data, size: %d bytes", len(msg.Payload.Data))
			default:
				log.Printf("Received arbitrary data of type: %s, size: %d bytes", msg.Payload.DataType, len(msg.Payload.Data))
			}
		}
		data := []byte(msg.Text)
		if err := pm.SendDataToPeer(msg.Target, data); err != nil {
			log.Printf("Failed to send message to %d: %v", msg.Target, err)
		}
	case "start-session":
		log.Printf("[WEBRTC SIGNALING] Received start command. Initiating peer offers.")
		for peerID := range pm.Peers {
			if peerID == pm.UserID {
				continue
			}
			if err := pm.CreateAndSendOffer(peerID, sendFunc); err != nil {
				log.Printf("Offer to peer %d failed: %v", peerID, err)
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

	pm.Mutex.Lock()
	pm.Peers[peerID] = peer
	pm.Mutex.Unlock()

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		peer.mutex.Lock()
		defer peer.mutex.Unlock()

		init := c.ToJSON()
		if err := sendFunc(SignalingMessage{
			Type:      "ice-candidate",
			Sender:    pm.UserID,
			Target:    peerID,
			Candidate: init.Candidate,
		}); err != nil {
			log.Printf("[SIGNALING] Failed to send ICE candidate to %d: %v", peerID, err)
		}
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("Peer connection with %d changed to %s", peerID, state)
		switch state {
		case webrtc.PeerConnectionStateConnected:
			peer.mutex.Lock()
			peer.isConnected = true
			peer.mutex.Unlock()
			log.Printf("[STATE] Peer %d connected successfully.", peerID)
		case webrtc.PeerConnectionStateDisconnected:
			log.Printf("[STATE] Peer %d disconnected.", peerID)
		case webrtc.PeerConnectionStateFailed:
			log.Printf("[STATE] Peer %d connection failed.", peerID)
		case webrtc.PeerConnectionStateClosed:
			log.Printf("[STATE] Peer %d connection closed.", peerID)
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
		peer.mutex.Lock()
		defer peer.mutex.Unlock()

		init := c.ToJSON()
		if peer.remoteDescriptionSet {
			_ = sendFunc(SignalingMessage{
				Type:      "ice-candidate",
				Sender:    pm.UserID,
				Target:    msg.Sender,
				Candidate: init.Candidate,
			})
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
			pm.RemovePeer(msg.Sender)
			go pm.CheckAllConnectedAndDisconnect(sendFunc)
		}
	})

	pm.Mutex.Lock()
	pm.Peers[msg.Sender] = peer
	pm.Mutex.Unlock()

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
	pm.Mutex.Lock()
	peer, ok := pm.Peers[msg.Sender]
	pm.Mutex.Unlock()

	if !ok {
		return fmt.Errorf("peer %d not found", msg.Sender)
	}
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
	pm.Mutex.Lock()
	peer, ok := pm.Peers[msg.Sender]
	pm.Mutex.Unlock()

	if !ok {
		err := fmt.Errorf("peer %d not found", msg.Sender)
		log.Println("[ICE] Error:", err)
		return err
	}

	log.Printf("[ICE] Handling ICE candidate from peer %d", msg.Sender)

	err := peer.Connection.AddICECandidate(webrtc.ICECandidateInit{Candidate: msg.Candidate})
	if err != nil {
		log.Printf("[ICE] Failed to add ICE candidate from peer %d: %v", msg.Sender, err)
	}
	return err
}

func (pm *PeerManager) GracefulShutdown() {
	fmt.Println("Gracefully shutting down signaling, but keeping P2P alive.")
}
func (peer *Peer) OnRemoteDescriptionSet(senderID uint64, sendFunc func(SignalingMessage) error) {
	peer.mutex.Lock()
	defer peer.mutex.Unlock()

	peer.remoteDescriptionSet = true
	for _, c := range peer.bufferedICECandidates {
		_ = sendFunc(SignalingMessage{
			Type:      "ice-candidate",
			Sender:    senderID,
			Target:    peer.ID,
			Candidate: c.Candidate,
		})
	}
	peer.bufferedICECandidates = nil
}

func (peer *Peer) handleSendLoop() {
	for {
		select {
		case <-peer.ctx.Done():
			return
		case msg := <-peer.sendChan:
			if peer.Connection.ConnectionState() == webrtc.PeerConnectionStateConnected {
				_ = peer.DataChannel.SendText(msg)
			}
		}
	}
}

func (pm *PeerManager) SendDataToPeer(peerID uint64, data []byte) error {
	pm.Mutex.Lock()
	peer, ok := pm.Peers[peerID]
	pm.Mutex.Unlock()

	if !ok {
		log.Printf("[SEND ERROR] Peer %d not found", peerID)
		return fmt.Errorf("peer %d not found", peerID)
	}
	if peer.DataChannel == nil {
		log.Printf("[SEND ERROR] Peer %d has no DataChannel", peerID)
		return fmt.Errorf("peer %d has no DataChannel", peerID)
	}

	return peer.DataChannel.Send(data)
}

func (pm *PeerManager) SendPayloadToPeer(peerID uint64, payload Payload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return pm.SendDataToPeer(peerID, data)
}

func (pm *PeerManager) OnPeerCreated(f func(*Peer, SignalingMessage)) {
	pm.onPeerCreated = f
}

func (pm *PeerManager) RemovePeer(peerID uint64) {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	peer, ok := pm.Peers[peerID]
	if !ok {
		log.Printf("[REMOVE ERROR] Peer %d not found for removal", peerID)
		return
	}

	// Perform any additional cleanup here (e.g., data channel closure)
	log.Printf("[REMOVE] Initiating graceful shutdown for peer %d", peerID)
	peer.cancel()
	peer.Connection.Close()

	delete(pm.Peers, peerID)
	log.Printf("[PEER] Peer %d removed successfully", peerID)
}

func (pm *PeerManager) CloseAll() {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	for id, peer := range pm.Peers {
		if peer.Connection != nil {
			peer.Connection.Close()
		}
		peer.cancel()
		delete(pm.Peers, id)
		log.Printf("[PEER] Closed connection to peer %d", id)
	}
}

func (pm *PeerManager) GetPeerIDs() []uint64 {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	ids := make([]uint64, 0, len(pm.Peers))
	for id := range pm.Peers {
		ids = append(ids, id)
	}
	return ids
}

func (pm *PeerManager) CheckAllConnectedAndDisconnect(sendFunc func(SignalingMessage) error) {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	for _, p := range pm.Peers {
		p.mutex.Lock()
		connected := p.isConnected
		p.mutex.Unlock()

		if !connected {
			return // Not all peers are connected yet
		}
	}

	log.Println("[SIGNALING] All peers connected. Closing signaling client for full P2P.")
	pm.Close()
}

func (pm *PeerManager) Close() {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	// Close all peer connections
	for _, peer := range pm.Peers {
		if peer.Connection != nil {
			peer.Connection.Close()
		}
	}

	// Any other cleanup you need
	log.Println("[SIGNALING] PeerManager closed.")
}

func (pm *PeerManager) WaitForDataChannel(peerID uint64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		pm.Mutex.Lock()
		peer, ok := pm.Peers[peerID]
		pm.Mutex.Unlock()

		if ok && peer.DataChannel != nil {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for DataChannel for peer %d", peerID)
		}

		time.Sleep(50 * time.Millisecond)
	}
}
