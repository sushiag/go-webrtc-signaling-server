package websocket

import (
	"fmt"
	"log"
	"strconv"
)

func (c *Client) Create() error {
	if err := c.Send(Message{Type: MessageTypeCreateRoom}); err != nil {
		log.Printf("Create room failed: %v", err)
		return err
	}
	log.Println("Room creation requested.")
	return nil
}

func (c *Client) JoinRoom(roomID string) error {
	roomIDUint64, err := strconv.ParseUint(roomID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid room ID: %v", err)
	}
	c.RoomID = roomIDUint64
	return c.Send(Message{Type: MessageTypeJoinRoom, RoomID: c.RoomID})
}

func (c *Client) RequestPeerList() {
	if err := c.Send(Message{Type: MessageTypePeerListRequest}); err != nil {
		log.Printf("Peer list request failed: %v", err)
	}
}
