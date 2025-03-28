package main

/*
#include <stdlib.h>

typedef void (*MessageHandler)(const char* sourceID, const char* message);

// Declaration only — implementation is in bridge.c
void CallMessageHandlerBridge(MessageHandler handler, const char* sourceID, const char* message);
*/
import "C"

import (
	"sync"
	"unsafe"

	"github.com/sushiag/go-webrtc-signaling-server/internal/webrtc"
	"github.com/sushiag/go-webrtc-signaling-server/internal/websocket"
)

var (
	clientInstance *webrtc.WebRTCClient
	clientOnce     sync.Once
	messageHandler C.MessageHandler
)

func callMessageHandler(sourceID, message string) {
	cSource := C.CString(sourceID)
	cMsg := C.CString(message)
	defer C.free(unsafe.Pointer(cSource))
	defer C.free(unsafe.Pointer(cMsg))

	C.CallMessageHandlerBridge(messageHandler, cSource, cMsg)
}

//export InitWebRTCClient
func InitWebRTCClient(apiKey, signalingURL, roomID, clientID *C.char) C.int {
	var connectErr error

	clientOnce.Do(func() {
		wm := websocket.NewWebSocketManager("")

		clientInstance = webrtc.NewWebRTCClient(
			C.GoString(apiKey),
			C.GoString(signalingURL),
			wm,
			C.GoString(roomID),
			C.GoString(clientID),
		)

		clientInstance.SetMessageHandler(func(sourceID string, message []byte) {
			if messageHandler != nil {
				callMessageHandler(sourceID, string(message))
			}
		})

		connectErr = clientInstance.Connect(true)
	})

	if clientInstance == nil || connectErr != nil {
		return -1
	}
	return 0
}

//export StartSession
func StartSession(sessionID *C.char) C.int {
	if clientInstance == nil {
		return -1
	}
	err := clientInstance.StartSession(C.GoString(sessionID))
	if err != nil {
		return -1
	}
	return 0
}

//export JoinSession
func JoinSession(sessionID *C.char) C.int {
	if clientInstance == nil {
		return -1
	}
	err := clientInstance.JoinSession(C.GoString(sessionID))
	if err != nil {
		return -1
	}
	return 0
}

//export SendSignalingMessage
func SendSignalingMessage(targetID, message *C.char) C.int {
	if clientInstance == nil {
		return -1
	}
	err := clientInstance.SendMessage(C.GoString(targetID), []byte(C.GoString(message)))
	if err != nil {
		return -1
	}
	return 0
}

//export SetMessageHandler
func SetMessageHandler(handler C.MessageHandler) {
	messageHandler = handler
}

//export CloseSession
func CloseSession() C.int {
	if clientInstance == nil {
		return 0
	}
	err := clientInstance.Close()
	if err != nil {
		return -1
	}
	return 0
}

func main() {}
