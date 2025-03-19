package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/pion/webrtc/v3"
)

type Client struct {
	ID   string
	Conn *websocket.Conn
	Room string
}

type Room struct {
	Clients map[string]*Client
	Mutex   sync.Mutex
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // r.Host == "insert-your-domain-here.com"
		},
	}
	rooms   = make(map[string]*Room)
	Mtx     sync.Mutex
	hmacKey = []byte(os.Getenv("HMAC_Secret")) // hash-based message authentication code, to encrypt a secret key
	clients = make(map[string]*websocket.Conn) // declared as global variable to track the connected clients
)

// this is the auth middleware
func authenticate(r *http.Request) bool {
	sentToken := r.Header.Get("X-Auth-Token")
	expectedToken := generateHMACToken()
	return sentToken == expectedToken
}

func disconnectClient(client *Client) {
	if client.Room != "" {
		Mtx.Lock()
		room, exists := rooms[client.Room]
		if exists {
			room.Mutex.Lock()
			delete(room.Clients, client.ID)
			room.Mutex.Unlock()
		}
		log.Printf("Client disconnected:  %s", client.ID)
	}

}

func readMessages(client *Client) {
	client.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(40 * time.Second))
		return nil
	})
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			log.Println("Error in Reading Message:", err)
			break
		}
		fmt.Printf("Received: from %s: %s\n", client.ID, string(message))
		if client.Room != "" {
			broadcastMessage(client.Room, client.ID, message)
		}
	}
}

func generateHMACToken() string {
	hash := hmac.New(sha256.New, hmacKey)
	hash.Write([]byte("fixed-data"))
	return hex.EncodeToString(hash.Sum(nil))
}

func initializePeerConnection() (*webrtc.PeerConnection, error) {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.1.google.com:19302"}}, // stun server, helps devices behind NATs discover their IP address (public)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create a peer connection: %v", err)
	}
	return peerConnection, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	if !authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return //authenticates the request before upgrading to the websocket
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("failed to upgrade to websocket connection", err)
		return
	}
	defer conn.Close()

	clientID := r.RemoteAddr // using Client ID
	client := &Client{ID: clientID, Conn: conn}
	Mtx.Lock()
	clients[clientID] = conn
	Mtx.Unlock()

	log.Println("A new client has connected: ", clientID)
	defer disconnectClient(client)

	peerConnection, err := initializePeerConnection()
	if err != nil {
		log.Println("Failed to initialize WebRTC connection:", err)
		return
	}

	go func() {
		for {
			time.Sleep(10 * time.Second) // receive pong
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("Ping failed, closing connection:", err)
				client.Conn.Close()
				break
			}
		}
	}()
	readMessages(client)
	defer peerConnection.Close()
}

func broadcastMessage(roomID, senderID string, message []byte) {
	Mtx.Lock()
	room, exist := rooms[roomID]
	Mtx.Unlock()

	if !exist {
		log.Println("Room does not exist:", roomID)
		return
	}

	room.Mutex.Lock()
	defer room.Mutex.Unlock()
	for id, client := range room.Clients {
		if id != senderID { // to no send the message back to senderID
			client.Conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error on loading your .env file:", err)
	}
	serverPort := ":" + os.Getenv("SERVER_PORT")
	serverHost := os.Getenv("SERVER_HOST")
	certFile := os.Getenv("CERT_FILE")
	keyFile := os.Getenv("KEY_FILE")
	// fixed .env variables

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler)

	server := &http.Server{
		Addr:         serverPort, // port address in const
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	log.Printf("running securely on server: %s%s press CTRL+C to exit", serverHost, serverPort)
	err = server.ListenAndServeTLS(certFile, keyFile)
	if err != nil {
		log.Fatal(server.ListenAndServeTLS("cert.pem", "key.pem"))
	}
}
