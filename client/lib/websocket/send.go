package websocket

import (
	"fmt"
	"log"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
)

func (c *Client) Send(msg Message) error {
	select {
	case c.sendQueue <- msg:
		return nil
	case <-c.doneCh:
		return fmt.Errorf("client is closed")
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
				log.Printf("[CLIENT SIGNALING] Failed to send '%s': %v", msg.Type.String(), err)
			}
		case <-c.doneCh:
			return
		}
	}
}

func (c *Client) SendDataToPeer(targetID uint64, data []byte) error {
	return c.Send(Message{
		Type:    common.MessageTypeSendMessage,
		Target:  targetID,
		RoomID:  c.RoomID,
		Sender:  c.UserID,
		Payload: Payload{DataType: "binary", Data: data},
	})
}

func (c *Client) SendSignalingMessage(targetID uint64, msgType common.MessageType, sdpOrCandidate string) error {
	msg := Message{
		Type:   msgType,
		Target: targetID,
		RoomID: c.RoomID,
		Sender: c.UserID,
	}

	switch msgType {
	case common.MessageTypeOffer, common.MessageTypeAnswer:
		msg.SDP = sdpOrCandidate
	case common.MessageTypeICECandidate:
		msg.Candidate = sdpOrCandidate
	case common.MessageTypeSendMessage:
		msg.Content = sdpOrCandidate
	default:
		return fmt.Errorf("unsupported message type: %s", msgType.String())
	}

	return c.Send(msg)
}
