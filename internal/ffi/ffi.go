package ffi

/*
#cgo CFLAGS: -I./
#cgo LDFLAGS: -L./ -lwebrtc
#include <stdio.h>
void StartWebRTC();
*/

import (
	"C"
	"fmt"
	"log"

	"github.com/pion/webrtc/v4"
)

var peerConnection *webrtc.PeerConnection

//export StartWebRTC
func StartWebRTC() {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Println("[FFI] Failed to create peer connection:", err)
		return
	}
	peerConnection = pc

	fmt.Println("[FFI] WebRTC PeerConnection started successfully")
}
