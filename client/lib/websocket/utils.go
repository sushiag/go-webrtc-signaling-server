package websocket

import "log"

func (c *Client) LeaveServer() {
	log.Println("[CLIENT SIGNALING] Leaving signaling server and switching to P2P")
	c.Close()
}
