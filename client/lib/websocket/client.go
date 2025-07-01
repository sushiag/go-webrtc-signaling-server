package websocket

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn        *websocket.Conn
	ServerURL   string
	ApiKey      string
	SessionKey  string
	UserID      uint64
	RoomID      uint64
	DoneCh      chan struct{}
	SendWSMsgCh chan Message // handles outgoing messages
	IsClosed    bool
}

func NewClient(serverURL string) *Client {
	return &Client{
		ServerURL:   serverURL,
		ApiKey:      os.Getenv("API_KEY"),
		DoneCh:      make(chan struct{}),
		SendWSMsgCh: make(chan Message, 32),
	}
}

func (c *Client) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.ServerURL, nil)
	if err != nil {
		return err
	}
	c.Conn = conn
	log.Println("[CLIENT SIGNALING] Connected to server:", c.ServerURL)
	return nil
}

func (c *Client) ConnectWithRetry(maxRetries int) error {
	backoff := time.Second
	for i := range maxRetries {
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

func (c *Client) SetServerURL(url string) { c.ServerURL = url }
func (c *Client) SetApiKey(key string)    { c.ApiKey = key }
func (c *Client) IsWebSocketClosed() bool {
	return c.Conn == nil || c.Conn.CloseHandler() != nil
}
