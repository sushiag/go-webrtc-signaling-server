package server

import (
	"encoding/json"
	"log"

	smsg "signaling-msgs"
)

// This handles the messages from the signaling client
func (wsm *WebSocketManager) handleMessage(msg *smsg.MessageRawJSONPayload) {

	// TODO:
	// - implement a 'close-room' message that the room owner can use
	switch msg.MsgType {
	case smsg.CreateRoom:
		{
			log.Printf("[WS] User %d requested to create a room", msg.From)
			roomID := wsm.createRoom(msg.From)
			resp := smsg.MessageAnyPayload{
				MsgType: smsg.RoomCreated,
				Payload: smsg.RoomCreatedPayload{RoomID: roomID},
			}

			if conn, ok := wsm.Connections[msg.From]; ok {
				_ = wsm.SafeWriteJSON(conn, resp)
			}
		}

	case smsg.JoinRoom:
		{
			var payload smsg.JoinRoomPayload
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				log.Printf("[ERROR] failed to unmarshal join room payload from: %d", msg.From)
				break
			}

			log.Printf("[User %d] requested to join room: %d", msg.From, payload.RoomID)
			wsm.addUserToRoom(payload.RoomID, msg.From)
		}

	case smsg.SDP, smsg.ICECandidate:
		{
			conn, exists := wsm.Connections[msg.To]
			if !exists {
				log.Printf("[WARN] client %d tried to send a %s message to an unknown client: %d", msg.From, msg.MsgType.AsString(), msg.To)
				return
			}

			// TODO(rmarinn): i removed the check if the users are in the same room... need to re-implement it

			log.Printf("[WS] Forwarding %s from %d to %d", msg.MsgType.AsString(), msg.From, msg.To)
			// kinda wasteful casting but i haven't figured out a way to deserialize only the
			// 'MsgType' field... gotta have to deal with the limitation of using JSON messages
			wsm.SafeWriteJSON(conn, smsg.MessageAnyPayload{
				MsgType: msg.MsgType,
				From:    msg.From,
				To:      msg.To,
				Payload: msg.Payload,
			})
		}

	case smsg.LeaveRoom:
		{
			log.Printf("[WS] Disconnect request from user %d", msg.From)
			wsm.disconnectChan <- msg.From
			// TODO: assign a new room owner if the one leaving the current owner
		}

	}
}

// This handles the connected users from the signaling client
func (wsm *WebSocketManager) handleNewConnection(conn *Connection) {
	log.Printf("[WS] User %d connected", conn.UserID)
	conn.Disconnected = wsm.disconnectChan

	conn.Conn.SetPongHandler(func(string) error {
		return nil
	})

	go conn.readLoop(wsm.messageChan)
	go conn.writeLoop()
	log.Printf("[WS] WS read and write loop for user %d started", conn.UserID)

	wsm.Connections[conn.UserID] = conn

}
