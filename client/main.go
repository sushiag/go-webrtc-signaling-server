package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

// message struct for websocket connection
type Message struct {
	Type    string `json:"type"`
	RoomID  uint32 `json:"roomId"`
	Sender  uint64 `json:"sender"`
	Target  uint64 `json:"target,omitempty"`
	Content string `json:"content"`
}

var (
	apiKey        = "my-api-key"
	roomID uint32 = 1001
	userID uint64 = 92633
	conn   *websocket.Conn
	peers  = make(map[uint64]*webrtc.PeerConnection)
)

// this connects to the websocket server
func connectWebsocket(serverUrl string, apikey string, userID uint64) (*websocket.Conn, error) { // added apikey
	headers := http.Header{}
	headers.Set("X-Api-Key", apikey) // get API key from list
	headers.Set("X-User-ID", fmt.Sprintf("%d", userID))

	conn, _, err := websocket.DefaultDialer.Dial(serverUrl, headers)
	if err != nil {
		return nil, fmt.Errorf("[Client] failed to connect to websocket: %v", err)
	}

	fmt.Println("Connected to websocket server:", serverUrl)
	return conn, nil
}

// init webrtc connection
func createPeerConnection(remoteID uint64) (*webrtc.PeerConnection, error) { // using defaul stun no turn server
	stunServer := "stun:stun.1.google.com:19302"

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{stunServer}},
		},
	}

	// create peerconnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Failed to create peer connection: %v", err)
	}
	// handles ice candidatesdidate)
	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			candidate, _ := json.Marshal(c.ToJSON())

			msg := Message{
				Type:    "ice-candidate",
				RoomID:  roomID,
				Sender:  uint64(userID),
				Target:  remoteID,
				Content: string(candidate),
			}
			conn.WriteJSON(msg) // sends ice candidate via webscoket connection
		}
	})
	peerConnection.OnNegotiationNeeded(func() {
		log.Println("[WERTC] Negotiation needed with", remoteID)
		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			log.Println("[Error] failed to create offer:", err)
			return
		}

		err = peerConnection.SetLocalDescription(offer)
		if err != nil {
			log.Println("[ERROR] Failed to set local description", err)
			return
		}
		msg := Message{
			Type:    "offer",
			RoomID:  roomID,
			Sender:  userID,
			Target:  remoteID,
			Content: offer.SDP,
		}
		conn.WriteJSON(msg) // send sdp offer via websocket connection
	})

	// handles wertc tracks for media use
	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("[WebRTC] new track received: %s\n", track.Kind())
	})

	// handles ice connection state change
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[ICE] Connection state change to: %s", state.String())

		if state == webrtc.ICEConnectionStateFailed {
			log.Println("[ICE] connection failed. Restarting ICE..")
			peerConnection.CreateOffer(&webrtc.OfferOptions{ICERestart: true})
		}
	})
	return peerConnection, nil
}

func handleMessage(msg Message) {
	if msg.Sender == userID {
		return
	}

	peerConnection, exists := peers[msg.Sender]
	if !exists {
		log.Println("[WEBRTC] New peer:", msg.Sender)
		NewPeerConnection, err := createPeerConnection(msg.Sender)
		if err != nil {
			log.Println("Error creating peer connection", err)
			return
		}
		peers[msg.Sender] = NewPeerConnection
		peerConnection = NewPeerConnection
	}

	switch msg.Type {
	case "offer":
		log.Println("[SIGNAL] Received offer from", msg.Sender)
		peerConnection.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: msg.Content})

		answer, _ := peerConnection.CreateAnswer(nil)
		peerConnection.SetLocalDescription(answer)

		reply := Message{
			Type:    "answer",
			RoomID:  roomID,
			Sender:  userID,
			Target:  msg.Sender,
			Content: answer.SDP,
		}
		conn.WriteJSON(reply)

	case "answer":
		log.Println("[SIGNAL] Received answer from", msg.Sender)
		peerConnection.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: msg.Content})

	case "ice-candidate":
		var candidate webrtc.ICECandidateInit
		json.Unmarshal([]byte(msg.Content), &candidate)
		peerConnection.AddICECandidate(candidate)

	}
}

func main() {
	// websocket server url
	defaultPort := "8080"
	port := os.Getenv("WS_PORT") // adjust path as needed to
	if port == "" {
		port = defaultPort
	}
	serverURL := fmt.Sprintf("ws://localhost:%s/ws", port)

	// websocket create room
	var err error
	conn, err := connectWebsocket(serverURL, apiKey, userID)
	if err != nil {
		log.Fatal("failed to connect", err)
	}
	defer conn.Close()

	// to close system
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	// goruotine to read messages from the server
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("[ERROR] Read error:", err)
				return
			}
			fmt.Println("[SERVER] Received:", string(message))

			// handles signaling messages
			var msg Message
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Println("[ERROR] Failed to parse WebSocket message:", err)
				continue
			}
			handleMessage(msg)
		}
	}()

	// send periodic messages to WebSocket
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			msg := "Hello from WebSocket client!"
			conn.WriteMessage(websocket.TextMessage, []byte(msg))
			fmt.Println("[CLIENT] Sent:", msg)

		case <-interrupt:
			fmt.Println("[SYSTEM] Received interrupt, closing connection...")
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Goodbye!"))
			return
		}
	}
}
