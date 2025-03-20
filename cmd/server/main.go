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
func generateHMACToken() string {
	hmacKey := []byte(os.Getenv("HMAC_SECRET"))
	hash := hmac.New(sha256.New, hmacKey)
	hash.Write([]byte("fixed-data"))
	return hex.EncodeToString(hash.Sum(nil))
}

// authenticate checks API key authentication
func authenticate(r *http.Request) bool {
	sentToken := r.Header.Get("X-Auth-Token")
	expectedToken := generateHMACToken()

	if sentToken != expectedToken {
		log.Warn("Unauthorized access attempt!")
		return false
	}
	return true
}

func main() {
	serverPort := ":" + os.Getenv("SERVER_PORT")
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")

	wsManager := websocket.NewWebSocketManager(allowedOrigin)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsManager.Handler(w, r, authenticate)
	})

	log.WithField("port", serverPort).Info("Server is running. Press CTRL+C to exit.")

	if err := http.ListenAndServe(serverPort, mux); err != nil {
		log.WithError(err).Fatal("Server failed to start")
	}
}
