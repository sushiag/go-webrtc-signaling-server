package client

import (
	"fmt"
	"os"
)

const apiKeyEnvName = "API_KEY"

func NewClient(wsEndpoint string) (Client, error) {
	var client Client

	apiKey, keyExists := os.LookupEnv(apiKeyEnvName)
	if !keyExists {
		return client, fmt.Errorf("the API_KEY")
	}

	return NewClientWithKey(wsEndpoint, apiKey)
}

func NewClientWithKey(wsEndpoint string, apiKey string) (Client, error) {
	var client Client
	eventsCh := make(chan Event, 32)

	if apiKey == "" {
		return client, fmt.Errorf("the apiKey cannot be an empty string")
	}

	pm := newPeerManager(eventsCh)

	signalingMngr, err := newSignalingManager(wsEndpoint, apiKey, pm, eventsCh)
	if err != nil {
		return client, err
	}

	client.signalingMngr = signalingMngr
	client.eventsCh = eventsCh

	return client, nil
}
