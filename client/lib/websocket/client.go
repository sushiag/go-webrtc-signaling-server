package websocket

import (
	"os"

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

func (c *Client) SetServerURL(url string) { c.ServerURL = url }
func (c *Client) SetApiKey(key string)    { c.ApiKey = key }
