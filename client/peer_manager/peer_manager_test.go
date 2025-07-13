package peer_manager

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	smsg "signaling-msgs"
)

type signalingChannels struct {
	in  chan smsg.MessageRawJSONPayload
	out chan smsg.MessageAnyPayload
}

func TestPeerDataExchange(t *testing.T) {
	signalingIn1 := make(chan smsg.MessageRawJSONPayload)
	signalingOut1 := make(chan smsg.MessageAnyPayload)
	signalingIn2 := make(chan smsg.MessageRawJSONPayload)
	signalingOut2 := make(chan smsg.MessageAnyPayload)
	signalingChannels := map[uint64]signalingChannels{
		1: {signalingIn1, signalingOut1},
		2: {signalingIn2, signalingOut2},
	}
	startMockSignalingServer(t, signalingChannels)

	client1 := NewPeerManager(signalingIn1, signalingOut1)
	client2 := NewPeerManager(signalingIn2, signalingOut2)

	// Start SDP exchange:
	// - Signaling starts when a client joins a room and receiveds the clients list.
	// - The joining client will send an SDP offer to all the clients in the list.
	roomJoinedPayload := smsg.RoomJoinedPayload{
		RoomID:        1,
		ClientsInRoom: []uint64{1},
	}
	rawPayload, jsonMarshalErr := json.Marshal(roomJoinedPayload)
	require.NoError(t, jsonMarshalErr)
	signalingIn2 <- smsg.MessageRawJSONPayload{
		MsgType: smsg.RoomJoined,
		Payload: rawPayload,
	}
	t.Log("sent room joined message to client 2")

	// Wait for DATA channels to open
	t.Log("waiting for data channels to open...")
	deadline := time.After(time.Second * 3)
	channelsOpened := 0
	for channelsOpened < 2 {
		select {
		case openedFor := <-client1.dataChOpened:
			{
				require.Equal(t, uint64(2), openedFor)
				channelsOpened += 1
				t.Log("data channel for client 2 opened")
			}
		case openedFor := <-client2.dataChOpened:
			{
				require.Equal(t, uint64(1), openedFor)
				channelsOpened += 1
				t.Log("data channel for client 1 opened")
			}
		case <-deadline:
			{
				t.Fatalf("clients took too long to establish connection")
			}
		}
	}
	t.Log("data channels opened!")

	// Data exchange start
	t.Log("starting data exchange...")
	client1.SendDataToPeer(2, []byte("hello from client 1"))
	client2.SendDataToPeer(1, []byte("hello from client 2"))

	require.Equal(t, PeerDataMsg{from: 2, data: []byte("hello from client 2")}, <-client1.GetPeerDataMsgCh())
	require.Equal(t, PeerDataMsg{from: 1, data: []byte("hello from client 1")}, <-client2.GetPeerDataMsgCh())
	t.Log("clients finished exchanging data")
	// Data exchange end
}

func startMockSignalingServer(t *testing.T, channels map[uint64]signalingChannels) {
	for clientID, clientCh := range channels {
		// Signaling output
		go func() {
			t.Logf("started mock signaling loop for client %d", clientID)
			for outMsg := range clientCh.out {
				target, exists := channels[outMsg.To]
				require.Truef(t, exists, "client %d tried to send a message to an unknown client: %d", clientID, outMsg.To)

				rawPayload, marshalErr := json.Marshal(outMsg.Payload)
				require.NoError(t, marshalErr)

				t.Logf("signaling %s msg %d->%d", outMsg.MsgType.AsString(), clientID, outMsg.To)
				target.in <- smsg.MessageRawJSONPayload{
					MsgType: outMsg.MsgType,
					From:    clientID,
					Payload: rawPayload,
				}
			}
		}()
	}
}
