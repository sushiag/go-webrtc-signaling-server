package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
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

type PeerManager struct {
	peers map[uint64]*webrtc.PeerConnection
}

var (
	roomID      uint32
	userID      uint64
	conn        *websocket.Conn
	peerManager = NewPeerManager()
)

func NewPeerManager() *PeerManager {
	return &PeerManager{
		peers: make(map[uint64]*webrtc.PeerConnection),
	}
}

func (pm *PeerManager) Add(userID uint64, pc *webrtc.PeerConnection) {
	pm.peers[userID] = pc
}

func (pm *PeerManager) Get(userID uint64) (*webrtc.PeerConnection, bool) {
	pc, exists := pm.peers[userID]
	return pc, exists
}

func (pm *PeerManager) Remove(userID uint64) {
	delete(pm.peers, userID)
}

// this connects to the websocket server
func connectToWebsocket(serverUrl string, apikey string, userID uint64) (*websocket.Conn, error) { // added apikey
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
		conn.WriteJSON(msg)
		// send sdp offer via websocket connection
	})

	// handles wertc tracks for media use
	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("[WebRTC] new track received: %s\n", track.Kind())
	})

	dataChannel, err := peerConnection.CreateDataChannel("fileTransfer", nil)
	if err != nil {
		log.Fatalf("Failed to create data channel: %v", err)
	}

	dataChannel.OnOpen(func() {
		log.Println("[DATA CHANNEL] Opened")
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("[DATA CHANNEL] Received: %d bytes", len(msg.Data))

	})

	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		log.Printf("[DATA CHANNEL] New channel: %s", dc.Label())
	})
	// handles ice connection state change
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[ICE] Connection state change to: %s", state.String())

		switch state {
		case webrtc.ICEConnectionStateConnected:
			log.Println("[ICE] Connected! Closing signaling server connection...")

			// Send disconnect message
			disconnectMsg := Message{
				Type:   "disconnect",
				RoomID: roomID,
				Sender: userID,
				Target: remoteID, // or leave blank depending on your logic
			}
			conn.WriteJSON(disconnectMsg)

			conn.Close()
		case webrtc.ICEConnectionStateFailed:
			log.Println("[ICE] connection failed. Restarting ICE..")
			peerConnection.CreateOffer(&webrtc.OfferOptions{ICERestart: true})
		}
	})
	return peerConnection, nil
}

// manages signaling messages
func handleMessage(msg Message) {
	if msg.Sender == userID {
		return
	}

	peerConnection, exists := peerManager.Get(msg.Sender)
	if !exists {
		log.Println("[WEBRTC] Creating new conection for:", msg.Sender)
		NewPeerConnection, err := createPeerConnection(msg.Sender)
		if err != nil {
			log.Println("Error creating peer connection", err)
			return
		}
		peerManager.Add(msg.Sender, NewPeerConnection)
		peerConnection = NewPeerConnection
	}

	switch msg.Type {
	case "offer":
		log.Println("[SIGNAL] Received offer from:", msg.Sender)
		if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  msg.Content,
		}); err != nil {
			log.Println("Error in setting up remote description:", err)
			return
		}

		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			log.Println("Error creating answer:", err)
			return
		}
		if err := peerConnection.SetLocalDescription(answer); err != nil {
			log.Println("Error setting local description:", err)
			return
		}

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
		if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  msg.Content,
		}); err != nil {
			log.Println("Error setting remote description:", err)
		}

	case "ice-candidate":
		var candidate webrtc.ICECandidateInit
		if err := json.Unmarshal([]byte(msg.Content), &candidate); err != nil {
			log.Println("Error unmarshalling ICE CANDIDATE:", err)
			return
		}
		if err := peerConnection.AddICECandidate(candidate); err != nil {
			log.Println("Error adding ICE CANDIDATE:", err)
		}

	}
}

func main() {
	// Load environment variables from .env file.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default values")
	}
	apiKey := os.Getenv("API_KEYS")

	if apiKey == "" {
		log.Println("No Available API KEY Check .env File")
	}
	roomStr := os.Getenv("ROOM_ID")
	userStr := os.Getenv("USER_ID")

	if roomStr == "" || userStr == "" {
		log.Fatal("ROOM_ID and USER_ID must be set")
	}

	roomID64, err := strconv.ParseUint(roomStr, 10, 32)
	if err != nil {
		log.Println("Invalid ROOM_ID, using default 1001")
		roomID = 1001
	} else {
		roomID = uint32(roomID64)
	}

	userIDParsed, err := strconv.ParseUint(userStr, 10, 64)
	if err != nil {
		userID = 92633
	} else {
		userID = userIDParsed
	}

	// read configuration from environment variables.
	port := os.Getenv("WS_PORT")
	if port == "" {
		port = "8080"
	}
	serverURL := fmt.Sprintf("ws://localhost:%s/ws", port)

	// websocket create room
	conn, err := connectToWebsocket(serverURL, apiKey, userID)
	if err != nil {
		log.Fatal("failed to connect", err)
	}
	defer conn.Close()

	joinMsg := Message{
		Type:   "join",
		RoomID: roomID,
		Sender: userID,
	}
	if err := conn.WriteJSON(joinMsg); err != nil {
		log.Println("[ERROR] Failed to send join message:", err)
	}

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
			msg := Message{
				Type:    "text",
				RoomID:  roomID,
				Sender:  userID,
				Content: "Connected to the signaling server",
			}
			err := conn.WriteJSON(msg)
			if err != nil {
				log.Println("[ERROR] Failed to send message:", err)
			}
			fmt.Println("[CLIENT] Sent:", msg)
		case <-interrupt:
			fmt.Println("[SYSTEM] Received interrupt, closing connection...")
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Goodbye!"))
			return
		}
	}

}
