package main

import (
	"log"
	"os"
	"os/signal"

	client "github.com/sushiag/go-webrtc-signaling-server/client/lib"
)

func main() {
	ctrl := client.NewClient("")
	defer ctrl.Close()
	if err := ctrl.CreateRoom(); err != nil {
		log.Fatalf("Failed to create room: %v", err)
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}
