package ffi

/*
#cgo CFLAGS: -I./
#cgo LDFLAGS: -L./ -lwebrtc
#include <stdio.h>
void StartWebRTC();
*/
import "C"

import "github.com/pion/webrtc/v4"

var peerConnection *webrtc.PeerConnection

//export StartWebRTC
func StartWebRTC() {
	config := webrtc.Configuration{ /* ICE servers config */ }
	pc, _ := webrtc.NewPeerConnection(config)
	peerConnection = pc
}
