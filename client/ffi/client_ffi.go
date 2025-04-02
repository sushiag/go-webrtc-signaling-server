package main

/*
#include <stdlib.h>
#cgo LDFLAGS: -L. -lbridge
#include "bridge.h"
*/
import "C"

import (
	"log"
	"sync"
	"unsafe"

	"github.com/sushiag/go-webrtc-signaling-server/server/internal/webrtc"
	"github.com/sushiag/go-webrtc-signaling-server/server/internal/websocket"
)

var (
	clients        sync.Map // Map of clientID -> *WebRTCClient
	messageHandler C.MessageHandler
)

// get the WebRTC client instance by clientID
func getClient(clientID string) *webrtc.WebRTCClient {
	v, exists := clients.Load(clientID)
	if !exists {
		return nil
	}
	return v.(*webrtc.WebRTCClient)
}

//export InitWebRTCClient
func InitWebRTCClient(apiKey, signalingURL, roomID, clientID *C.char) C.int {
	id := C.GoString(clientID)
	wm := websocket.NewWebSocketManager("")

	client := webrtc.NewWebRTCClient(
		C.GoString(apiKey),
		C.GoString(signalingURL),
		wm,
		C.GoString(roomID),
		id,
	)

	// Set the Go function as the message handler
	client.SetMessageHandler(func(sourceID string, message []byte) {
		if messageHandler != nil {
			cSource := C.CString(sourceID)
			cMsg := C.CString(string(message))
			defer C.free(unsafe.Pointer(cSource))
			defer C.free(unsafe.Pointer(cMsg))

			// Call the C function, passing the Go callback
			C.CallMessageHandlerBridge(messageHandler, cSource, cMsg)
		}
	})

	// Connect and handle errors if the connection fails
	if err := client.Connect(true); err != nil {
		log.Println("[ERROR] Failed to connect:", err)
		return -1
	}

	clients.Store(id, client)
	return 0
}

//export StartSession
func StartSession(clientID, sessionID *C.char) C.int {
	client := getClient(C.GoString(clientID))
	if client == nil {
		return -1
	}
	if err := client.StartSession(C.GoString(sessionID)); err != nil {
		return -1
	}
	return 0
}

//export JoinSession
func JoinSession(clientID, sessionID *C.char) C.int {
	client := getClient(C.GoString(clientID))
	if client == nil {
		return -1
	}
	if err := client.JoinSession(C.GoString(sessionID)); err != nil {
		return -1
	}
	return 0
}

//export SendSignalingMessage
func SendSignalingMessage(clientID, targetID, message *C.char) C.int {
	client := getClient(C.GoString(clientID))
	if client == nil {
		return -1
	}
	if err := client.SendMessage(C.GoString(targetID), []byte(C.GoString(message))); err != nil {
		return -1
	}
	return 0
}

//export SetMessageHandler
func SetMessageHandler(handler C.MessageHandler) {
	messageHandler = handler
}

//export CloseSession
func CloseSession(clientID *C.char) C.int {
	id := C.GoString(clientID)
	client := getClient(id)
	if client == nil {
		log.Println("[ERROR] Client not found:", id)
		return 0
	}
	if err := client.Close(); err != nil {
		log.Println("[ERROR] Failed to close session:", err)
		return -1
	}
	clients.Delete(id)
	log.Println("[INFO] Session closed for client:", id)
	return 0
}

func main() { /* No-op */ }
