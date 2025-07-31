package client

import (
	"fmt"
	"os"

	pm "github.com/sushiag/go-webrtc-signaling-server/client/peer_manager"
	signaling "github.com/sushiag/go-webrtc-signaling-server/client/signaling_client"
)

// This represents a client connecting to the server, managing rooms, and sending/receivinng peer messages.
type Client struct {
	sClient *signaling.SignalingClient
	pm      *pm.PeerManager
}

const apiKeyEnvName = "API_KEY"

// This checks if an cient connecting to the websocket has a API-Key, thiis returns an error if the API-Key is missing.
func NewClient(wsEndpoint string) (*Client, error) {
	apiKey, keyExists := os.LookupEnv(apiKeyEnvName)
	if !keyExists || apiKey == "" {
		return nil, fmt.Errorf("API_KEY environment variable is not set or empty")
	}
	return NewClientWithKey(wsEndpoint, apiKey)
}

// This creates a nnew client using the websocket endpoint and API-Key, as well as initialize the signaling client and peer manager.
func NewClientWithKey(wsEndpoint string, apiKey string) (*Client, error) {
	sClient, err := signaling.NewSignalingClient(wsEndpoint, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialized signaling client: %v", err)
	}

	pm := pm.NewPeerManager(sClient.SignalingIn, sClient.SignalingOut)

	client := &Client{
		sClient: sClient,
		pm:      pm,
	}

	return client, nil
}

// This sends a request to the signaling server to create a new room, it returns an aissgned room Id or an error.
func (c *Client) CreateRoom() (uint64, error) {
	return c.sClient.CreateRoom()
}

// This sends a request to join an eixsting room
func (c *Client) JoinRoom(roomID uint64) ([]uint64, error) {
	return c.sClient.JoinRoom(roomID)
}

// this notify the signaling server that a client is leaving the room.
func (c *Client) LeaveRoom() {
	c.sClient.LeaveRoom()
}

// GetDataChOpened returns a read-only channel that emits the peer ID (uint64)
// whenever a new data channel is successfully established with a peer.
func (c *Client) GetDataChOpened() <-chan uint64 {
	return c.pm.GetDataChOpenedCh()
}

// GetPeerDataMsgCh returns a read-only channel that receives messages
// from connected peers. Each message is represented as a PeerDataMsg.
func (c *Client) GetPeerDataMsgCh() <-chan pm.PeerDataMsg {
	return c.pm.GetPeerDataMsgCh()
}

// SendDataToPeer sends a byte slice of data to the specified peer identified
// by peerID. Returns an error if the peer is not connected or the send fails.
// Sends data to peer
func (c *Client) SendDataToPeer(peerID uint64, data []byte) error {
	return c.pm.SendDataToPeer(peerID, data)
}

// This returns a uniqye user ID assigned ot the client by the servet
func (c *Client) GetClientID() uint64 {
	return c.sClient.ClientID
}
