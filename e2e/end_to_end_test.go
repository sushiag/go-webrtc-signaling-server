package e2e_test

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	clientwrapper "github.com/sushiag/go-webrtc-signaling-server/client/clientwrapper"
	"github.com/sushiag/go-webrtc-signaling-server/server"
)

func startServer() {
	go func() {
		log.Println("[SERVER] Starting signaling server on :8080")
		http.HandleFunc("/ws", server.HandleWebSocket)
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal("Server error:", err)
		}
	}()
	time.Sleep(1 * time.Second)
}

func runClientTest() {
	log.Println("== STARTING CLIENT TEST ==")

	os.Setenv("SERVER_URL", "ws://localhost:8080/ws")
	os.Setenv("API_KEY", "test-api-key")

	host := clientwrapper.NewClient()
	if err := host.Connect(); err != nil {
		log.Fatal("Host connection error:", err)
	}

	if err := host.CreateRoom(); err != nil {
		log.Fatal("Host failed to create room:", err)
	}
	log.Println("Host created room.")

	time.Sleep(1 * time.Second)
	roomID := host.Client.RoomID
	log.Println("Room ID:", roomID)

	peer := clientwrapper.NewClient()
	if err := peer.Connect(); err != nil {
		log.Fatal("Peer connection error:", err)
	}

	if err := peer.JoinRoom(roomID); err != nil {
		log.Fatal("Peer failed to join room:", err)
	}
	log.Println("Peer joined room.")

	time.Sleep(2 * time.Second)

	if err := host.StartSession(); err != nil {
		log.Fatal("Host failed to start session:", err)
	}
	log.Println("Session started.")

	time.Sleep(3 * time.Second)

	for peerID := range peer.PeerManager.Peers {
		log.Println("Peer -> Host message...")
		_ = peer.SendMessageToPeer(peerID, "Hello from peer!")
	}

	for peerID := range host.PeerManager.Peers {
		log.Println("Host -> Peer message...")
		_ = host.SendMessageToPeer(peerID, "Hello from host!")
	}

	time.Sleep(3 * time.Second)

	log.Println("Closing signaling server via host...")
	host.CloseServer()

	peer.Close()
	host.Close()

	log.Println("== TEST COMPLETE ==")
}

func main() {
	_ = godotenv.Load()

	startServer()
	runClientTest()
}
