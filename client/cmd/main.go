package main

import (
	"log"
	"os"
	"os/signal"

	client "github.com/sushiag/go-webrtc-signaling-server/client"
)

func main() {
	os.Setenv("API_KEY", "valid-api-key-1")

	client, err := client.NewClient("ws://127.0.0.1:50299/ws")
	if err != nil {
		log.Panicf("failed to initialize client: %v", err)
	}

	roomID, err := client.CreateRoom()
	if err != nil {
		log.Fatalf("failed to create room: %v", err)
	}
	log.Printf("joined room %d", roomID)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}
