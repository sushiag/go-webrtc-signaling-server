package signaling_client

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"

	smsg "signaling-msgs"
)

type SignalingClient struct {
	ClientID     uint64
	SignalingIn  <-chan smsg.MessageRawJSONPayload
	SignalingOut chan<- smsg.MessageAnyPayload
}

const apiKeyEnvName = "API_KEY"

func NewSignalingClient(wsEndpoint string, apiKey string) (*SignalingClient, error) {
	client := &SignalingClient{}

	if apiKey == "" {
		return nil, fmt.Errorf("the apiKey cannot be an empty string")
	}

	// connect to the WS endpoint
	headers := http.Header{"X-Api-Key": []string{apiKey}}
	wsConn, resp, err := websocket.DefaultDialer.Dial(wsEndpoint, headers)
	if err != nil {
		return nil, err
	}

	// store the WS Client ID
	clientIDStr := resp.Header.Get("X-Client-ID")
	clientID, err := strconv.ParseUint(clientIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("got an invalid client ID from the server: %s", clientIDStr)
	}
	client.ClientID = clientID

	signalingIn := make(chan smsg.MessageRawJSONPayload)
	signalingOut := make(chan smsg.MessageAnyPayload)

	// WS Read Loop
	go func() {
		for {
			var msg smsg.MessageRawJSONPayload
			if err := wsConn.ReadJSON(&msg); err != nil {
				log.Printf("[ERROR] failed to read WS message from server: %v", err)
				continue
			}

			switch msg.MsgType {
			case smsg.Ping:
				signalingOut <- smsg.MessageAnyPayload{MsgType: smsg.Pong}
			default:
				signalingIn <- msg
			}

		}
	}()

	// WS Send Loop
	go func() {
		for msg := range signalingOut {
			if err := wsConn.WriteJSON(msg); err != nil {
				log.Printf("[ERROR] failed to send WS message to server: %v", err)
				continue
			}
		}
	}()

	return client, nil
}
