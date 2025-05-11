package websocket

import (
	"log"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("[CLIENT SIGNALING] Warning: No .env file found or failed to load it")
	}
}
