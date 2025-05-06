package client

import (
	"log"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/webrtc"
	"github.com/sushiag/go-webrtc-signaling-server/client/websocket"
)

type Wrapper struct {
	Client      *websocket.Client
	PeerManager *webrtc.PeerManager
	IsHost      bool
}

// creates a new wrapper that sets up both signaling and WebRTC handling
func NewClient(wsEndpoint string) *Wrapper {
	client := websocket.NewClient(wsEndpoint)
	pm := webrtc.NewPeerManager()

	w := &Wrapper{
		Client:      client,
		PeerManager: pm,
	}

	client.SetMessageHandler(func(msg websocket.Message) {
		// this aass all signaling messages to the PeerManager
		pm.HandleSignalingMessage(msg, client)
	})

	return w
}

// this performs the authentication and WebSocket initialization.
func (w *Wrapper) Connect() error {
	if err := w.Client.PreAuthenticate(); err != nil {
		return err
	}
	if err := w.Client.Init(); err != nil {
		return err
	}
	return nil
}

func (w *Wrapper) CreateRoom() error {
	w.IsHost = true
	log.Println("[CLIENT] Set as host after creating room.")
	return w.Client.Create()
}

// joins an existing room
func (w *Wrapper) JoinRoom(roomID string) error {
	// if the client is joining a room, they are no longer the host
	w.IsHost = false
	return w.Client.JoinRoom(roomID)
}

// sends the start-session signal to kick of exchanigng
func (w *Wrapper) StartSession() error {
	return w.Client.StartSession()
}

// sends a string message over the WebRTC DataChannel to a specific peer
func (w *Wrapper) SendMessageToPeer(peerID uint64, msg string) error {
	return w.PeerManager.SendDataToPeer(peerID, []byte(msg))
}

// disconnects and removes a specific peer connection
func (w *Wrapper) LeaveRoom(peerID uint64) {
	w.PeerManager.RemovePeer(peerID)
}

func (cw *Wrapper) CloseServer() {
	// only allow the host to close the server and transition to P2P
	if cw.IsHost {
		if cw.PeerManager != nil && cw.Client != nil {
			// disconnect all peers and transition to P2P
			cw.PeerManager.CheckAllConnectedAndDisconnect(cw.Client)
		}
	} else {
		// log or handle error if non-host tries to close the server
		log.Println("Error: Non-host client cannot close the signaling server.")
	}
}

// close cleanly shuts down all connections and peer sessions
func (w *Wrapper) Close() {
	w.Client.Close()
	w.PeerManager.CloseAll()
}

// for testing
func (w *Wrapper) SetServerURL(url string) {
	w.Client.SetServerURL(url)
}

// for testing
func (w *Wrapper) SetApiKey(key string) {
	w.Client.SetApiKey(key)
}
