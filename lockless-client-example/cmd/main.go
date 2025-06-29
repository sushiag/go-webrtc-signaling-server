package main

import (
	"fmt"
	"math/rand"

	. "go_example/lib"
)

func main() {
	// establish the websocket here
	wsConn := MockGorillaClient{}

	// this is our library's client
	client := NewClient(wsConn, true)

	for {
		// here's how we receive
		msg, hasMsg := client.RecvMsg()
		if hasMsg {
			fmt.Printf("GOT from peer %d: %s\n", msg.PeerID, msg.Content)
		}

		if rand.Float32() > 0.5 {
			// here's how we send
			client.SendMsg(WebRTCMsg{PeerID: msg.PeerID, Content: "sucka"})
		}
	}
}
