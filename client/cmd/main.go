package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sushiag/go-webrtc-signaling-server/client/lib/webrtc"
	"github.com/sushiag/go-webrtc-signaling-server/client/lib/websocket"
)

func main() {
	room := flag.String("room", "", "Room ID (optional)")
	disconnectfromserver := flag.Bool("start", false, "Send 'start' signal to server to disconnect for P2P")

	flag.Parse()

	wsPort := os.Getenv("WS_PORT")
	if wsPort == "" {
		wsPort = "ws://localhost:8080/ws"
	}
	wsEndpoint := fmt.Sprintf("ws://localhost:%s/ws", wsPort)

	client := websocket.NewClient(wsEndpoint)

	if err := client.PreAuthenticate(); err != nil {
		log.Fatal("[CLIENT] Authentication Failed:", err)
	}

	if err := client.Init(); err != nil {
		log.Fatal("[CLIENT] Init failed:", err)
	}
	defer client.Close()

	peerManager := webrtc.NewPeerManager()
	client.SetMessageHandler(func(msg websocket.Message) {
		peerManager.HandleSignalingMessage(msg, client)
	})

	if *room != "" {
		log.Printf("[CLIENT] You're attempting to join room: %s", *room)
		if err := client.JoinRoom(*room); err != nil {
			log.Fatal("[CLIENT] Join failed:", err)
		}
	} else {
		log.Println("[CLIENT] No room provided. Creating new room...")
		if err := client.Create(); err != nil {
			log.Fatal("[CLIENT] Create room failed:", err)
		}
	}

	if *disconnectfromserver {
		startMsg := websocket.Message{
			Type:   "disconnect-from-server",
			RoomID: client.RoomID,
			Sender: client.UserID,
		}
		if err := client.Send(startMsg); err != nil {
			log.Println("[CLIENT] Failed to send 'start':", err)
		} else {
			log.Println("[CLIENT] Sent 'disconnect-from-server' command to server to go P2P")
		}
	}

	waitForInterrupt()
}

func waitForInterrupt() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("[APP] Shutdown signal received")
}
