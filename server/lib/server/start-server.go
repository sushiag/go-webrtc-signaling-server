package server

import (
	"bufio"
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
	log.Printf("[SERVER] Loading API keys from: %s", apiKeyPath)

	manager := NewWebSocketManager()

	apiKeys, err := LoadValidApiKeys(apiKeyPath)
	if err != nil {
		log.Printf("[SERVER] Failed to load API keys: %v", err)
	} else {
		log.Printf("[SERVER] Loaded %d API keys", len(apiKeys))
	}
	manager.SetValidApiKeys(apiKeys)
	log.Printf("[SERVER] API keys set in manager")

	mux := http.NewServeMux()
	mux.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[SERVER] /auth called from %s", r.RemoteAddr)
		manager.AuthHandler(w, r)
	})
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

// LoadValidApiKeys loads API keys from a file
func LoadValidApiKeys(path string) (map[string]bool, error) {
	log.Printf("[SERVER] Opening API key file: %s", path)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	keys := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		key := scanner.Text()
		log.Printf("[SERVER] Loaded API key: %s", key)
		keys[key] = true
		count++
	}
	log.Printf("[SERVER] Total API keys loaded: %d", count)

	return keys, scanner.Err()
}
