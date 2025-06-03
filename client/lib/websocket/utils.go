package websocket

import "log"

func (c *Client) CloseSignaling() {
	if c.isClosed.CompareAndSwap(false, true) {
		close(c.doneCh)
		if c.Conn != nil {
			_ = c.Conn.Close()
		}
		log.Println("[CLIENT SIGNALING] Client disconnected from signaling server.")
	}
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
