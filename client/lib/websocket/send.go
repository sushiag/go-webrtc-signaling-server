package websocket

import (
	"fmt"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/common"
)

func (c *Client) SendDataToPeer(targetID uint64, data []byte) {
	msg := Message{
		Type:    common.MessageTypeSendMessage,
		Target:  targetID,
		RoomID:  c.RoomID,
		Sender:  c.UserID,
		Payload: Payload{DataType: "binary", Data: data},
	}

	c.SendWSMsgCh <- msg
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
		return fmt.Errorf("unsupported message type: %d", msgType)
	}

	c.SendWSMsgCh <- msg

	return nil
}
