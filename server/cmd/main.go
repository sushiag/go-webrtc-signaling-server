package main

import (
	"database/sql"
	"io/ioutil"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sushiag/go-webrtc-signaling-server/server/lib/db"
	"github.com/sushiag/go-webrtc-signaling-server/server/lib/server"
)

func applySchema(conn *sql.DB, path string) error {
	schemaSQL, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = conn.Exec(string(schemaSQL))
	return err
}

func main() {
	conn, err := sql.Open("sqlite3", "file:users.db?cache=shared")

	if err != nil {
		log.Fatalf("[MAIN] Failed to connect to database: %v", err)
	}

	if err := applySchema(conn, "../lib/db/schema.sql"); err != nil {
		log.Fatalf("[MAIN] Failed to apply schema: %v", err)
	}

	queries := db.New(conn)

	httpServer, wsURL := server.StartServer("58526", queries)
	log.Printf("[SERVER] WebSocket server started at %s", wsURL)

	defer func() {
		if err := httpServer.Close(); err != nil {
			log.Fatalf("[SERVER] Failed to stop the server: %v", err)
		}
	}()

	select {}
}
