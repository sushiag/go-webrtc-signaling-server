package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"

	wsserver "github.com/sushiag/go-webrtc-signaling-server/server/wsserver"
)

// LoadValidApiKeys loads API keys from a file
func LoadValidApiKeys(path string) (map[string]bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	keys := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		keys[scanner.Text()] = true
	}
	return keys, scanner.Err()
}

func main() {
	wsManager := wsserver.NewWebSocketManager()
	apiKeys, _ := LoadValidApiKeys("apikeys.txt")
	wsManager.SetValidApiKeys(apiKeys)

	http.HandleFunc("/auth", wsManager.AuthHandler)
	http.HandleFunc("/ws", wsManager.Handler)

	log.Println("[SERVER] WebSocket server listening on :8080")
	if err := http.ListenAndServe("127.0.0.1:8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
