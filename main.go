package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
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
	Mutex sync.Mutex
)

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

	go readMessages(client)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("Client disconnected", err)
			room := rooms[client.Room]
			room.Mutex.Lock()
			delete(room.Clients, clientID)
			room.Mutex.Unlock()
			// this securely removes the client
			break
		}
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
		fmt.Println("Received:", string(message))
	}
}

func main() {

	http.HandleFunc("/ws", handler)
	httpServer := &http.Server{
		Addr:         ":8080", // port address for https
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Println("running on localhost: 8080, press CTRL+C to exit")
	log.Fatal(httpServer.ListenAndServeTLS("cert.pem", "key.pem"))
}
