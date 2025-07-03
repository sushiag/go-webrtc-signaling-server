package websocket

import (
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
