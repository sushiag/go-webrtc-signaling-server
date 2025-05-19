package websocket

import (
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn              *websocket.Conn
	ServerURL         string
	ApiKey            string
	SessionKey        string
	UserID            uint64
	RoomID            uint64
	onMessage         func(Message)
	doneCh            chan struct{}
	sendQueue         chan Message
	isClosed          atomic.Bool
	isSendLoopStarted atomic.Bool
	isListenStarted   atomic.Bool
}

func NewClient(serverURL string) *Client {
	return &Client{
		ServerURL: serverURL,
		ApiKey:    os.Getenv("API_KEY"),
		doneCh:    make(chan struct{}),
		sendQueue: make(chan Message, 32),
	}
}

func (c *Client) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.ServerURL, nil)
	if err != nil {
		return err
	}
	c.Conn = conn
	log.Println("[CLIENT SIGNALING] Connected to server:", c.ServerURL)
	c.maybeStartListen()
	return nil
}

func (c *Client) ConnectWithRetry(maxRetries int) error {
	backoff := time.Second
	for i := 0; i < maxRetries; i++ {
		err := c.Connect()
		if err == nil {
			return nil
		}
		log.Printf("[CLIENT SIGNALING] Connect attempt %d failed: %v. Retrying in %s...", i+1, err, backoff)
		time.Sleep(backoff)
		backoff *= 2
	}
	return fmt.Errorf("failed to connect after %d attempts", maxRetries)
}

func (c *Client) SetServerURL(url string)           { c.ServerURL = url }
func (c *Client) SetApiKey(key string)              { c.ApiKey = key }
func (c *Client) SetMessageHandler(h func(Message)) { c.onMessage = h }
func (c *Client) IsWebSocketClosed() bool {
	return c.Conn == nil || c.Conn.CloseHandler() != nil
}
func (c *Client) StartSession() error {
	msg := Message{
		Type: "start-session",
	}
	return c.Send(msg)
}
