package webrtchandler

import (
	"fmt"
	"log"
	"sync"
	"time"

	clienthandle "github.com/sushiag/go-webrtc-signaling-server/client/clienthandler"

	"github.com/pion/webrtc/v4"
)

type Peer struct {
	ID                    uint64
	Connection            *webrtc.PeerConnection
	DataChannel           *webrtc.DataChannel
	bufferedICECandidates []webrtc.ICECandidateInit
	remoteDescriptionSet  bool
	mutex                 sync.Mutex
}

type PeerManager struct {
	Peers  map[uint64]*Peer
	Config webrtc.Configuration
	Mutex  sync.Mutex
}

func NewPeerManager() *PeerManager {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	return &PeerManager{
		Peers:  make(map[uint64]*Peer),
		Config: config,
	}
}

func (pm *PeerManager) HandleSignalingMessage(msg clienthandle.Message, client *clienthandle.Client) {
	switch msg.Type {
	case clienthandle.MessageTypePeerList:
		for _, peerID := range msg.Users {
			if peerID == client.UserID {
				continue
			}
			log.Printf("[WEBRTC SIGNALING] Connecting to peer: %d", peerID)
			if err := pm.CreateAndSendOffer(peerID, client); err != nil {
				log.Printf("[WEBRTC SIGNALING] Offer to %d failed: %v", peerID, err)
			}
		}

	case clienthandle.MessageTypeOffer:
		log.Printf("[WEBRTC SIGNALING] Received offer from %d", msg.Sender)
		if err := pm.HandleOffer(msg, client); err != nil {
			log.Printf("[WEBRTC SIGNALING] Handle offer error: %v", err)
		}

	case clienthandle.MessageTypeAnswer:
		log.Printf("[WEBRTC SIGNALING] Received answer from %d", msg.Sender)
		if err := pm.HandleAnswer(msg, client); err != nil {
			log.Printf("[WEBRTC SIGNALING] Handle answer error: %v", err)
		}

	case clienthandle.MessageTypeICECandidate:
		log.Printf("[WEBRTC SIGNALING] Received ICE candidate from %d", msg.Sender)
		if err := pm.HandleICECandidate(msg); err != nil {
			log.Printf("[WEBRTC SIGNALING] Failed to handle candidate from %d: %v", msg.Sender, err)
		}

	case clienthandle.MessageTypeStart:
		log.Printf("[CLIENT SIGNALING] Received start from host %d. Initiating peer connections...", msg.Sender)

		for peerID := range pm.Peers {
			if peerID == client.UserID {
				continue
			}
			err := pm.CreateAndSendOffer(peerID, client)
			if err != nil {
				log.Printf("Error sending offer to peer %d: %v", peerID, err)
			}
		}
	}
}

func (pm *PeerManager) CreateAndSendOffer(peerID uint64, client *clienthandle.Client) error {
	pc, err := webrtc.NewPeerConnection(pm.Config)
	if err != nil {
		return fmt.Errorf("create peer connection: %w", err)
	}

	dc, err := pc.CreateDataChannel("data", nil)
	if err != nil {
		return fmt.Errorf("create data channel: %w", err)
	}

	dc.OnOpen(func() {
		fmt.Println("DataChannel opened")
		go func() {
			for {
				time.Sleep(2 * time.Second)
				err := dc.SendText("Hello from peer!")
				if err != nil {
					fmt.Println("Send error:", err)
				}
			}
		}()
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("[DATA] From %d: %s", peerID, string(msg.Data))
	})

	peer := &Peer{
		ID:          peerID,
		Connection:  pc,
		DataChannel: dc,
	}
	pm.Mutex.Lock()
	pm.Peers[peerID] = peer
	pm.Mutex.Unlock()

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		peer.mutex.Lock()
		defer peer.mutex.Unlock()

		candidateInit := c.ToJSON()
		if peer.remoteDescriptionSet {
			log.Printf("[SIGNALING] Sending ICE candidate immediately to %d", peerID)
			err := client.Send(clienthandle.Message{
				Type:      clienthandle.MessageTypeICECandidate,
				Sender:    client.UserID,
				Target:    peerID,
				Candidate: candidateInit.Candidate,
			})
			if err != nil {
				log.Printf("[SIGNALING] Failed to send ICE candidate to %d: %v", peerID, err)
			}
		} else {
			log.Printf("[SIGNALING] Buffering ICE candidate for %d", peerID)
			peer.bufferedICECandidates = append(peer.bufferedICECandidates, candidateInit)
		}
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("Connection state with %d has changed: %s", peerID, state)

		if state == webrtc.PeerConnectionStateConnected {
			fmt.Println("WebRTC P2P connection established!")
		}

		if state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateClosed ||
			state == webrtc.PeerConnectionStateDisconnected {
			log.Printf("[PEER] Connection to %d is %s. Cleaning up.", peerID, state)
			pm.RemovePeer(peerID)
		}
	})

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("create offer: %w", err)
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("set local description: %w", err)
	}
	log.Printf("[SIGNALING] Sending offer to %d", peerID)

	return client.Send(clienthandle.Message{
		Type:   clienthandle.MessageTypeOffer,
		Target: peerID,
		SDP:    offer.SDP,
		Sender: client.UserID,
	})
}

func (pm *PeerManager) HandleOffer(msg clienthandle.Message, client *clienthandle.Client) error {
	pc, err := webrtc.NewPeerConnection(pm.Config)

	if err != nil {
		return err
	}

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnOpen(func() {
			log.Printf("[DATA] Channel open with %d", msg.Sender)
		})
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("[DATA] Received message: %s", string(msg.Data))
		})
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("Connection state with %d has changed: %s", msg.Sender, state)

		if state == webrtc.PeerConnectionStateConnected {
			fmt.Println("WebRTC P2P connection established (incoming offer)!")
		}

		if state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateClosed ||
			state == webrtc.PeerConnectionStateDisconnected {
			log.Printf("[PEER] Connection to %d is %s. Cleaning up.", msg.Sender, state)
			pm.RemovePeer(msg.Sender)
		}
	})

	peer := &Peer{
		ID:          msg.Sender,
		Connection:  pc,
		DataChannel: nil,
	}
	pm.Mutex.Lock()
	pm.Peers[msg.Sender] = peer
	pm.Mutex.Unlock()

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		peer.mutex.Lock()
		defer peer.mutex.Unlock()

		candidateInit := c.ToJSON()
		if peer.remoteDescriptionSet {
			log.Printf("[SIGNALING] Sending ICE candidate immediately to %d", msg.Sender)
			err := client.Send(clienthandle.Message{
				Type:      clienthandle.MessageTypeICECandidate,
				Sender:    client.UserID,
				Target:    msg.Sender,
				Candidate: candidateInit.Candidate,
			})
			if err != nil {
				log.Printf("[SIGNALING] Failed to send ICE candidate to %d: %v", msg.Sender, err)
			}
		} else {
			log.Printf("[SIGNALING] Buffering ICE candidate for %d", msg.Sender)
			peer.bufferedICECandidates = append(peer.bufferedICECandidates, candidateInit)
		}
	})

	err = pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  msg.SDP,
	})
	if err != nil {
		return err
	}

	peer.OnRemoteDescriptionSet(client)

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return err
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		return err
	}

	return client.Send(clienthandle.Message{
		Type:   clienthandle.MessageTypeAnswer,
		Target: msg.Sender,
		SDP:    answer.SDP,
		Sender: client.UserID,
	})
}

func (pm *PeerManager) HandleAnswer(msg clienthandle.Message, client *clienthandle.Client) error {
	pm.Mutex.Lock()
	peer, exists := pm.Peers[msg.Sender]
	pm.Mutex.Unlock()

	if !exists {
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

	peer.OnRemoteDescriptionSet(client)

	return nil
}

func (pm *PeerManager) HandleICECandidate(msg clienthandle.Message) error {
	pm.Mutex.Lock()
	peer, exists := pm.Peers[msg.Sender]
	pm.Mutex.Unlock()

	if !exists {
		return fmt.Errorf("peer %d not found", msg.Sender)
	}

	log.Printf("[ICE] Handling ICE candidate from peer %d", msg.Sender)
	return peer.Connection.AddICECandidate(webrtc.ICECandidateInit{
		Candidate: msg.Candidate,
	})
}

func (pm *PeerManager) GracefulShutdown() {
	fmt.Println("Gracefully shutting down signaling, but keeping P2P alive.")
}

// NEW: Add this method to Peer
func (p *Peer) OnRemoteDescriptionSet(client *clienthandle.Client) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.remoteDescriptionSet = true

	for _, candidate := range p.bufferedICECandidates {
		err := client.Send(clienthandle.Message{
			Type:      clienthandle.MessageTypeICECandidate,
			Sender:    client.UserID,
			Target:    p.ID,
			Candidate: candidate.Candidate,
		})
		if err != nil {
			log.Printf("[SIGNALING] Failed to flush buffered candidate to %d: %v", p.ID, err)
		}
	}
	p.bufferedICECandidates = nil
}

func (pm *PeerManager) RemovePeer(peerID uint64) {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	if peer, ok := pm.Peers[peerID]; ok {
		log.Printf("[PEER] Removing peer %d", peerID)
		if peer.Connection != nil {
			peer.Connection.Close() // make sure connection is closed
		}
		delete(pm.Peers, peerID)
	} else {
		log.Printf("[PEER] Tried to remove non-existent peer %d", peerID)
	}
}
