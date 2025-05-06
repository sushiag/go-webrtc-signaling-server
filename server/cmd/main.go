package main

import (
	"log"

	"github.com/sushiag/go-webrtc-signaling-server/server/lib/server"
)

func main() {
	server, wsURL := server.StartServer(":8080") // Pass bind address here

	log.Printf("[SERVER] WebSocket server started at %s", wsURL)
	W
	defer func() {
		if err := server.Close(); err != nil {
			log.Fatalf("[SERVER] Failed to stop the server: %v", err)
		}
	}()
}
