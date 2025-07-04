package server

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sync/atomic"
)

type authHandler struct {
	// NOTE: this map is only thread-safe if we don't change the keys after initialization!
	apiKeys    map[string]*atomic.Bool
	nextUserID uint64
}

func newAuthHandler() (authHandler, error) {
	authHandler := authHandler{
		nextUserID: 0,
	}

	apiKeyPath := os.Getenv("APIKEY_PATH")
	if apiKeyPath == "" {
		apiKeyPath = "apikeys.txt"
	}
	log.Printf("[SERVER] Loading API keys from: %s", apiKeyPath)

	file, err := os.Open(apiKeyPath)
	if err != nil {
		return authHandler, fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	authHandler.apiKeys = make(map[string]*atomic.Bool)
	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		key := scanner.Text()
		log.Printf("[SERVER] Loaded API key: %s", key)
		authHandler.apiKeys[key] = &atomic.Bool{}
		count++
	}
	log.Printf("[SERVER] Total API keys loaded: %d", count)

	return authHandler, scanner.Err()
}

func (handler *authHandler) authenticate(apiKey string) (uint64, error) {
	var userID uint64

	isUsed, exists := handler.apiKeys[apiKey]
	if !exists {
		return userID, fmt.Errorf("invalid api key")
	}

	if !isUsed.CompareAndSwap(false, true) {
		return userID, fmt.Errorf("api key already in use!")
	}

	userID = handler.nextUserID
	handler.nextUserID += 1

	return userID, nil
}

func (wsm *WebSocketManager) SafeWriteJSON(c *Connection, v Message) error {
	c.Outgoing <- v
	return nil
}
