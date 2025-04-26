module e2e_test

replace github.com/sushiag/go-webrtc-signaling-server/client => ../client

go 1.23.5

require github.com/sushiag/go-webrtc-signaling-server/client v1.1.0-20250426030520-c5f46dc5b624

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
)
