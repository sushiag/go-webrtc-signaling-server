// binding.go
package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"fmt"

	"github.com/sushiag/go-webrtc-signaling-server/internal/webrtc"
)

//export InitWebRTC
func InitWebRTC() {
	fmt.Println("InitWebRTC called from Rust")
	webrtc.Init() // call into rtc.go
}

//export CreateOffer
func CreateOffer() *C.char {
	offer := webrtc.CreateOffer()
	return C.CString(offer)
}

//export SetRemoteAnswer
func SetRemoteAnswer(answer *C.char) {
	goStr := C.GoString(answer)
	webrtc.SetRemoteAnswer(goStr)
}

//export AddIceCandidate
func AddIceCandidate(candidate *C.char) {
	goStr := C.GoString(candidate)
	webrtc.AddIceCandidate(goStr)
}
