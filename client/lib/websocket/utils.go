package websocket

import "log"

func (c *Client) Close() {
	if !c.isClosed {
		c.isClosed = true
		close(c.doneCh)
		if c.Conn != nil {
			_ = c.Conn.Close()
			c.Conn = nil
		}
		log.Println("[CLIENT SIGNALING] Client closed.")
	}
}

func (c *Client) CloseSignaling() {
	if !c.isClosed {
		c.isClosed = true
		close(c.doneCh)
		if c.Conn != nil {
			_ = c.Conn.Close()
			c.Conn = nil
		}
		log.Println("[CLIENT SIGNALING] Client disconnected from signaling server.")
	}
}
