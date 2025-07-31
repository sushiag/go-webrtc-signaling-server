package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/sushiag/go-webrtc-signaling-server/server/server/db"
)

// This handles the setup of the HTTP signaling server
func StartServer(port string, queries *db.Queries) (*http.Server, string) {
	host := os.Getenv("SERVER_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	log.Printf("[SERVER] Using host: %s", host)

	// This builds url from host:port
	serverUrl := fmt.Sprintf("%s:%s", host, port)
	log.Printf("[SERVER] Binding to %s", serverUrl)

	// Create the TCP listener
	listener, err := net.Listen("tcp", serverUrl)
	if err != nil {
		log.Fatalf("[SERVER] Error starting server: %v", err)
	}
	// This gets the actual bound address (if "0" it will automatically choose an available one)
	serverUrl = listener.Addr().String()
	log.Printf("[SERVER] Listening on %s", serverUrl)

	// This creayes a new WebsocketManger to manager all the active websocket from the signaling client
	wsManager := NewWebSocketManager()

	mux := http.NewServeMux()

	// This set the HTTP handlers
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		registerNewUser(w, r, queries)
	})
	mux.HandleFunc("/newpassword", func(w http.ResponseWriter, r *http.Request) {
		updatePassword(w, r, queries)
	})
	mux.HandleFunc("/regenerate", func(w http.ResponseWriter, r *http.Request) {
		regenerateNewAPIKeys(w, r, queries)
	})

	// WebSocket Connection
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[SERVER] /ws called from %s", r.RemoteAddr)
		handleWSEndpoint(w, r, wsManager.newConnChan, queries)
	})

	// This creates the HTTP server with the given handler
	server := &http.Server{
		Addr:    serverUrl,
		Handler: mux,
	}

	// This starts the HTTP server in a new goroutine
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
