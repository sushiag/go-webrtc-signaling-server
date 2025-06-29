package lib

import "fmt"

type WebRTCMsg struct {
	PeerID  int
	Content string // this should be raw bytes/binary on the actual
}

type webRTCClient struct {
	// we will probably store the real client in here
	peerID    int
	connected bool
}

func newWebRTCClient(peerID int) webRTCClient {
	return webRTCClient{peerID, false}
}

func (c webRTCClient) createOffer() string {
	// generate the sdp offer here
	return "here my SDP 8-D"
}

func (c webRTCClient) receiveOffer(sdp string) {
	// TODO: update state internally here
	fmt.Printf("yummy %s\n", sdp)
}

func (c webRTCClient) receiveResponse(sdp string) {
	// TODO: update state internally here
	fmt.Printf("ugh %s\n", sdp)
}

func (c webRTCClient) receiveICE(ice string) {
	// TODO: update state internally here
	fmt.Printf("mmmmmmmm %s\n", ice)
}

func (c webRTCClient) isFinishedExchanging() bool {
	// TODO: actually check
	return true
}

func (c webRTCClient) readMessage() WebRTCMsg {
	// TODO: actually receive
	return WebRTCMsg{c.peerID, "XD"}
}

func (webRTCClient) sendMessage(msg string) {
	// TODO: implement this
}
