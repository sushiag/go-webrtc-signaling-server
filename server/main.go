package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"server/room"
	"server/websocket"
)

var (
	log         = logrus.New()
	apiKeys     = make(map[string]struct{}) // clientID -> apiKey
	roomManager = room.NewRoomManager()
)

func generateAPIKey(seed string) string { // to genrate unique api key using HMAC
	secret := os.Getenv("HMAC_SECRET")
	if secret == "" {
		log.Fatal("HMAC_SECRET not set in evironment variables")
	}
	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write([]byte(seed))
	return hex.EncodeToString(hash.Sum(nil))
}

// Check if Client ID and API Key is valid
func authenticate(r *http.Request) bool {
	apiKey := r.Header.Get("X-API-Key")

	_, exists := apiKeys[apiKey]

	if !exists {
		// maybe log the ip of the requester here...
		log.Printf("[Auth] invalid API key")
		return false
	}

	return true
}

func main() {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	serverPort := ":" + port

	wsManager := websocket.NewWebSocketManager()

	mux := http.NewServeMux()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWs(wsManager, w, r)
	})

	log.WithField("port", serverPort).Info("Server is running. Press CTRL+C to exit.")

	// a test token for your client ID using your .env secret
	seed := "some-random-string"
	api_key := generateAPIKey(seed)
	fmt.Println("-----------")
	fmt.Println("Test Token Generator")
	fmt.Println("X-API-Key:", api_key)
	fmt.Println("-----------")
	// Insert the api key here
	apiKeys[api_key] = struct{}{}

	// TODO:
	// instead of generating a random api key, read a keys.txt file containing the API keys.
	// example txt file:
	// ```
	// some-random-string-1
	// some-random-string-2
	// ```
	// One line, one api-key

	// Block forever
	log.Fatal(http.ListenAndServe(serverPort, mux))
}

// TODO: make this cleaner
func handleWs(wsManager *websocket.WebSocketManager, w http.ResponseWriter, r *http.Request) {
	// Authenticate
	if !authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade to WS
	conn, err := wsManager.Upgrade(w, r)
	if err != nil {
		log.Println("[WS] WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close() // ??

	// Get client ID then insert it to the clients map
	clientId := r.URL.Query().Get("client_id")
	wsManager.AddClient(clientId, conn)

	// TODO: move client into a room

	log.Println("[WS] Client connected:", clientId)

	defer wsManager.DisconnectClient(clientId)

	go wsManager.SendPings(clientId)
	wsManager.ReadMessages(clientId)
}
