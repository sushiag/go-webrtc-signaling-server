package signaling_client

import (
	"encoding/json"
	"errors"
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

	// NOTE: it's kinda unsafe to not lock these channels but we like to live dangerously
	// * concurrent access == skill issues
	createRoom chan smsg.MessageRawJSONPayload
	joinRoom   chan smsg.MessageRawJSONPayload
}

type roomCommand struct {
	msg    *smsg.MessageAnyPayload
	respCh chan<- error
}

func NewSignalingClient(wsEndpoint string, apiKey string) (*SignalingClient, error) {
	client := &SignalingClient{}

	if apiKey == "" {
		return nil, fmt.Errorf("the apiKey cannot be an empty string")
	}

	// connect to the WS endpoint
	headers := http.Header{"Authorization": []string{"Bearer " + apiKey}}
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

	signalingIn := make(chan smsg.MessageRawJSONPayload, 32)
	signalingOut := make(chan smsg.MessageAnyPayload, 32)
	client.SignalingIn = signalingIn
	client.SignalingOut = signalingOut

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
				{
					signalingOut <- smsg.MessageAnyPayload{MsgType: smsg.Pong}
				}
			case smsg.RoomCreated:
				{
					if client.createRoom != nil {
						client.createRoom <- msg
					}
				}
			case smsg.RoomJoined:
				{
					// NOTE: we need to send the message to both the signaling channel and create
					// room response channel here
					if client.joinRoom != nil {
						client.joinRoom <- msg
					}
					signalingIn <- msg
				}
			default:
				{
					signalingIn <- msg
				}
			}
		}
	}()

	// WS Send Loop
	go func() {
		for msg := range signalingOut {
			if err := wsConn.WriteJSON(msg); err != nil {
				log.Printf("[ERROR] failed to send WS message to server: %v", err)
			}
			log.Printf("[DEBUG] sent '%s' message to server", msg.MsgType.AsString())
		}
	}()

	return client, nil
}

func (c *SignalingClient) CreateRoom() (uint64, error) {
	var resp smsg.MessageRawJSONPayload
	if c.createRoom == nil {
		c.createRoom = make(chan smsg.MessageRawJSONPayload, 1)
		c.SignalingOut <- smsg.MessageAnyPayload{MsgType: smsg.CreateRoom}
	}

	resp = <-c.createRoom

	var err error
	if resp.Error != "" {
		err = errors.New(resp.Error)
	}

	var respMsg smsg.RoomJoinedPayload
	if err := json.Unmarshal(resp.Payload, &respMsg); err != nil {
		return 0, fmt.Errorf("failed to unmarshal create room response payload: %v", err)
	}

	return respMsg.RoomID, err
}

func (c *SignalingClient) JoinRoom(roomID uint64) ([]uint64, error) {
	var resp smsg.MessageRawJSONPayload
	if c.joinRoom == nil {
		c.joinRoom = make(chan smsg.MessageRawJSONPayload, 1)
		c.SignalingOut <- smsg.MessageAnyPayload{
			MsgType: smsg.JoinRoom,
			Payload: smsg.JoinRoomPayload{
				RoomID: roomID,
			},
		}
	}

	resp = <-c.joinRoom

	var err error
	if resp.Error != "" {
		err = errors.New(resp.Error)
	}

	var respMsg smsg.RoomJoinedPayload
	if err := json.Unmarshal(resp.Payload, &respMsg); err != nil {
		return []uint64{}, fmt.Errorf("failed to unmarshal join room response payload: %v", err)
	}

	if respMsg.RoomID != roomID {
		log.Printf("[WARN] got put in room %d instead of the requestd %d", respMsg.RoomID, roomID)
	}

	return respMsg.ClientsInRoom, err
}

func (c *SignalingClient) LeaveRoom() {
	c.SignalingOut <- smsg.MessageAnyPayload{
		MsgType: smsg.LeaveRoom,
	}
}
