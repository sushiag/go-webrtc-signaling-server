package client

import (
	"fmt"
	"log"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/webrtc"
	"github.com/sushiag/go-webrtc-signaling-server/client/lib/websocket"
)

type Client struct {
	Websocket    *websocket.Client
	PeerManager  *webrtc.PeerManager
	cmdChan      chan peerCommand
	wsResponseCh chan webrtc.SignalingMessage
}

type peerCommand struct {
	cmd    string
	peerID uint64
	msg    webrtc.SignalingMessage
}

func NewClient(wsEndpoint string) *Client {
	client := &Client{
		Websocket:    websocket.NewClient(wsEndpoint),
		cmdChan:      make(chan peerCommand),
		wsResponseCh: make(chan webrtc.SignalingMessage),
	}

	// NOTE: this goroutine will be responsible for sending websocket messages.
	//
	// * Send websocket messages by sending the message through the channel
	// * Do NOT send websocket messages anywhere outside this loop
	go func() {
		for msg := range client.wsResponseCh {
			err := client.Websocket.Send(websocket.Message{
				Type:      msg.Type,
				Sender:    msg.Sender,
				Target:    msg.Target,
				SDP:       msg.SDP,
				Candidate: msg.Candidate,
				Text:      msg.Text,
				Users:     msg.Users,
			})
			if err != nil {
				log.Printf("[ERROR] failed to send websocket message: %v", err)
			}
		}

		log.Println("[INFO] closing WS send loop")
	}()

	return client
}

func (c *Client) Connect() error {
	if err := c.Websocket.PreAuthenticate(); err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}

	c.PeerManager = webrtc.NewPeerManager(c.Websocket.UserID)
	if c.PeerManager == nil {
		return fmt.Errorf("failed to initialize PeerManager")
	}

	if err := c.Websocket.Init(); err != nil {
		return err
	}
	c.Websocket.Start()

	c.Websocket.OnMessage = func(msg websocket.Message) {
		signalingMsg := webrtc.SignalingMessage{
			Type:      msg.Type,
			Sender:    msg.Sender,
			Target:    msg.Target,
			SDP:       msg.SDP,
			Candidate: msg.Candidate,
			Text:      msg.Text,
			Users:     msg.Users,
		}
		c.PeerManager.HandleIncomingMessage(signalingMsg, c.wsResponseCh)
	}

	return nil
}

func (c *Client) CreateRoom() error {
	log.Println("[Client] Creating room (set as host)")
	return c.Websocket.Create()
}

func (c *Client) JoinRoom(roomID string) error {
	return c.Websocket.JoinRoom(roomID)
}

func (c *Client) StartSession() error {
	return c.Websocket.StartSession()
}

func (c *Client) SendMessageToPeer(peerID uint64, data string) error {
	return c.PeerManager.SendBytesToPeer(peerID, []byte(data))
}

func (c *Client) PopMessage() ([]byte, bool) {
	// NOTE:
	// to implement this, we must
	// 1. initialize a [][]byte at startup; this will be the message storage
	// 2. push the messages onto a [][]byte every time the client receives a new WebRTC message
	// 3a. when the function is called, return the first one and true
	// 3b. when the function is called and the array is empty, return empty and false
	panic("TODO: implement this function")
}

func (c *Client) LeaveRoom(peerID uint64) {
	c.PeerManager.RemovePeer(peerID, c.wsResponseCh)
}

func (c *Client) Close() {
	if c.Websocket != nil {
		c.Websocket.Close()
	}
	if c.cmdChan != nil {
		close(c.cmdChan)
	}
}

func (c *Client) SetServerURL(url string) {
	c.Websocket.SetServerURL(url)
}

func (c *Client) SetApiKey(key string) {
	c.Websocket.SetApiKey(key)
}

func (c *Client) RetrySignaling(maxRetries int) {
	c.Websocket.ConnectWithRetry(maxRetries)
}
