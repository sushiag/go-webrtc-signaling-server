package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

// ffi = foreign functoin interface == a translation of your Go functions that other languages can call.
// So... you need your functions first!

func main() {
	// Define WebSocket server URL
	serverURL := "ws://localhost:8080/ws" // Adjust path if needed

	// Create a WebSocket dialer and connect
	headers := http.Header{}
	headers.Set("X-API-Key", "my-api-key")
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, headers)
	if err != nil {
		log.Fatal("Error connecting to WebSocket server:", err)
	}
	defer conn.Close()

	fmt.Println("Connected to WebSocket server:", serverURL)

	// TODO: handle whatever webrtc shenanigans

	// Handle system interrupts (CTRL+C)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Start a goroutine to read messages from the server
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				return
			}
			fmt.Println("Received from server:", string(message))
		}
	}()

	// Periodically send messages to the server
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			msg := "Hello from WebSocket client!"
			err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {

				log.Println("Write error:", err)
				return
			}
			fmt.Println("Sent:", msg)

		case <-interrupt:
			fmt.Println("Received interrupt, closing connection...")
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Goodbye!"))
			if err != nil {

				log.Println("Close error:", err)
			}
			return
		}
	}
}
