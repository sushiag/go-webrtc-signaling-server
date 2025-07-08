package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/sushiag/go-webrtc-signaling-server/server/lib/db"
)

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

	manager := NewWebSocketManager()

	handler := &Handler{
		Queries: queries,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/register", handler.registerNewUser)
	mux.HandleFunc("/login", handler.loginUser)
	mux.HandleFunc("/newpassword", handler.updatePassword)
	mux.HandleFunc("/regenerate", handler.regenerateNewAPIKeys)
	//mux.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
	//	log.Printf("[SERVER] /auth called from %s", r.RemoteAddr)
	//	manager.AuthHandler(w, r)
	// })
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[SERVER] /ws called from %s", r.RemoteAddr)
		manager.Handler(w, r)
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
