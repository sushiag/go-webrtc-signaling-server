package server

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/sushiag/go-webrtc-signaling-server/server/lib/server/db"
)

func newDefaultDB() *db.Queries {
	conn, err := sql.Open("sqlite3", "file:users.db?cache=shared")
	if err != nil {
		log.Fatalf("[SERVER] Failed to open DB: %v", err)
	}
	if err := applySchema(conn, "../server/lib/server/db/schema.sql"); err != nil { // adjust path if using cmd/main.go use '../lib/server/db/schema.sql'
		log.Fatalf("[SERVER] Failed to apply schema: %v", err)
	}
	return db.New(conn)
}

func applySchema(conn *sql.DB, path string) error {
	schemaSQL, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = conn.Exec(string(schemaSQL))
	return err
}
func StartServer(port string, queries *db.Queries) (*http.Server, string) {
	host := os.Getenv("SERVER_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	log.Printf("[SERVER] Using host: %s", host)

	serverUrl := fmt.Sprintf("%s:%s", host, port)
	log.Printf("[SERVER] Binding to %s", serverUrl)

	listener, err := net.Listen("tcp", serverUrl)
	if err != nil {
		log.Fatalf("[SERVER] Error starting server: %v", err)
	}
	serverUrl = listener.Addr().String()
	log.Printf("[SERVER] Listening on %s", serverUrl)

	if queries == nil {
		queries = newDefaultDB()
	}

	wsManager := NewWebSocketManager()
	wsManager.Queries = queries

	handler := &Handler{
		Queries: queries,
	}
	mux := http.NewServeMux()

	// Auth handlers
	mux.HandleFunc("/register", handler.registerNewUser)
	mux.HandleFunc("/newpassword", handler.updatePassword)
	mux.HandleFunc("/regenerate", handler.regenerateNewAPIKeys)

	// WebSocket Connection
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[SERVER] /ws called from %s", r.RemoteAddr)
		handleWSEndpoint(w, r, wsManager.newConnChan, wsManager, handler)
	})

	server := &http.Server{
		Addr:    serverUrl,
		Handler: mux,
	}

	go func() {
		log.Printf("[SERVER] Starting HTTP server goroutine")
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("[SERVER] HTTP server error: %v", err)
		}
		log.Printf("[SERVER] HTTP server goroutine stopped")
	}()

	time.Sleep(100 * time.Millisecond)
	log.Printf("[SERVER] StartServer returning")

	return server, serverUrl
}
