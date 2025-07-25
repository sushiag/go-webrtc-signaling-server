package main

import (
	"log"

	_ "github.com/mattn/go-sqlite3"
	server "github.com/sushiag/go-webrtc-signaling-server/server/lib/server"
	"github.com/sushiag/go-webrtc-signaling-server/server/lib/server/db"
)

func main() {

	httpServer, wsURL := server.StartServer("0", &db.Queries{})
	log.Printf("[SERVER] WebSocket server started at %s", wsURL)

	defer func() {
		if err := httpServer.Close(); err != nil {
			log.Fatalf("[SERVER] Failed to stop the server: %v", err)
		}
	}()

	select {}
}
