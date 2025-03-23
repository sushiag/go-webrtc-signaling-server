package client

/*
#include <stdint.h>
*/
import "C"

import (
	"sync"
	"unsafe"

	"github.com/sushiag/go-webrtc-signaling-server/internal/webrtc"
)

var (
	clientInstance *webrtc.WebRTCClient
	callback       func(sourceID string, message []byte)
	once           sync.Once
)

//export InitWebRTCClient
func InitWebRTCClient(apiKey *C.char, signalingURL *C.char) *C.char {
	once.Do(func() {
		clientInstance = webrtc.NewWebRTCClient(C.GoString(apiKey), C.GoString(signalingURL))
	})
	err := clientInstance.Connect()
	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}

//export StartSession
func StartSession(sessionID *C.char) *C.char {
	if clientInstance == nil {
		return C.CString("Client not initialized")
	}
	err := clientInstance.StartSession(C.GoString(sessionID))
	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}

//export JoinSession
func JoinSession(sessionID *C.char) *C.char {
	if clientInstance == nil {
		return C.CString("Client not initialized")
	}
	err := clientInstance.JoinSession(C.GoString(sessionID))
	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}

//export SendSignalingMessage
func SendSignalingMessage(targetID *C.char, message *C.char, length C.int) *C.char {
	if clientInstance == nil {
		return C.CString("Client not initialized")
	}
	data := C.GoBytes(unsafe.Pointer(message), length)
	err := clientInstance.SendMessage(C.GoString(targetID), data)
	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}

//export SetMessageHandler
func SetMessageHandler(cb unsafe.Pointer) {
	clientInstance.SetMessageHandler(func(sourceID string, message []byte) {
		if cb != nil {
			fn := *(*func(sourceID string, message []byte))(cb)
			fn(sourceID, message)
		}
	})
}

//export CloseSession
func CloseSession() *C.char {
	if clientInstance == nil {
		return C.CString("Client not initialized")
	}
	err := clientInstance.Close()
	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}
