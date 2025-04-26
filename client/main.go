package main

import (
	"log"
	"os"
	"os/signal"
)

var peerManager = NewPeerManager()

func main() {
	apiKey := loadAPIKey()
	serverURL := getServerURL()

	client, err := connect(apiKey, serverURL)
	if err != nil {
		log.Fatal("Failed to connect to signaling server:", err)
	}

	joinSession(client)

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Println("[SYSTEM] Shutting down...")
	client.Conn.Close()
}
