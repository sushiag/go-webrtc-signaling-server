package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"

	smsg "signaling-msgs"
)

func newSignalingManager(wsEndpoint string, apiKey string, pm *peerManager, eventsCh chan<- Event) (*signalingManager, error) {
	mngr := &signalingManager{
		clients: make(map[uint64]*peerManager, 2),
	}

	// connect to the WS endpoint
	headers := http.Header{"X-Api-Key": []string{apiKey}}
	wsConn, resp, err := websocket.DefaultDialer.Dial(wsEndpoint, headers)
	if err != nil {
		log.Printf("[DEBUG] http response after failing to establish WS connection:\n%v", resp)
		return nil, err
	}

	// store the WS Client ID
	clientIDStr := resp.Header.Get("X-Client-ID")
	clientID, err := strconv.ParseUint(clientIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("got an invalid client ID from the server: %s", clientIDStr)
	}
	mngr.wsClientID = clientID

	wsSendCh := make(chan smsg.MessageAnyPayload, 32) // WS output channel

	// WS read loop
	go func() {
		for {
			var msg smsg.MessageRawJSONPayload
			err := wsConn.ReadJSON(&msg)
			if err != nil {
				log.Printf("[ERROR] failed to read incoming WS message: %v", err)
			}
			log.Printf("[DEBUG] received WS message with type: '%s'", msg.MsgType.AsString())

			switch msg.MsgType {
			case smsg.Ping:
				{
					log.Printf("[DEBUG] got ping from server")
					wsSendCh <- smsg.MessageAnyPayload{MsgType: smsg.Pong}
					eventsCh <- ServerPingEvent{}
				}
			case smsg.Pong:
				{
					log.Printf("[DEBUG] got pong from server!")
					eventsCh <- ServerPongEvent{}
				}
			case smsg.RoomCreated:
				{
					var payload smsg.RoomCreatedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						log.Printf("[ERROR] failed to unmarshal RoomCreated message payload: %v", err)
						return
					}

					eventsCh <- RoomCreatedEvent{RoomID: payload.RoomID}
				}
			case smsg.RoomJoined:
				{
					var payload smsg.RoomJoinedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						log.Printf("[ERROR] failed to unmarshal RoomJoined message payload: %v", err)
						return
					}

					// tell the peer manager to make an offer
					if pm != nil {
						for _, clientID := range payload.ClientsInRoom {
							pm.newPeerOffer(clientID)
						}
					}

					eventsCh <- RoomJoinedEvent{RoomID: payload.RoomID, ClientsInRoom: payload.ClientsInRoom}
				}
			default:
				{
					log.Printf("[WARN] unhandled message type: '%s'", msg.MsgType.AsString())
				}
			}
		}
	}()

	// WS send loop
	var sdpCh <-chan sendSDP
	var iceCh <-chan sendICECandidate
	if pm != nil {
		sdpCh = pm.sdpCh
		iceCh = pm.iceCh
	}
	go func() {
		for {
			select {
			case msgToSend := <-wsSendCh:
				{
					err := wsConn.WriteJSON(msgToSend)
					if err != nil {
						log.Printf("[ERROR] failed to send WS message: %v", err)
					}

					log.Printf("[DEBUG] sent WS message with type: %s", msgToSend.MsgType.AsString())
				}
			case sdpReq := <-sdpCh:
				{
					payload, marshalPayloadErr := json.Marshal(smsg.SDPPayload{
						SDP: sdpReq.sdp,
						To:  sdpReq.to,
					})
					if marshalPayloadErr != nil {
						log.Printf("[ERROR] failed to JSON marshal SDP message: %v", marshalPayloadErr)
					}

					msg := smsg.MessageAnyPayload{
						MsgType: smsg.SDP,
						Payload: payload,
					}
					wsWriteErr := wsConn.WriteJSON(msg)
					if wsWriteErr != nil {
						log.Printf("[ERROR] failed to send SDP message: %v", marshalPayloadErr)
					}

					log.Printf("[DEBUG] sent SDP message")
				}
			case iceReq := <-iceCh:
				{
					payload, marshalPayloadErr := json.Marshal(smsg.ICECandidatePayload{
						ICE: iceReq.iceCandidate,
						To:  iceReq.to,
					})
					if marshalPayloadErr != nil {
						log.Printf("[ERROR] failed to JSON marshal ICE candidate: %v", marshalPayloadErr)
					}

					msg := smsg.MessageAnyPayload{
						MsgType: smsg.SDP,
						Payload: payload,
					}
					wsWriteErr := wsConn.WriteJSON(msg)
					if wsWriteErr != nil {
						log.Printf("[ERROR] failed to send ICE candidate: %v", marshalPayloadErr)
					}

					log.Printf("[DEBUG] sent ICE candidate")
				}
			}
		}
	}()

	mngr.wsSendCh = wsSendCh

	return mngr, nil
}
