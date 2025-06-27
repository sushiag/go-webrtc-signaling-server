package websocket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func (c *Client) PreAuthenticate() error {
	payload := map[string]string{"apikey": c.ApiKey}
	body, _ := json.Marshal(payload)

	authUrl := fmt.Sprintf("http://%s/auth", c.ServerURL)
	resp, err := http.Post(authUrl, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth failed: %s", resp.Status)
	}

	var result struct {
		UserID uint64 `json:"userid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	c.UserID = result.UserID
	return nil
}

func (c *Client) Init() error {
	headers := http.Header{}
	headers.Set("X-Api-Key", c.ApiKey) // set the auth header

	wsEndpoint := fmt.Sprintf("ws://%s/ws", c.ServerURL)
	conn, _, err := websocket.DefaultDialer.Dial(wsEndpoint, headers)
	if err != nil {
		return fmt.Errorf("[CLIENT SIGNALING] websocket connection failed: %v", err)
	}

	log.Println("[CLIENT SIGNALING] Connected to:", c.ServerURL)
	c.Conn = conn
	return nil
}
