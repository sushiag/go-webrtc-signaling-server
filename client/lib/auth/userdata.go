package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func Register(serverURL, username, password string) error {
	payload := map[string]string{
		"username": username,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post("http://"+serverURL+"/register", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("register failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusConflict:
		return fmt.Errorf("username already taken")
	case http.StatusUnprocessableEntity:
		return fmt.Errorf("invalid input")
	default:
		return fmt.Errorf("register failed: %s", resp.Status)
	}
}

func ChangePassword(serverURL, username, oldPass, newPass string) error {
	payload := map[string]string{
		"username":     username,
		"old_password": oldPass,
		"new_password": newPass,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post("http://"+serverURL+"/change_pass", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("change password failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("incorrect old password")
	case http.StatusUnprocessableEntity:
		return fmt.Errorf("invalid input")
	default:
		return fmt.Errorf("change password failed: %s", resp.Status)
	}
}
