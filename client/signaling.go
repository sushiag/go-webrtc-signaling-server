package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"

	sm "signaling-msgs"
)

type SignalingEvent any

type RoomCreatedEvent struct {
	RoomID uint64
}

type RoomJoinedEvent struct {
	RoomID uint64
}

func newSignalingManager(wsEndpoint string, apiKey string) (*signalingManager, error) {
	mngr := &signalingManager{
		clients:        make(map[uint64]*webRTCPeerManager, 2),
		sdpSignalingCh: make(chan sdpSignalingRequest, 10),
		iceSignalingCh: make(chan iceSignalingRequest, 10),
	}

	headers := http.Header{"X-Api-Key": []string{apiKey}}
	wsConn, resp, err := websocket.DefaultDialer.Dial(wsEndpoint, headers)
	if err != nil {
		log.Printf("[DEBUG] http response after failing to establish WS connection:\n%v", resp)
		return nil, err
	}

	clientIDStr := resp.Header.Get("X-Client-ID")
	clientID, err := strconv.ParseUint(clientIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("got an invalid client ID from the server: %s", clientIDStr)
	}
	mngr.wsClientID = clientID

	wsSendCh := make(chan sm.Message, 32)
	signalingEventCh := make(chan SignalingEvent, 32)

	// WS read loop
	go func() {
		for {
			var msg sm.Message
			err := wsConn.ReadJSON(&msg)
			if err != nil {
				log.Printf("[ERROR] failed to read incoming WS message: %v", err)
			}
			log.Printf("[DEBUG] received WS message with type: '%d'", msg.MsgType)

			switch msg.MsgType {
			case sm.Ping:
				{
					log.Printf("[INFO] got ping from server!")
					wsSendCh <- sm.Message{MsgType: sm.Pong}
				}
			case sm.Pong:
				{
					log.Printf("[INFO] got pong from server!")
				}
			case sm.RoomCreated:
				{
					var payload sm.RoomCreatedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						log.Printf("[ERROR] failed to unmarshal RoomCreated message payload: %v", err)
						continue
					}

					signalingEventCh <- RoomCreatedEvent{RoomID: payload.RoomID}
				}
			case sm.RoomJoined:
				{
					var payload sm.RoomJoinedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						log.Printf("[ERROR] failed to unmarshal RoomJoined message payload: %v", err)
						continue
					}

					signalingEventCh <- RoomJoinedEvent{RoomID: payload.RoomID}
				}
			}
		}
	}()

	// WS send loop
	go func() {
		for {
			select {
			case msgToSend := <-wsSendCh:
				{
					err := wsConn.WriteJSON(msgToSend)
					if err != nil {
						log.Printf("[ERROR] failed to send WS message: %v", err)
					}

					log.Printf("[DEBUG] sent WS message with type: %d", msgToSend.MsgType)
				}
			case sdpReq := <-mngr.sdpSignalingCh:
				{
					payload, marshalPayloadErr := json.Marshal(sm.SDPPayload{
						SDP: sdpReq.sdp.SDP,
						For: sdpReq.to,
					})
					if marshalPayloadErr != nil {
						log.Printf("[ERROR] failed to JSON marshal SDP message: %v", marshalPayloadErr)
					}

					msg := sm.Message{
						MsgType: sm.SDP,
						Payload: payload,
					}
					wsWriteErr := wsConn.WriteJSON(msg)
					if wsWriteErr != nil {
						log.Printf("[ERROR] failed to send SDP message: %v", marshalPayloadErr)
					}

					log.Printf("[DEBUG] sent SDP message")
				}
			case iceReq := <-mngr.iceSignalingCh:
				{
					payload, marshalPayloadErr := json.Marshal(sm.ICECandidatePayload{
						ICE: iceReq.iceCandidate.ToJSON().Candidate,
						For: iceReq.to,
					})
					if marshalPayloadErr != nil {
						log.Printf("[ERROR] failed to JSON marshal ICE candidate: %v", marshalPayloadErr)
					}

					msg := sm.Message{
						MsgType: sm.SDP,
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
	mngr.signalingEventCh = signalingEventCh

	return mngr, nil
}
