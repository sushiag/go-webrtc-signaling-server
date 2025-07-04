package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

func StartServer(port string) (*http.Server, string) {
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

	apiKeyPath := os.Getenv("APIKEY_PATH")
	if apiKeyPath == "" {
		apiKeyPath = "apikeys.txt"
	}

	wsManager := NewWebSocketManager()

	authHandler, err := newAuthHandler()
	if err != nil {
		// TODO: return an error instead of panicking
		log.Fatalf("[ERROR] failed to initialize auth handler: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[SERVER] /ws called from %s", r.RemoteAddr)
		handleWSEndpoint(w, r, &authHandler, wsManager.newConnChan)
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
