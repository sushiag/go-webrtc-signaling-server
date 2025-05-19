package main

import (
	"log"

	"github.com/sushiag/go-webrtc-signaling-server/server/lib/server"
)

func main() {
	server, wsURL := server.StartServer("")

	log.Printf("[SERVER] WebSocket server started at %s", wsURL)

	defer func() {
		if err := server.Close(); err != nil {
			log.Fatalf("[SERVER] Failed to stop the server: %v", err)
		}
	}()

	select {} // to keep running
}
