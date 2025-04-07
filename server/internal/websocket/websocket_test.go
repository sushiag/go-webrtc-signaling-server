package websocket

import "testing"

func TestLoadApiKeys(t *testing.T) {
	keys, err := LoadValidApiKeys("../test_apikeys.txt")

	if err != nil {
		t.Errorf("failed to read file %s", err)
	}

	expectedKeys := map[string]bool{
		"valid-api-key-1": true,
		"valid-api-key-2": true,
		"valid-api-key-3": true,
	}

	if !mapsEqual(keys, expectedKeys) {
		t.Errorf("got %+v; want %+v", keys, expectedKeys)
	}
}

// helper function to compare two maps
func mapsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
