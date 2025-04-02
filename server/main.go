package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"server/websocket"
	"sync"
)

var (
	log        = logrus.New()
	apiKeys    = make(map[string]string) // clientID -> apiKey
	apiKeysMtx sync.Mutex
)

func generateAPIKey(clientID string) string { // to genrate unique api key using HMAC
	secret := os.Getenv("HMAC_SECRET")
	if secret == "" {
		log.Fatal("HMAC_SECRET not set in evironment variables")
	}
	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write([]byte(clientID))
	return hex.EncodeToString(hash.Sum(nil))
}

func registerNewClient(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "client_id is required", http.StatusBadRequest)
		return
	}

	apiKeysMtx.Lock()
	apiKey := generateAPIKey(clientID)
	apiKeys[clientID] = apiKey
	apiKeysMtx.Unlock()

	log.Printf("[AUTH] New API key has been genrated for client: %s", clientID)

	w.Header().Set("Content-type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"client_id": clientID,
		"apikey":    apiKey,
	})
	w.WriteHeader(http.StatusOK)
}

// authenticate checks API key authentication
func authenticate(r *http.Request) bool {
	clientID := r.Header.Get("X-Client-ID") //client unique identifiier
	apiKey := r.Header.Get("X-Auth-Token")

	apiKeysMtx.Lock()
	defer apiKeysMtx.Unlock()

	expectedKey, exists := apiKeys[clientID]

	if !exists || expectedKey != apiKey {

		log.Printf("[Auth] failed API key auth for client %s", clientID)
		return false
	}

	return true
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	if !authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Authorized"}`))
}

func init() {
	// load the environment variables
	if err := godotenv.Load(); err != nil {
		log.Warn("Error loading .env file, using defaults")

	}

	log.SetFormatter(&logrus.JSONFormatter{}) // structured logging
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel) // adjust as needed
}

func main() {

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	serverPort := ":" + port

	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "ws://localhost:" + port
	}

	wsManager := websocket.NewWebSocketManager(allowedOrigin)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth", authHandler)           // optional: client-side auth check
	mux.HandleFunc("/register", registerNewClient) // client registration route
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Inject query params into headers if present
		clientID := r.URL.Query().Get("client_id")
		token := r.URL.Query().Get("token")
		if clientID != "" && token != "" {

			r.Header.Set("X-Client-ID", clientID)
			r.Header.Set("X-Auth-Token", token)
		}

		// Proceed with authentication and connection
		wsManager.Handler(w, r, authenticate)
	})

	log.WithField("port", serverPort).Info("Server is running. Press CTRL+C to exit.")

	// a test token for your client ID using your .env secret
	clientID := "my-client"
	token := generateAPIKey(clientID)
	fmt.Println("-----------")
	fmt.Println("Test Token Generator")
	fmt.Println("X-Client-ID:", clientID)

	fmt.Println("X-Auth-Token:", token)

	fmt.Println("-----------")

	// Block forever
	log.Fatal(http.ListenAndServe(serverPort, mux))

}
