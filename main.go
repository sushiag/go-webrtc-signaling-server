package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	serverPort = ":8080" // you can change the port to 443
	certFile   = "cert.pem"
	keyFile    = "key.pem"
	serverHost = "localhost" // you can change this to your domain
	authToken  = "your-secret-token"
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

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return r.Host == "insert-your-domain-here.com"
	},
}
var (
	rooms = make(map[string]*Room)
	Mtx   sync.Mutex
)

// this is the auth middleware
func authenticate(r *http.Request) bool {
	token := r.Header.Get("Sec-Websocket-Protocol")
	return token == "replace-with-your-actual-token"
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

	clientID := uuid.New().String()
	client := &Client{
		ID:   clientID,
		Conn: conn,
	}

	log.Println("A new client has connected: ", clientID)
	defer disconnectClient(client)
	readMessages(client)

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
	defer client.Conn.Close()
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			log.Println("Error in Reading Message:", err)
			break
		}
		fmt.Printf("Received: from %s: %s\n", client.ID, string(message))
	}
}

func main() {
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
	log.Println("running securely on server: press CTRL+C to exit")

	err := server.ListenAndServeTLS(certFile, keyFile)
	if err != nil {
		log.Fatal(server.ListenAndServeTLS("cert.pem", "key.pem"))
	}
}
