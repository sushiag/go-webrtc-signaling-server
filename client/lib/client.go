package client

import (
	"fmt"
	"log"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
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
		for {
			msg := <-client.wsResponseCh
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

	c.Websocket.SetOnMessage(func(msg websocket.Message) {
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
		// c.handleSignalingMessage(webrtc.SignalingMessage{
		// 	Type:      msg.Type,
		// 	Sender:    msg.Sender,
		// 	Target:    msg.Target,
		// 	SDP:       msg.SDP,
		// 	Candidate: msg.Candidate,
		// 	Text:      msg.Text,
		// 	Users:     msg.Users,
		// })
	})

	// go c.dispatchPeerCommands()

	return nil
}

// Handles incoming WS Messages
func (c *Client) handleSignalingMessage(msg webrtc.SignalingMessage) {
	log.Printf("[Client] Handling signaling message: %s from %d", msg.Type, msg.Sender)

	switch msg.Type {
	case common.MessageTypePeerJoined, common.MessageTypeRoomCreated:
		{
			log.Printf("[Client] Peer %d joined or created room", msg.Sender)
			c.cmdChan <- peerCommand{cmd: "add", peerID: msg.Sender}
		}

	case common.MessageTypeDisconnect:
		{
			log.Printf("[Client] Peer %d disconnected", msg.Sender)
			c.cmdChan <- peerCommand{cmd: "remove", peerID: msg.Sender}
		}

	case common.MessageTypePeerList,
		common.MessageTypeHostChanged,
		common.MessageTypeStartSession:
		{
			c.PeerManager.HandleIncomingMessage(msg, c.wsResponseCh)
		}

	case common.MessageTypeOffer,
		common.MessageTypeAnswer,
		common.MessageTypeICECandidate,
		common.MessageTypeSendMessage:
		{
			log.Printf("[Client] Routing signaling to peer %d: %s", msg.Sender, msg.Type)
			c.cmdChan <- peerCommand{cmd: "send", peerID: msg.Sender, msg: msg}
		}

	default:
		{
			log.Printf("[Client] Unknown message type: %s", msg.Type)
		}
	}
}

// Forwards WS Messages to the Peer Manager
func (c *Client) dispatchPeerCommands() {
	peers := make(map[uint64]chan webrtc.SignalingMessage)

	for cmd := range c.cmdChan {
		switch cmd.cmd {
		case "add":
			{
				if _, exists := peers[cmd.peerID]; exists {
					continue
				}
				log.Printf("[Client] Adding peer %d", cmd.peerID)
				msgCh := make(chan webrtc.SignalingMessage, 16)
				peers[cmd.peerID] = msgCh

				for msg := range msgCh {
					c.PeerManager.HandleIncomingMessage(msg, c.wsResponseCh)
				}
			}

		case "send":
			{
				ch, ok := peers[cmd.peerID]
				if !ok {
					log.Printf("[Client] Auto-adding unknown peer %d before sending", cmd.peerID)
					msgCh := make(chan webrtc.SignalingMessage, 16)
					peers[cmd.peerID] = msgCh

					for msg := range ch {
						c.PeerManager.HandleIncomingMessage(msg, c.wsResponseCh)
					}

					ch = msgCh
				}
				ch <- cmd.msg
			}

		case "remove":
			{
				if ch, ok := peers[cmd.peerID]; ok {
					log.Printf("[Client] Removing peer %d", cmd.peerID)
					close(ch)
					delete(peers, cmd.peerID)
				}
			}
		}
	}
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
