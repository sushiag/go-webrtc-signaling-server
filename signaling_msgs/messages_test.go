package ws_messages

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/require"
)

func TestMessageMarshalling(t *testing.T) {
	msgToSend := MessageAny{
		MsgType: SDP,
		Payload: SDPPayload{
			SDP:  webrtc.SessionDescription{Type: webrtc.SDPTypeOffer},
			From: 1,
			To:   2,
		},
	}

	// Marshal message
	jsonMsg, jsonMarshalErr := json.Marshal(msgToSend)
	require.NoError(t, jsonMarshalErr)

	expectedJSON := fmt.Sprintf(`{"type":%d,"payload":{"sdp":{"type":"offer","sdp":""},"from":1,"to":2}}`, SDP)
	require.JSONEq(t, expectedJSON, string(jsonMsg), "Marshalled JSON mismatch")

	// Unmarshal message
	var msgToReceive MessageRawJSON
	unmarshalMsgErr := json.Unmarshal(jsonMsg, &msgToReceive)
	require.NoError(t, unmarshalMsgErr)
	require.Equal(t, msgToSend.MsgType, msgToReceive.MsgType)
	t.Logf("msgToReceive type: %s", msgToReceive.MsgType.AsString())
	t.Logf("msgToReceive payload: %s", string(msgToReceive.Payload))

	// Unmarshal payload
	var receivedPayload SDPPayload
	unmarshalPayloadErr := json.Unmarshal(msgToReceive.Payload, &receivedPayload)
	require.NoError(t, unmarshalPayloadErr)
	require.Equal(t, msgToSend.Payload, receivedPayload)
}
