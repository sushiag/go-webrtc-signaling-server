module e2e_test

replace github.com/sushiag/go-webrtc-signaling-server/server => ../server

replace github.com/sushiag/go-webrtc-signaling-server/client => ../client

go 1.23.5

require github.com/sushiag/go-webrtc-signaling-server/client v0.0.0-20250426025455-8ad70f05814e // indirect
