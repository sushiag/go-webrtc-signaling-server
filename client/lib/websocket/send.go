package websocket

import (
	"fmt"
	"log"
)

func (c *Client) Send(msg Message) error {
	// TODO: instead of calling creating a new goroutine forsendLoop here,
	// just start running sendLoop when the connection is made so we don't
	// need to start a new goroutine every time we want to send a single message.
	//
	// then sendLoop will just be an infinite loop which waits for new messages
	// through a channel.

	if !c.isSendLoopStarted {
		c.isSendLoopStarted = true
		go c.sendLoop()
	}

	select {
	case c.sendQueue <- msg:
		return nil
	case <-c.doneCh:
		return fmt.Errorf("client is closed")
	}
}

// TODO: the reason why we need this is because the "listening" goroutine is created
// after every message?
// 
// what if we just create one goroutine when the connection is made then it also has
// a loop in it that always listens for messages... basically a worker
func (c *Client) maybeStartListen() {
	if !c.isListenStarted {
		c.isListenStarted = true
		go c.listen()
	}
}

func (c *Client) sendLoop() {
	for {
		select {
		case msg := <-c.sendQueue:
			if c.Conn == nil || c.isClosed {
				log.Printf("[CLIENT SIGNALING] Cannot send, connection is closed.")
				continue
			}
			if msg.RoomID == 0 {
				msg.RoomID = c.RoomID
			}
			if msg.Sender == 0 {
				msg.Sender = c.UserID
			}
			err := c.Conn.WriteJSON(msg)
			if err != nil {
				log.Printf("[CLIENT SIGNALING] Failed to send '%s': %v", msg.Type, err)
			}
		case <-c.doneCh:
			return
		}
	}
}

func (c *Client) SendDataToPeer(targetID uint64, data []byte) error {
	return c.Send(Message{
		Type:    MessageTypeSendMessage,
		Target:  targetID,
		RoomID:  c.RoomID,
		Sender:  c.UserID,
		Payload: Payload{DataType: "binary", Data: data},
	})
}

func (c *Client) SendSignalingMessage(targetID uint64, msgType, sdpOrCandidate string) error {
	msg := Message{
		Type:   msgType,
		Target: targetID,
		RoomID: c.RoomID,
		Sender: c.UserID,
	}

	switch msgType {
	case MessageTypeOffer, MessageTypeAnswer:
		msg.SDP = sdpOrCandidate
	case MessageTypeICECandidate:
		msg.Candidate = sdpOrCandidate
	case MessageTypeSendMessage:
		msg.Content = sdpOrCandidate
	default:
		return fmt.Errorf("unsupported message type: %s", msgType)
	}

	return c.Send(msg)
}
