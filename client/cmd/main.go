package main

import (
	"log"
	"os"
	"os/signal"

	client "github.com/sushiag/go-webrtc-signaling-server/client/lib"
)

func main() {
	os.Setenv("API_KEY", "valid-api-key-1")

	ctrl := client.NewClient("127.0.0.1:50299")

	if err := ctrl.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	if err := ctrl.CreateRoom(); err != nil {
		log.Fatalf("Failed to create Lobby: %v", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}
