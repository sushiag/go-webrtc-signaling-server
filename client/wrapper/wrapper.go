package webrtcclient

import (
	"fmt"

	clienthandle "github.com/sushiag/go-webrtc-signaling-server/client/clienthandler"
)

var client *clienthandle.Client

func InitWebRTCClient(apiKey string, signalingURL string) error {
	client = clienthandle.NewClient()
	client.ApiKey = apiKey
	client.ServerURL = signalingURL
	if err := client.PreAuthenticate(); err != nil {
		return err
	}
	return client.Init()
}

func StartSession(sessionID string) error {
	return client.Start()
}

func JoinSession(sessionID string) error {
	return client.Join(sessionID)
}

func SendSignalingMessage(targetID string, message []byte) error {
	return client.SendMessage(targetID, message)
}

func SetMessageHandler(callback func(sourceID string, message []byte)) {
	client.SetMessageHandler(func(msg clienthandle.Message) {
		callback(fmt.Sprintf("%d", msg.FromID), msg.Data)
	})
}

func CloseSession() error {
	client.Close()
	return nil
}
