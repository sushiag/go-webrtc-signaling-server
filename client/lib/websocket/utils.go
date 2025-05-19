package websocket

import "log"

func (c *Client) LeaveServer() {
	log.Println("[CLIENT SIGNALING] Leaving signaling server and switching to P2P")
	c.Close()
}

func (c *Client) Close() {
	if c.isClosed.CompareAndSwap(false, true) {
		close(c.doneCh)
		if c.Conn != nil {
			_ = c.Conn.Close()
		}
		log.Println("[CLIENT SIGNALING] Client closed.")
	}
}
