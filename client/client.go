package client

import (
	"fmt"
	"os"

	pm "github.com/sushiag/go-webrtc-signaling-server/client/peer_manager"
	signaling "github.com/sushiag/go-webrtc-signaling-server/client/signaling_client"
)

type Client struct {
	sClient *signaling.SignalingClient
	pm      *pm.PeerManager
}

const apiKeyEnvName = "API_KEY"

func NewClient(wsEndpoint string) (*Client, error) {
	apiKey, keyExists := os.LookupEnv(apiKeyEnvName)
	if !keyExists {
		return nil, fmt.Errorf("the API_KEY")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("the API_KEY cannot be an empty string")
	}

	return NewClientWithKey(wsEndpoint, apiKey)
}

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
