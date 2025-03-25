package webrtc

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"os"

	"github.com/joho/godotenv"
	"github.com/pion/webrtc/v4"
	"github.com/sushiag/go-webrtc-signaling-server/internal/websocket"
)

type WebRTCClient struct {
	APIKey       string
	SignalingURL string
	RoomID       string
	ClientID     string
	WM           *websocket.WebSocketManager
	PC           *webrtc.PeerConnection
	Handler      func(sourceID string, message []byte)
}

func NewWebRTCClient(apiKey, signalingURL string, wm *websocket.WebSocketManager, roomID, clientID string) *WebRTCClient {
	return &WebRTCClient{
		APIKey:       apiKey,
		SignalingURL: signalingURL,
		WM:           wm,
		RoomID:       roomID,
		ClientID:     clientID,
	}
}

func (c *WebRTCClient) Connect(isOfferer bool) error {
	pc, err := InitializePeerConnection(c.WM, c.RoomID, c.ClientID, isOfferer)
	if err != nil {
		return err
	}
	c.PC = pc
	return nil
}

func (c *WebRTCClient) StartSession(sessionID string) error {
	if c.PC == nil {
		return errors.New("peer connection not initialized")
	}
	offer, err := c.PC.CreateOffer(nil)
	if err != nil {
		return err
	}
	err = c.PC.SetLocalDescription(offer)
	if err != nil {
		return err
	}

	msg := websocket.Message{
		Type:    "offer",
		RoomID:  c.RoomID,
		Sender:  c.ClientID,
		Content: offer.SDP,
	}
	c.WM.SendToRoom(c.RoomID, c.ClientID, msg)
	return nil
}

func (c *WebRTCClient) JoinSession(sessionID string) error {
	log.Println("[JoinSession] Placeholder - handle offer/answer from signaling server")
	return nil
}

func (c *WebRTCClient) SendMessage(targetID string, message []byte) error {
	msg := websocket.Message{
		Type:    "signal",
		RoomID:  c.RoomID,
		Sender:  c.ClientID,
		Content: string(message),
	}
	c.WM.SendToRoom(c.RoomID, targetID, msg)
	return nil
}

func (c *WebRTCClient) SetMessageHandler(handler func(sourceID string, message []byte)) {
	c.Handler = handler
}

func (c *WebRTCClient) Close() error {
	if c.PC != nil {
		return c.PC.Close()
	}
	return nil
}

func (c *WebRTCClient) HandleOffer(sdp string) error {
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	}
	if err := c.PC.SetRemoteDescription(offer); err != nil {
		return fmt.Errorf("failed to set remote offer: %w", err)
	}
	answer, err := c.PC.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("failed to create answer: %w", err)
	}
	if err := c.PC.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("failed to set local answer: %w", err)
	}
	msg := websocket.Message{
		Type:    "answer",
		RoomID:  c.RoomID,
		Sender:  c.ClientID,
		Content: answer.SDP,
	}
	c.WM.SendToRoom(c.RoomID, c.ClientID, msg)
	return nil
}

func (c *WebRTCClient) HandleAnswer(sdp string) error {
	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	}
	return c.PC.SetRemoteDescription(answer)
}

func (c *WebRTCClient) HandleICECandidate(candidateJSON string) error {
	var candidate webrtc.ICECandidateInit
	if err := json.Unmarshal([]byte(candidateJSON), &candidate); err != nil {
		return fmt.Errorf("invalid ICE candidate JSON: %w", err)
	}
	return c.PC.AddICECandidate(candidate)
}

func (c *WebRTCClient) HandleSignalingMessage(msg websocket.Message) error {
	switch msg.Type {
	case "offer":
		return c.HandleOffer(msg.Content)
	case "answer":
		return c.HandleAnswer(msg.Content)
	case "ice-candidate":
		return c.HandleICECandidate(msg.Content)
	default:
		if c.Handler != nil {
			c.Handler(msg.Sender, []byte(msg.Content))
		}
	}
	return nil
}

func LoadSTUNServer() string {
	_ = godotenv.Load() // loads the env variables
	stunServer := os.Getenv("STUN_SERVER")
	if stunServer == "" {
		stunServer = "stun:stun.1.google.com:19302" // default google stun server
	}
	return stunServer
}

func InitializePeerConnection(wm *websocket.WebSocketManager, roomID, clientID string, createDataChannel bool) (*webrtc.PeerConnection, error) {
	stunServer := LoadSTUNServer()
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{stunServer}},
		},
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %v", err)
	}
	// create datachannel for offerer
	if createDataChannel {
		wm.SetupDataChannel(peerConnection, clientID)
	}
	// listen for the data channel
	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		log.Printf("[WS] DataChannel received for client %s\n", clientID)

		dc.OnOpen(func() {
			log.Printf("[WS] DataChannel opened (received) for client %s\n", clientID)
		})

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("[WS] DataChannel received message from %s: %s\n", clientID, string(msg.Data))
		})

		wm.DataChannelMtx.Lock()
		wm.DataChannels[clientID] = dc
		wm.DataChannelMtx.Unlock()
	})

	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			candidate := c.ToJSON()
			candidateJSON, _ := json.Marshal(candidate)

			message := websocket.Message{
				Type:    "ice-candidate",
				RoomID:  roomID,
				Sender:  clientID,
				Content: string(candidateJSON),
			}
			wm.SendToRoom(roomID, clientID, message)
		}
	})

	peerConnection.OnNegotiationNeeded(func() {
		log.Println("[SDP] Negotiation needed")
	})

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Println("[WebRTC] New track received:", track.Kind())
	})

	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[ICE] Connection state changed to: %s", state.String())

		switch state {
		case webrtc.ICEConnectionStateFailed:
			log.Println("[ICE] connection failed. restart attemping:")
			offer, err := peerConnection.CreateOffer(&webrtc.OfferOptions{ICERestart: true})
			if err != nil {
				log.Println("[ICE] failed to create/restart offer:", err)
				return
			}
			peerConnection.SetLocalDescription(offer)
			wm.SendToRoom(roomID, clientID, websocket.Message{
				Type:    "offer",
				RoomID:  roomID,
				Sender:  clientID,
				Content: string(offer.SDP),
			})
		case webrtc.ICEConnectionStateDisconnected:
			log.Println("[ICE] Disconnected, checking if reconnection is possible.. attempting to reconnect..")

		}

	})

	return peerConnection, nil
}
