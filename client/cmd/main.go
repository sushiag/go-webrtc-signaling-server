package main

import (
	clienthandle "client/clienthandler"
	"client/webrtchandler"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	room := flag.String("room", "", "Room ID (optional)")
	startP2P := flag.Bool("start", false, "Send 'start' signal to server to disconnect for P2P")

	flag.Parse()

	client := clienthandle.NewClient()

	if err := client.PreAuthenticate(); err != nil {
		log.Fatal("[CLIENT] Authentication Failed:", err)
	}

	if err := client.Init(); err != nil {
		log.Fatal("[CLIENT] Init failed:", err)
	}
	defer client.Close()

	peerManager := webrtchandler.NewPeerManager()
	client.SetMessageHandler(func(msg clienthandle.Message) {
		peerManager.HandleSignalingMessage(msg, client)
	})

	if *room != "" {
		log.Printf("[CLIENT] You're attempting to join room: %s", *room)
		if err := client.Join(*room); err != nil {
			log.Fatal("[CLIENT] Join failed:", err)
		}
	} else {
		log.Println("[CLIENT] No room provided. Creating new room...")
		if err := client.Start(); err != nil {
			log.Fatal("[CLIENT] Create room failed:", err)
		}
	}

	if *startP2P {
		startMsg := clienthandle.Message{
			Type:   "start",
			RoomID: client.RoomID,
			Sender: client.UserID,
		}
		if err := client.Send(startMsg); err != nil {
			log.Println("[CLIENT] Failed to send 'start':", err)
		} else {
			log.Println("[CLIENT] Sent 'start' command to server to go P2P")
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
