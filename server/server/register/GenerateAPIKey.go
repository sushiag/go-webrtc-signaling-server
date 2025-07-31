package sqlitedb

import (
	"crypto/rand"
	"encoding/hex"
)

// This generates the API-Key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
