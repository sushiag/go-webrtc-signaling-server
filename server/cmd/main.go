package main

import (
	"log"

	"github.com/sushiag/go-webrtc-signaling-server/lib/server/server"
)

func main() {
	server, wsURL := server.StartServer(":8080") // Pass bind address here

	log.Printf("[SERVER] WebSocket server started at %s", wsURL)

	defer func() {
		if err := server.Close(); err != nil {
			log.Fatalf("[SERVER] Failed to stop the server: %v", err)
		}
	}()
}
