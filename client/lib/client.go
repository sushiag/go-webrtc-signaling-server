package client

import (
	"fmt"
	"log"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/webrtc"
	"github.com/sushiag/go-webrtc-signaling-server/client/lib/websocket"
)

type Client struct {
	Websocket   *websocket.Client
	PeerManager *webrtc.PeerManager
}

// NewClient() creates a wrapper with WebSocket signaling
func NewClient(wsEndpoint string) *Client {
	clientwebsocket := websocket.NewClient(wsEndpoint)
	return &Client{
		Websocket: clientwebsocket,
	}
}

// Connect() handles authentication, then sets up PeerManager and signaling message handler
func (w *Client) Connect() error {
	if err := w.Websocket.PreAuthenticate(); err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}

	w.PeerManager = webrtc.NewPeerManager(w.Websocket.UserID)
	if w.PeerManager == nil {
		return fmt.Errorf("PeerManager is not initialized")
	}

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
			// TODO: are we not sending our API key anymore and the server just trusts it?
			// ... or maybe it's fine because the connection is already established? just doublecheck
			if err := w.Websocket.Send(websocket.Message{
				Type:      m.Type,
				Sender:    m.Sender,
				Target:    m.Target,
				SDP:       m.SDP,
				Candidate: m.Candidate,
				Text:      m.Text,
				Users:     m.Users,
				Payload:   websocket.Payload{},
			}); err != nil {
				log.Printf("Error sending signaling message: %v", err)
			}
			return nil
		})
	})

	return w.Websocket.Init()
}

func (w *Client) CreateRoom() error {
	log.Println("[CLIENT] Set as host after creating room.")
	return w.Websocket.Create()
}

func (w *Client) JoinRoom(roomID string) error {
	return w.Websocket.JoinRoom(roomID)
}

func (w *Client) StartSession() error {
	return w.Websocket.StartSession()

}

func (w *Client) SendMessageToPeer(peerID uint64, data string) error {
	return w.PeerManager.SendDataToPeer(peerID, []byte(data))
}

func (c *Client) LeaveRoom(peerID uint64) {
	c.PeerManager.RemovePeer(peerID, func(msg webrtc.SignalingMessage) error {
		log.Printf("[CLIENT] Removed peer %d, signaling message: %+v", peerID, msg)
		return nil
	})
}

func (w *Client) Close() {
	if w.Websocket != nil {
		w.Websocket.Close()
	}
}

func (w *Client) SetServerURL(url string) {
	w.Websocket.SetServerURL(url)
}

func (w *Client) SetApiKey(key string) {
	w.Websocket.SetApiKey(key)
}

func (w *Client) RetrySignaling(maxRetries int) {
	w.Websocket.ConnectWithRetry(2)
}
