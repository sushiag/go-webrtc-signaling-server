package ws_messages

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/require"
)

func TestMessageMarshalling(t *testing.T) {
	msgToSend := MessageAnyPayload{
		MsgType: SDP,
		From:    1,
		To:      2,
		Payload: SDPPayload{
			SDP: webrtc.SessionDescription{Type: webrtc.SDPTypeOffer},
		},
	}

	// Marshal message
	jsonMsg, jsonMarshalErr := json.Marshal(msgToSend)
	require.NoError(t, jsonMarshalErr)

	expectedJSON := fmt.Sprintf(`{"type":%d,"from":1,"to":2,"payload":{"sdp":{"type":"offer","sdp":""}}}`, SDP)
	require.JSONEq(t, expectedJSON, string(jsonMsg), "Marshalled JSON mismatch")

	// Unmarshal message
	var msgToReceive MessageRawJSONPayload
	unmarshalMsgErr := json.Unmarshal(jsonMsg, &msgToReceive)
	require.NoError(t, unmarshalMsgErr)
	require.Equal(t, msgToSend.MsgType, msgToReceive.MsgType)
	t.Logf("msgToReceive type: %s", msgToReceive.MsgType.AsString())
	t.Logf("msgToReceive payload: %s", string(msgToReceive.Payload))
	require.Equal(t, "", msgToReceive.Error)

	// Unmarshal payload
	var receivedPayload SDPPayload
	unmarshalPayloadErr := json.Unmarshal(msgToReceive.Payload, &receivedPayload)
	require.NoError(t, unmarshalPayloadErr)
	require.Equal(t, msgToSend.Payload, receivedPayload)
}
