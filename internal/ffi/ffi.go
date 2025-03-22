package ffi

/*
#cgo CFLAGS: -I./
#cgo LDFLAGS: -L./ -lwebrtc
#include <stdio.h>
void ReceiveMessageFromGo(const char* msg);
*/
import "C"

import (
	"encoding/json"
	"log"
	"unsafe"

	"github.com/pion/webrtc/v4"
	"github.com/sushiag/go-webrtc-signaling-server/internal/webrtc"
)

var peerConnection *webrtc.PeerConnection

func StartWebRTC() {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.1.google.com:19302"}},
		},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Println("Failed to create PeerConnection:", err)
		return
	}

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			candidate := c.ToJSON()
			candidateJSON, _ := json.Marshal(candidate)

			sendToRust(string(candidateJSON))
		}
	})

	peerConnection = pc
}

func sendToRust(msg string) {
	cMsg := C.CString(msg)
	defer C.free(unsafe.Pointer(cMsg))

	C.ReceiveMessageFromGo(cMsg)
}
