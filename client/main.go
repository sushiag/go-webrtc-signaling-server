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
	RoomID  string `json:"roomId"`
	Sender  string `json:"sender"`
	Content string `json:"content"`
}

// this connects to the websocket server then creates a new room
func CreateRoom(serverUrl, apikey string) (*websocket.Conn, error) { // added apikey
	headers := http.Header{}
	headers.Set("X-Api-Key", apikey) // get API key from list
	conn, _, err := websocket.DefaultDialer.Dial(serverUrl, headers)
	if err != nil {
		return nil, fmt.Errorf("[Client] failed to connect to websocket: %v", err)
	}

	fmt.Println("Connected to websocket server:", serverUrl)
	return conn, nil
}

// connectcs to an existing room
func JoinRoom(serverUrl, roomID, apiKey string) (*websocket.Conn, error) {
	headers := http.Header{}
	headers.Set("X-API-Key", apiKey) // apikey from list
	headers.Set("X-Room-ID", roomID)

	conn, _, err := websocket.DefaultDialer.Dial(serverUrl, headers)
	if err != nil {
		return nil, fmt.Errorf("[Joining room] failed to join room: %v", err)
	}
	fmt.Println("[Joining room] Success in joining room:", serverUrl)
	return conn, nil
}

// init webrtc connection

func InitializePeerConnection(conn *websocket.Conn, roomID string) (*webrtc.PeerConnection, error) { // using defaul stun no turn server
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
			candidate := c.ToJSON()
			candidateJSON, _ := json.Marshal(candidate)

			msg := Message{
				Type:    "ice-candidate",
				RoomID:  roomID,
				Content: string(candidateJSON),
			}
			conn.WriteJSON(msg) // sends ice candidate via webscoket connection
		}
	})
	peerConnection.OnNegotiationNeeded(func() {
		log.Println("[SDP] Negotiation needed, creating an offer..")

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

func main() {
	// websocket server url
	defaultPort := "8080"
	port := os.Getenv("WS_PORT") // adjust path as needed to
	if port == "" {
		port = defaultPort
	}
	serverURL := fmt.Sprintf("ws://localhost:%s/ws", port)
	// to retrieve from config file
	apiKey := "my-api-key"
	roomID := "test-room"

	// websocket create room
	conn, err := CreateRoom(serverURL, apiKey)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	peerConnection, err := InitializePeerConnection(conn, roomID)
	if err != nil {
		log.Fatal("[ERROR] failed to initialize WebRTC:", err)
	}
	defer peerConnection.Close()
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

			switch msg.Type {
			case "offer":
				log.Println("[SIGNALING] Received offer, creating answer...")
				offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: msg.Content}
				peerConnection.SetRemoteDescription(offer)

				answer, err := peerConnection.CreateAnswer(nil)
				if err != nil {
					log.Println("[ERROR] Failed to create answer:", err)
					continue
				}

				peerConnection.SetLocalDescription(answer)

				response := Message{Type: "answer", RoomID: msg.RoomID, Sender: msg.Sender, Content: answer.SDP}
				conn.WriteJSON(response)

			case "answer":
				log.Println("[SIGNALING] Received answer")
				answer := webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: msg.Content}
				peerConnection.SetRemoteDescription(answer)

			case "ice-candidate":
				log.Println("[SIGNALING] Received ICE candidate")
				var candidate webrtc.ICECandidateInit
				json.Unmarshal([]byte(msg.Content), &candidate)
				peerConnection.AddICECandidate(candidate)
			}
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
