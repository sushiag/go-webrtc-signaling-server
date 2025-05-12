package websocket

import (
	"fmt"
	"log"
)

func (c *Client) StartSender() {
	go func() {
		for {
			select {
			case msg := <-c.sendChan:
				if err := c.WriteMessage(msg); err != nil {
					log.Printf("[CLIENT SIGNALING] Failed to send '%s': %v", msg.Type, err)
					return
				}
			case <-c.doneChan:
				return
			}
		}
	}()
}
func (c *Client) Send(msg Message) error {
	if c.isClosed {
		return fmt.Errorf("connection is closed")
	}
	select {
	case c.sendChan <- msg:
		return nil
	case <-c.doneChan:
		return fmt.Errorf("client is closed")
	}
}

func (c *Client) WriteMessage(msg Message) error {
	if c.isClosed {
		return fmt.Errorf("connection is closed")
	}
	if msg.RoomID == 0 {
		msg.RoomID = c.RoomID
	}
	if msg.Sender == 0 {
		msg.Sender = c.UserID
	}
	return c.Conn.WriteJSON(msg)
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
	// Build the message
	msg, err := c.buildMessage(targetID, msgType, sdpOrCandidate)
	if err != nil {
		return err
	}

	// Send the message through the WebSocket
	return c.Send(msg)
}

// buildMessage constructs a signaling message based on the provided type and data
func (c *Client) buildMessage(targetID uint64, msgType, sdpOrCandidate string) (Message, error) {
	// Create the base message
	msg := Message{
		Type:   msgType,
		Target: targetID,
		RoomID: c.RoomID,
		Sender: c.UserID,
	}

	// Set message specific fields based on message type
	switch msgType {
	case MessageTypeOffer, MessageTypeAnswer:
		msg.SDP = sdpOrCandidate
	case MessageTypeICECandidate:
		msg.Candidate = sdpOrCandidate
	case MessageTypeSendMessage:
		msg.Content = sdpOrCandidate
	default:
		return Message{}, fmt.Errorf("unsupported message type: %s", msgType)
	}

	return msg, nil
}
