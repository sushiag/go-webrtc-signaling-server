package websocket

import (
	"fmt"
	"log"
)

func (c *Client) Send(msg Message) error {
	if c.isClosed {
		return fmt.Errorf("connection is closed")
	}
	c.SendMutex.Lock()
	defer c.SendMutex.Unlock()

	if msg.RoomID == 0 {
		msg.RoomID = c.RoomID
	}
	if msg.Sender == 0 {
		msg.Sender = c.UserID
	}

	if err := c.Conn.WriteJSON(msg); err != nil {
		log.Printf("[CLIENT SIGNALING] Failed to send '%s': %v", msg.Type, err)
		return err
	}
	return nil
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
