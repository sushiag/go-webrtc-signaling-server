package main

import (
	"log"
	"net/http"
	"server/internal/websocket"
)

var validApiKeys map[string]bool

func main() {
	// Load valid API keys
	var err error
	validApiKeys, err = websocket.LoadValidApiKeys() // Call the function to load the keys as a map
	if err != nil {
		log.Fatalf("Error loading API keys: %v", err)
	}

	// Print the loaded API keys (optional)
	// for testing only, commenting outfmt.Println("Loaded API Keys:", validApiKeys)

	// Create a new WebSocketManager and set the valid API keys
	wsManager := websocket.NewWebSocketManager()
	wsManager.SetValidApiKeys(validApiKeys) // Set valid API keys in WebSocketManager

	// Register the WebSocket handler
	http.HandleFunc("/ws", wsManager.Handler)

	// Start the HTTP server
	log.Println("[SERVER] WebSocket server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("[SERVER] Failed to start server: %v", err)
	}
}
