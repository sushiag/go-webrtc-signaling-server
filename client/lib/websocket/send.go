package websocket

import (
	"fmt"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
)

func (c *Client) Send(msg Message) error {
	select {
	case c.sendQueue <- msg:
		// NOTE: this is what causes us to need sleeps in the tests
		//
		// TODO: 
		// wait for the response here before returning so we dont need to sleep for a
		// guesstimated time in the tests.
		return nil
	case <-c.doneCh:
		return fmt.Errorf("client is closed")
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
