package client

import (
	"fmt"
	"log"
	"strconv"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
	"github.com/sushiag/go-webrtc-signaling-server/client/lib/webrtc"
	ws "github.com/sushiag/go-webrtc-signaling-server/client/lib/websocket"
)

type Client struct {
	Websocket    *ws.Client
	MsgOutCh     chan common.WebRTCMessage
	PeerManager  *webrtc.PeerManager
	cmdChan      chan peerCommand
	wsResponseCh chan ws.Message

	// crazy stuff
	createRoomRespCh   chan error
	joinRoomRespCh     chan error
	startSessionRespCh chan error
}

type peerCommand struct {
	cmd    string
	peerID uint64
	msg    webrtc.SignalingMessage
}

func NewClient(wsEndpoint string) *Client {
	client := &Client{
		Websocket:    ws.NewClient(wsEndpoint),
		cmdChan:      make(chan peerCommand),
		wsResponseCh: make(chan ws.Message),
		MsgOutCh:     make(chan common.WebRTCMessage),
	}

	// NOTE: this goroutine will be responsible for sending websocket messages.
	//
	// * Send websocket messages by sending the message through the channel
	// * Do NOT send websocket messages anywhere outside this loop
	//
	// TODO: maybe we can also just send to the SendWSMsgCh directly
	go func() {
		for msg := range client.wsResponseCh {
			convertedMsg := ws.Message{
				Type:      msg.Type,
				Sender:    msg.Sender,
				Target:    msg.Target,
				SDP:       msg.SDP,
				Candidate: msg.Candidate,
				Text:      msg.Text,
				Users:     msg.Users,
			}
			client.Websocket.SendWSMsgCh <- convertedMsg
		}

		log.Println("[INFO] closing WS send loop")
	}()

	return client
}

func (c *Client) Connect() error {
	if err := c.Websocket.PreAuthenticate(); err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}

	c.PeerManager = webrtc.NewPeerManager(c.Websocket.UserID, c.MsgOutCh)
	if c.PeerManager == nil {
		return fmt.Errorf("failed to initialize PeerManager")
	}

	if err := c.Websocket.Init(); err != nil {
		return err
	}
	c.startWSLoops()

	return nil
}

func (c *Client) CreateRoom() error {
	log.Println("[Client] Creating room (set as host)")

	c.createRoomRespCh = make(chan error, 1)
	defer func() {
		c.createRoomRespCh = nil
	}()

	c.Websocket.SendWSMsgCh <- ws.Message{Type: common.MessageTypeCreateRoom}

	err := <-c.createRoomRespCh

	return err
}

func (c *Client) JoinRoom(roomID string) error {
	c.joinRoomRespCh = make(chan error, 1)
	defer func() {
		c.joinRoomRespCh = nil
	}()

	roomIDUint64, parseErr := strconv.ParseUint(roomID, 10, 64)
	if parseErr != nil {
		return fmt.Errorf("invalid room ID: %v", parseErr)
	}
	c.Websocket.RoomID = roomIDUint64

	c.Websocket.SendWSMsgCh <- ws.Message{Type: common.MessageTypeJoinRoom, RoomID: c.Websocket.RoomID}

	joinRoomErr := <-c.joinRoomRespCh

	return joinRoomErr
}

func (c *Client) StartSession() error {
	c.startSessionRespCh = make(chan error, 1)
	defer func() {
		c.startSessionRespCh = nil
	}()

	msg := ws.Message{
		Type: common.MessageTypeStartSession,
	}
	c.Websocket.SendWSMsgCh <- msg
	err := <-c.startSessionRespCh

	return err
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
