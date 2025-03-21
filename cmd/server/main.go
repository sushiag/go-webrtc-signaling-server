package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/sushiag/go-webrtc-signaling-server/internal/websocket"
)

var log = logrus.New()

func init() {
	// load the environment variables
	if err := godotenv.Load(); err != nil {
		log.Warn("Error loading .env file, using defaults")
	}

	log.SetFormatter(&logrus.JSONFormatter{}) // structured logging
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel) // adjust as needed
}

// generateHMACToken creates an HMAC-based token
func generateHMACToken(clientID string) string {
	hmacKey := []byte(os.Getenv("HMAC_SECRET"))
	if len(hmacKey) == 0 {
		log.Fatal("HMAC_SECRET is not set environmental vairables")
	}
	hash := hmac.New(sha256.New, hmacKey)
	hash.Write([]byte(clientID)) // use clientID instead of fixed data
	return hex.EncodeToString(hash.Sum(nil))
}

// authenticate checks API key authentication
func authenticate(r *http.Request) bool {
	sentToken := r.Header.Get("X-Auth-Token") // Get token from headers
	clientID := r.Header.Get("X-Client-ID")   //client unique identifiier
	if clientID == "" {
		log.Warn("Missing client ID")
		return false
	}

	expectedToken := generateHMACToken(clientID) // Generate expected token

	log.WithFields(logrus.Fields{
		"expected_token": expectedToken,
		"sent_token":     sentToken,
	}).Warn("Authentication attempt")

	if sentToken != expectedToken {
		log.Warn("Unauthorized access attempt!")
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

func main() {
	serverPort := ":" + os.Getenv("SERVER_PORT")
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")

	wsManager := websocket.NewWebSocketManager(allowedOrigin)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth", authHandler) // Add this
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsManager.Handler(w, r, authenticate)
	})

	log.WithField("port", serverPort).Info("Server is running. Press CTRL+C to exit.")

	if err := http.ListenAndServe(serverPort, mux); err != nil {
		log.WithError(err).Fatal("Server failed to start")
	}
}
