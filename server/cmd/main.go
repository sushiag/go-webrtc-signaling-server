package main

import (
	"log"

	_ "github.com/mattn/go-sqlite3"
	server "github.com/sushiag/go-webrtc-signaling-server/server/server"
	sqlitedb "github.com/sushiag/go-webrtc-signaling-server/server/server/register"
)

func main() {
	const serverData = "Server.db"
	queries, dbConn := sqlitedb.NewDatabase(serverData)
	httpServer, wsURL := server.StartServer("0", queries)
	log.Printf("[SERVER] WebSocket server started at %s", wsURL)

	defer func() {
		if err := httpServer.Close(); err != nil {
			log.Fatalf("[SERVER] Failed to stop the server: %v", err)
		}
		if err := dbConn.Close(); err != nil {
			log.Fatalf("[SERVER] Failed to close the database: %v", err)
		}
	}()

	select {}
}
