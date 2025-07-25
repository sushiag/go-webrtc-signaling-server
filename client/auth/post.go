package post

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Credentials struct {
	Username string
	Password string
}

func Registration(wsurl, username, password string) error {
	url := fmt.Sprintf("%s/register", wsurl)
	body := map[string]string{
		"Username": username,
		"Password": password,
	}

	b, err := json.Marshal(body)

	if err != nil {
		return fmt.Errorf("[MARSHAL] Registration failed: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("[POST] Registration Failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Unexxpected status during registration: %s", resp.Status)
	}

	return nil
}

func GetLoginFromAPIKey(url string, apiKey string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	return client.Do(req)
}

func RegenerateAPIKey(wsurl, username, password string) (string, error) {
	url := fmt.Sprintf("%s/regenerate", wsurl)
	body := map[string]string{
		"Username": username,
		"Password": password,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("[MARSHAL] Error: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("[POST] Failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Unexxpected Error: %s", resp.Status)
	}

	var res struct {
		APIKey string `json:"api_key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("[DECODE Error] error cause: %w", err)
	}
	return res.APIKey, nil
}

func ResetPassword(baseURL, username, oldPassword, newPassword string) error {
	url := fmt.Sprintf("%s/newpassword", baseURL)
	body := map[string]string{
		"username":     username,
		"old_password": oldPassword,
		"new_password": newPassword,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("POST failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return nil
}
