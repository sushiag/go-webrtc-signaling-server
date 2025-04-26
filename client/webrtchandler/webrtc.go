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
	ID          uint64
	Connection  *webrtc.PeerConnection
	DataChannel *webrtc.DataChannel
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
		if err := pm.HandleAnswer(msg); err != nil {
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
				continue // skip self
			}
			err := pm.CreateAndSendOffer(peerID, client)
			if err != nil {
				log.Printf("Error sending offer to peer %d: %v", peerID, err)
			}
		}

		// client.Close() // optional disconnect from signaling

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

		// Send a message every few seconds
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

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		log.Printf("[SIGNALING] Sending ICE candidate to %d", peerID)
		err := client.Send(clienthandle.Message{
			Type:      clienthandle.MessageTypeICECandidate,
			Sender:    client.UserID,
			Target:    peerID,
			Candidate: c.ToJSON().Candidate,
		})
		if err != nil {
			log.Printf("[SIGNALING] Failed to send ICE candidate to %d: %v", peerID, err)
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

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("Connection state with %d has changed: %s", peerID, state)

		if state == webrtc.PeerConnectionStateConnected {
			fmt.Println("WebRTC P2P connection established!")
		}
	})

	pm.Mutex.Lock()
	pm.Peers[peerID] = &Peer{
		ID:          peerID,
		Connection:  pc,
		DataChannel: dc,
	}
	pm.Mutex.Unlock()

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

		pm.Mutex.Lock()
		if peer, ok := pm.Peers[msg.Sender]; ok {
			peer.DataChannel = dc
		} else {
			pm.Peers[msg.Sender] = &Peer{
				ID:          msg.Sender,
				Connection:  pc,
				DataChannel: dc,
			}
		}
		pm.Mutex.Unlock()
	})

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		log.Printf("[SIGNALING] Sending ICE candidate to %d", msg.Sender)
		err := client.Send(clienthandle.Message{
			Type:      clienthandle.MessageTypeICECandidate,
			Sender:    client.UserID,
			Target:    msg.Sender,
			Candidate: c.ToJSON().Candidate,
		})
		if err != nil {
			log.Printf("[SIGNALING] Failed to send ICE candidate to %d: %v", msg.Sender, err)
		}
	})

	if err := pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  msg.SDP,
	}); err != nil {
		return err
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return err
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		return err
	}
	log.Printf("[SIGNALING] Sending answer to %d", msg.Sender)

	pm.Mutex.Lock()
	pm.Peers[msg.Sender] = &Peer{
		ID:         msg.Sender,
		Connection: pc,
	}
	pm.Mutex.Unlock()

	return client.Send(clienthandle.Message{
		Type:   clienthandle.MessageTypeAnswer,
		Target: msg.Sender,
		SDP:    answer.SDP,
		Sender: client.UserID,
	})
}

func (pm *PeerManager) HandleAnswer(msg clienthandle.Message) error {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	peer, exists := pm.Peers[msg.Sender]
	if !exists {
		return fmt.Errorf("peer %d not found", msg.Sender)
	}

	log.Printf("[SIGNALING] Setting remote description for answer from %d", msg.Sender)
	return peer.Connection.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  msg.SDP,
	})
}
func (pm *PeerManager) HandleICECandidate(msg clienthandle.Message) error {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	peer, exists := pm.Peers[msg.Sender]
	if !exists {
		return fmt.Errorf("peer %d not found", msg.Sender)
	}

	log.Printf("[ICE] Handling ICE candidate from peer %d", msg.Sender)
	err := peer.Connection.AddICECandidate(webrtc.ICECandidateInit{
		Candidate: msg.Candidate,
	})
	if err != nil {

		log.Printf("[ICE] Failed to add ICE candidate from peer %d: %v", msg.Sender, err)
		return err
	}

	log.Printf("[ICE] Successfully added ICE candidate from peer %d", msg.Sender)
	return nil
}

func (pm *PeerManager) GracefulShutdown() {
	fmt.Println("Gracefully shutting down signaling, but keeping P2P alive.")
}
