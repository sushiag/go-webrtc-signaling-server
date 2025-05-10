package websocket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

// pre-defined constants for all the signaling messages types used in client-server communication
const (
	MessageTypeCreateRoom   = "create-room" // 	for function Create () to create rooms
	MessageTypeRoomCreated  = "room-created"
	MessageTypeJoinRoom     = "join-room" // for function Join() to join rooms
	MessageTypeRoomJoined   = "room-joined"
	MessageTypeOffer        = "offer"
	MessageTypeAnswer       = "answer"
	MessageTypeICECandidate = "ice-candidate"
	MessageTypeDisconnect   = "disconnect" // for function DisconnectHandle to disconnect from room
	MessageTypePeerJoined   = "peer-joined"
	MessageTypePeerListReq  = "peer-list-request"
	MessageTypePeerList     = "peer-list"
	MessageTypeStartSession = "start-session" // for func StartSession to start p2p
	MessageTypeSendMessage  = "send-message"
	MessageTypeHostChanged  = "host-changed"
)

type Payload struct {
	DataType string `json:"data_type"`
	Data     []byte `json:"data"`
}

// defined struct 'Message' for websocket communication
type Message struct {
	Type      string   `json:"type"`                // type of message
	Content   string   `json:"content,omitempty"`   // content
	RoomID    uint64   `json:"roomid,omitempty"`    // room id
	Sender    uint64   `json:"from,omitempty"`      // sender user id
	Target    uint64   `json:"to,omitempty"`        // target user id
	Candidate string   `json:"candidate,omitempty"` // ice-candiate string
	SDP       string   `json:"sdp,omitempty"`       // session description
	Users     []uint64 `json:"users,omitempty"`     // list of user ids
	Text      string   `json:"text,omitempty"`      // for send messages
	Payload   Payload  `json:"Payload,omitempty"`   // for send messages
}

// defined struct client instance with connection and state data
type Client struct {
	Conn       *websocket.Conn // websocket connection
	ServerURL  string          //server address
	ApiKey     string          // api key for the auth
	SessionKey string          // session key token to be received by the server
	UserID     uint64          // unique id assigned to the client
	RoomID     uint64          // current room joined, assigned to client, user
	onMessage  func(Message)   // callback function for handling messagess
	doneCh     chan struct{}   // chanel for the signal connection closing
	isClosed   bool            // closing when os exit
	SendMutex  sync.Mutex      // concurrent writting to the websocket. safe thread
	closeOnce  sync.Once       // close websocket connection
}

// to load the .env file once the package/module initializes
func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("[CLIENT SIGNALING] Warning: No .env file found or failed to load it")
	}
}

// this creates and returns a new client default config
func NewClient(serverUrl string) *Client {
	return &Client{
		ServerURL: serverUrl,
		ApiKey:    os.Getenv("API_KEY"), // gets api key from .env
		doneCh:    make(chan struct{}),  // initializes donechl
	}
}

// for testing purposes
func (c *Client) SetServerURL(url string) {
	c.ServerURL = url
}

// for testing purposes
func (c *Client) SetApiKey(key string) {
	c.ApiKey = key
}

// performs initial http authentication with api key to get session key and userid
func (c *Client) PreAuthenticate() error {
	payload := map[string]string{"apikey": c.ApiKey} // prepares the request body
	body, _ := json.Marshal(payload)                 // encodes to json

	authUrl := fmt.Sprintf("http://%s/auth", c.ServerURL)
	resp, err := http.Post(authUrl, "application/json", bytes.NewBuffer(body)) // sends request
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[CLIENT SIGNALING] auth failed: %s", resp.Status)
	}
	// decodes the response unto user id
	var result struct {
		UserID uint64 `json:"userid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	c.UserID = result.UserID
	return nil
}

// initializes the websocket connection and starts listening for message
func (c *Client) Init() error {
	headers := http.Header{}
	headers.Set("X-Api-Key", c.ApiKey) // set the auth header

	wsEndpoint := fmt.Sprintf("ws://%s/ws", c.ServerURL)
	conn, _, err := websocket.DefaultDialer.Dial(wsEndpoint, headers)
	if err != nil {
		return fmt.Errorf("[CLIENT SIGNALING] websocket connection failed: %v", err)
	}

	log.Println("[CLIENT SIGNALING] Connected to:", c.ServerURL)
	c.Conn = conn
	go c.listen()
	return nil
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.doneCh)
		c.isClosed = true

		if c.Conn != nil {
			c.SendMutex.Lock()
			_ = c.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			_ = c.Conn.Close()
			c.SendMutex.Unlock()
		}
		log.Println("[CLIENT SIGNALING] Connection closed")
	})
}

func (c *Client) Send(msg Message) error {
	if c.isClosed {
		return fmt.Errorf("[CLIENT SIGNALING] Cannot send message, connection is closed")
	}

	c.SendMutex.Lock()
	defer c.SendMutex.Unlock()

	if msg.RoomID == 0 {
		msg.RoomID = c.RoomID
	}
	if msg.Sender == 0 {
		msg.Sender = c.UserID
	}
	if err := c.Conn.WriteJSON(msg); err != nil {
		log.Printf("[CLIENT SIGNALING] Failed to send '%s': %v", msg.Type, err)
		return err
	}
	return nil
}

func (c *Client) JoinRoom(roomID string) error {
	roomIDUint64, err := strconv.ParseUint(roomID, 10, 64)
	if err != nil {
		return fmt.Errorf("[CLIENT SIGNALING] invalid room ID: %v", err)
	}
	c.RoomID = roomIDUint64

	err = c.Send(Message{
		Type:   MessageTypeJoinRoom,
		RoomID: c.RoomID,
	})
	if err != nil {
		log.Printf("[CLIENT] Failed to join room: %v", err)
		return fmt.Errorf("[CLIENT SIGNALING] could not join room %d: %v", c.RoomID, err)
	}

	log.Printf("[CLIENT] Join request sent for room: %d", c.RoomID)
	return nil
}

func (c *Client) Create() error {
	err := c.Send(Message{
		Type: MessageTypeCreateRoom,
	})
	if err != nil {
		log.Printf("[CLIENT SIGNALING] Failed to create room: %v", err)
		return err
	}
	log.Println("[CLIENT SIGNALING] Room creation request sent.")
	return nil
}

func (c *Client) SetMessageHandler(handler func(Message)) {
	c.onMessage = handler
}

func (c *Client) StartSession() error { // go to peer to peer
	return c.Send(Message{
		Type: MessageTypeStartSession,
	})
}

func (c *Client) listen() {
	for {
		select {
		case <-c.doneCh:
			return
		default:
			_, data, err := c.Conn.ReadMessage()
			if err != nil {
				if closeErr, ok := err.(*websocket.CloseError); ok {
					log.Printf("[CLIENT SIGNALING] Failed to join room: %s", closeErr.Text)
				} else {
					log.Println("[CLIENT SIGNALING] Read error:", err)
				}
				c.Close()
				return
			}

			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Println("[CLIENT SIGNALING] Unmarshal error:", err)
				continue
			}

			switch msg.Type {
			case MessageTypePeerList:
				log.Printf("[CLIENT SIGNALING] Peer list received for room %d", c.RoomID)
			case MessageTypeRoomCreated:
				fmt.Printf("[CLIENT SIGNALING] \n Room created: %d\n Copy this Room ID and share it with a friend!\n\n", msg.RoomID)
				c.RoomID = msg.RoomID
				c.RequestPeerList()
				log.Printf("[CLIENT SIGNALING] userID: %d | roomID: %d", c.UserID, c.RoomID)

			case MessageTypeRoomJoined:
				log.Printf("[CLIENT SIGNALING] Successfully joined room: %d", msg.RoomID)
				c.RoomID = msg.RoomID
				c.RequestPeerList() // request the peer list to connect to others in the room
				log.Printf("[CLIENT SIGNALING] userID: %d | roomID: %d", c.UserID, c.RoomID)

			case MessageTypePeerJoined:
				log.Printf("[CLIENT SIGNALING] Peer joined: %d", msg.Sender)

			case MessageTypeStartSession:
				log.Printf("[CLIENT SIGNALING] Received start-session. Telling PeerManager to initiate P2P…")
				if c.onMessage != nil {
					c.onMessage(msg)
				}
				c.LeaveServer()
				return

			case MessageTypeHostChanged:
				log.Printf("[CLIENT SIGNALING] New host assigned: %d", msg.Sender)
				if c.UserID == msg.Sender {
					log.Println("[CLIENT SIGNALING] You are now the host.")
				}
			case MessageTypeSendMessage:
				if msg.Text == "" && msg.Payload.Data == nil {
					log.Printf("Empty message received from %d, ignoring", msg.Sender)
					break
				}

				if msg.Text != "" {
					log.Printf("[CLIENT SIGNALINGCLIENT SIGNALING] Received text message from %d to %d: %s", msg.Sender, msg.Target, msg.Text)
				}

				if msg.Payload.Data != nil {
					log.Printf("[CLIENT SIGNALING] Received %s data from %d", msg.Payload.DataType, msg.Sender)
					switch msg.Payload.DataType {
					case "audio":
						log.Printf("Received audio data, size: %d bytes", len(msg.Payload.Data))
					case "video":
						log.Printf("Received video data, size: %d bytes", len(msg.Payload.Data))
					case "file":
						log.Printf("Received file data, size: %d bytes", len(msg.Payload.Data))
					default:
						log.Printf("Received arbitrary data of type: %s, size: %d bytes", msg.Payload.DataType, len(msg.Payload.Data))
					}
				}
				data := []byte(msg.Text)
				if err := c.SendDataToPeer(msg.Target, data); err != nil {
					log.Printf("Failed to send message to %d: %v", msg.Target, err)
				}

			case MessageTypeDisconnect:
				log.Printf("[CLIENT SIGNALING] Disconnected by server: %s", msg.Content)
				c.Close()
				os.Exit(1)
			}

			if c.onMessage != nil {
				c.onMessage(msg)
			}
		}
	}
}

func (c *Client) SendDataToPeer(targetID uint64, data []byte) error {
	return c.Send(Message{
		Type:    MessageTypeSendMessage,
		Target:  targetID,
		RoomID:  c.RoomID,
		Sender:  c.UserID,
		Payload: Payload{DataType: "binary", Data: data},
	})
}

func (c *Client) RequestPeerList() {
	err := c.Send(Message{
		Type: MessageTypePeerListReq,
	})
	if err != nil {
		log.Printf("[CLIENT SIGNALING] Failed to request peer list: %v", err)
		return
	}
	log.Println("[CLIENT SIGNALING] Requested peer list")
}

func (c *Client) IsWebSocketClosed() bool {
	if c.Conn == nil {
		return true
	}
	return c.Conn.CloseHandler() != nil
}

// sends a signaling message to a specific target user
func (c *Client) SendSignalingMessage(targetID uint64, msgType string, sdpOrCandidate string) error {
	msg := Message{
		Type:   msgType,
		Target: targetID,
		RoomID: c.RoomID,
		Sender: c.UserID,
	}

	switch msgType {
	case MessageTypeOffer, MessageTypeAnswer:
		msg.SDP = sdpOrCandidate
	case MessageTypeICECandidate:
		msg.Candidate = sdpOrCandidate
	case MessageTypeSendMessage:
		msg.Content = sdpOrCandidate // repurpose to send custom data

	default:
		return fmt.Errorf("unsupported signaling message type: %s", msgType)
	}

	return c.Send(msg)
}

func (c *Client) LeaveServer() {
	log.Println("[CLIENT SIGNALING] Leaving signaling server and switching to P2P")
	c.Close()
}
