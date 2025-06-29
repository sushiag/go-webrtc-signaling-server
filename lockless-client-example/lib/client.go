package lib

import (
	"fmt"
	"time"
)

// Client handle
type Client struct {
	sendCh    chan WebRTCMsg
	receiveCh chan WebRTCMsg
}

type ConnectedPeer struct {
	peerID       int
	WebRTCSendCh chan WebRTCMsg
}

func NewClient(wsClient MockGorillaClient, isHost bool) Client {
	// used to talk to the server
	wsMsgSendCh := make(chan websocketMsg)

	// Peer handler thread
	//
	// NOTE: if we want to access data from the peer handler, for example to do the following:
	// - kick peers
	// - get the peer ids
	// - etc.
	// we can always make 2 more channels to issue and receive "commands"
	newConnectedPeerCh := make(chan ConnectedPeer)
	allPeersRecvCh := make(chan WebRTCMsg) // ch for
	clientSendCh := make(chan WebRTCMsg)
	go func() {
		connectedPeers :=
			make(map[int]ConnectedPeer)
		for {
			select {
			case new_connected_peer := <-newConnectedPeerCh:
				{
					connectedPeers[new_connected_peer.peerID] = new_connected_peer
				}
			case msg := <-clientSendCh:
				{
					peer, hasPeer := connectedPeers[msg.PeerID]
					if hasPeer {
						peer.WebRTCSendCh <- msg
					} else {
						fmt.Println("Tried to send message to unknown peer:", msg.PeerID)
					}
				}
			}
		}
	}()

	// here we process the WS messages
	go func() {
		// notice that this thread owns this map so there's no chance
		// of concurrent access
		pendingConnections := make(map[int]webRTCClient)

		for {
			msg, err := wsClient.ReadMessage()
			if err != nil {
				println("failed to read WS message")
			}

			switch msg.msgType {

			case WSClientJoinedRoom:
				{
					if !isHost {
						// we don't care if we're not the host
						continue
					}

					// Start Establishing WebRTC connection

					// 1. get the client's id
					recipientID := msg.clientID

					// 2. make the client
					webRTCClient := newWebRTCClient(msg.clientID)

					// 3. Send and SDP offer
					wsMsgSendCh <- websocketMsg{
						WsSDPOffer,
						recipientID,
						webRTCClient.createOffer(),
					}

					// 3. store the pending connection
					pendingConnections[recipientID] = webRTCClient
				}

			case WsSDPOffer:
				{
					if !isHost {
						fmt.Println("im not the host bruh!!! know your place")
					}

					senderID := msg.clientID

					conn, ok := pendingConnections[senderID]
					if !ok {
						fmt.Println("who does this guy think he is? i dunno him")
					}

					conn.receiveOffer(msg.content)
				}

			case WsSDPResponse:
				{
					if !isHost {
						fmt.Println("im not the host bruh? what do i do with this")
					}

					senderID := msg.clientID

					conn, ok := pendingConnections[senderID]
					if !ok {
						fmt.Println("who does this guy think he is? i dunno him")
					}

					conn.receiveResponse(msg.content)
				}

			case WsSDPICE:
				{
					senderID := msg.clientID

					conn, ok := pendingConnections[senderID]
					if !ok {
						fmt.Println("who does this guy think he is? i dunno him")
					}

					conn.receiveICE(msg.content)

					// once the connection is established, we can handle this
					// on a separate thread that no longer need to meddle with
					// the websocket state
					if conn.isFinishedExchanging() {
						// we create a new send channel for this specific peer
						newPeerSendCh := make(chan WebRTCMsg)

						go func() {
							for {
								select {
								case msgToSend := <-newPeerSendCh:
									{
										conn.sendMessage(msgToSend.Content)
									}
								default:
									{
										msg := conn.readMessage()
										allPeersRecvCh <- msg
									}
								}
							}
						}()

						delete(pendingConnections, senderID)
						newConnectedPeerCh <- ConnectedPeer{conn.peerID, newPeerSendCh}
					}
				}

			// this will only happen while the connection is pending
			case WSClientLeftRoom:
				{
					delete(pendingConnections, msg.clientID)
				}
			}

			// then we wait for a response
			resp := <-wsMsgSendCh

			// let the gorilla send it
			wsClient.WriteMessage(resp)

			// so we dont flood the test output
			time.Sleep(time.Second)
		}
	}()

	return Client{sendCh: clientSendCh, receiveCh: allPeersRecvCh}
}

func (c Client) SendMsg(msg WebRTCMsg) {
	c.sendCh <- msg
}

// Pops one message from the recv channel
//
// It will be the client's responsibility to create a loop the gets all the messages
func (c Client) RecvMsg() (WebRTCMsg, bool) {
	select {
	case msg := <-c.receiveCh:
		{
			return msg, true
		}
	default:
		{
			var msg WebRTCMsg
			return msg, false
		}
	}
}
