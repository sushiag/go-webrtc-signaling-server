package websocket

import (
	"log"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn       *websocket.Conn
	ServerURL  string
	ApiKey     string
	SessionKey string
	UserID     uint64
	RoomID     uint64
	onMessage  func(Message)
	sendChan   chan Message
	doneChan   chan struct{}
	isClosed   bool
	SendMutex  sync.Mutex
	closeOnce  sync.Once
}

func NewClient(serverUrl string) *Client {
	return &Client{
		ServerURL: serverUrl,
		ApiKey:    os.Getenv("API_KEY"),
		doneChan:  make(chan struct{}),
	}
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
	return c.WriteMessage(msg)
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.isClosed = true

		if c.Conn != nil {
			err := c.Conn.Close()
			if err != nil {
				log.Println("[CLIENT SIGNALING] Error closing WebSocket connection:", err)
			}
		}

		close(c.doneChan)
		log.Println("[CLIENT SIGNALING] WebSocket connection closed.")
	})
}
