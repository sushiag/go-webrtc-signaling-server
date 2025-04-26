package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func loadAPIKey() string {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found.")
	}
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatal("API_KEY not set in .env")
	}
	return apiKey
}

func getServerURL() string {
	port := os.Getenv("WS_PORT")
	if port == "" {
		port = "8080"
	}
	return fmt.Sprintf("ws://localhost:%s/ws", port)
}
