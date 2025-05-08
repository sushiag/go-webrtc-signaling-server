package client

import (
	"log"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/webrtc"
	"github.com/sushiag/go-webrtc-signaling-server/client/lib/websocket"
)

type Client struct {
	Websocket   *websocket.Client
	PeerManager *webrtc.PeerManager
	IsHost      bool
}

// NewClient() creates a wrapper with WebSocket signaling (PeerManager initialized later).
func NewClient(wsEndpoint string) *Client {
	clientwebsocket := websocket.NewClient(wsEndpoint)
	return &Client{
		Websocket: clientwebsocket,
	}
}

// Connect() handles authentication, then sets up PeerManager and signaling message handler.
func (w *Client) Connect() error {
	if err := w.Websocket.PreAuthenticate(); err != nil {
		log.Fatal("Failed to authenticate:", err)
	}

	w.PeerManager = webrtc.NewPeerManager(w.Websocket.UserID)

	// message forwarding from webrtc to websocket
	w.Websocket.SetMessageHandler(func(msg websocket.Message) {
		signalingMsg := webrtc.SignalingMessage{
			Type:      msg.Type,
			Sender:    msg.Sender,
			Target:    msg.Target,
			SDP:       msg.SDP,
			Candidate: msg.Candidate,
			Text:      msg.Text,
			Users:     msg.Users,
			Payload:   webrtc.Payload{},
		}
		log.Printf("[Client] Incoming signaling message: %+v", signalingMsg)

		// message forwarding from websocket to webrtc
		w.PeerManager.HandleSignalingMessage(signalingMsg, func(m webrtc.SignalingMessage) error {
			err := w.Websocket.Send(websocket.Message{
				Type:      m.Type,
				Sender:    m.Sender,
				Target:    m.Target,
				SDP:       m.SDP,
				Candidate: m.Candidate,
				Text:      m.Text,
				Users:     m.Users,
				Payload:   websocket.Payload{},
			})
			if err != nil {
				log.Printf("Failed to send signaling message: %v", err)
			}
			return err
		})
	})

	return w.Websocket.Init()
}

func (w *Client) CreateRoom() error {
	w.IsHost = true
	log.Println("[CLIENT] Set as host after creating room.")
	return w.Websocket.Create()
}

func (w *Client) JoinRoom(roomID string) error {
	w.IsHost = false
	return w.Websocket.JoinRoom(roomID)
}

func (w *Client) StartSession() error {
	return w.Websocket.StartSession()
}

func (w *Client) SendMessageToPeer(peerID uint64, data string) error {
	return w.PeerManager.SendDataToPeer(peerID, []byte(data))
}

func (w *Client) LeaveRoom(peerID uint64) {
	w.PeerManager.RemovePeer(peerID)
}

func (w *Client) CloseServer() {
	if w.IsHost {
		if w.PeerManager != nil && w.Websocket != nil {
			w.PeerManager.CheckAllConnectedAndDisconnect(func(m webrtc.SignalingMessage) error {
				return w.Websocket.Send(websocket.Message{
					Type:      m.Type,
					Sender:    m.Sender,
					Target:    m.Target,
					SDP:       m.SDP,
					Candidate: m.Candidate,
					Text:      m.Text,
					Users:     m.Users,
				})
			})
		}
	} else {
		log.Println("Error: Non-host client cannot close the signaling server.")
	}
}

func (w *Client) Close() {
	w.Websocket.Close()
	w.PeerManager.CloseAll()
}

func (w *Client) SetServerURL(url string) {
	w.Websocket.SetServerURL(url)
}

func (w *Client) SetApiKey(key string) {
	w.Websocket.SetApiKey(key)
}
