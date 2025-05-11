package client

import (
	"log"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/webrtc"
	"github.com/sushiag/go-webrtc-signaling-server/client/lib/websocket"
)

type Client struct {
	Websocket   *websocket.Client
	PeerManager *webrtc.PeerManager
}

func NewClient(wsEndpoint string) *Client {
	wsClient := websocket.NewClient(wsEndpoint)
	return &Client{
		Websocket: wsClient,
	}
}

// and sets up message forwarding between WebSocket and WebRTC layers.
func (c *Client) Connect() error {
	if err := c.Websocket.PreAuthenticate(); err != nil {
		log.Fatal("[CLIENT] Failed to authenticate:", err)
	}

	c.PeerManager = webrtc.NewPeerManager(c.Websocket.UserID)

	c.Websocket.SetMessageHandler(func(msg websocket.Message) {
		signalingMsg := webrtc.SignalingMessage{
			Type:      msg.Type,
			Sender:    msg.Sender,
			Target:    msg.Target,
			SDP:       msg.SDP,
			Candidate: msg.Candidate,
			Text:      msg.Text,
			Users:     msg.Users,
			Payload:   webrtc.Payload{}, // extend this if used later
		}
		log.Printf("[CLIENT] Incoming signaling message: %+v", signalingMsg)

		c.PeerManager.HandleSignalingMessage(signalingMsg, func(m webrtc.SignalingMessage) error {
			return c.Websocket.Send(websocket.Message{
				Type:      m.Type,
				Sender:    m.Sender,
				Target:    m.Target,
				SDP:       m.SDP,
				Candidate: m.Candidate,
				Text:      m.Text,
				Users:     m.Users,
				Payload:   websocket.Payload{}, // extend this if used later
			})
		})
	})

	return c.Websocket.Init()
}

func (c *Client) CreateRoom() error {
	log.Println("[CLIENT] Creating room and assuming host role.")
	return c.Websocket.Create()
}

func (c *Client) JoinRoom(roomID string) error {
	log.Println("[CLIENT] Joining room:", roomID)
	return c.Websocket.JoinRoom(roomID)
}

func (c *Client) StartSession() error {
	return c.Websocket.StartSession()
}

func (c *Client) SendMessageToPeer(peerID uint64, data string) error {
	return c.PeerManager.SendDataToPeer(peerID, []byte(data))
}

func (c *Client) LeaveRoom(peerID uint64) {
	c.PeerManager.RemovePeer(peerID, func(msg webrtc.SignalingMessage) error {
		log.Printf("[CLIENT] Removed peer %d, signaling message: %+v", peerID, msg)
		return nil
	})
}

func (c *Client) Close() {
	c.Websocket.Close()
	c.PeerManager.CloseAll()
}

func (c *Client) SetServerURL(url string) {
	c.Websocket.SetServerURL(url)
}

func (c *Client) SetApiKey(key string) {
	c.Websocket.SetApiKey(key)
}
