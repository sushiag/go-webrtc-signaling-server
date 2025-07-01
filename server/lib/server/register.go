package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sushiag/go-webrtc-signaling-server/server/lib/db"
)

type UserPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func RegisterHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var input UserPayload
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.Username) == "" || strings.TrimSpace(input.Password) == "" {
			http.Error(w, "Invalid input", http.StatusUnprocessableEntity)
			return
		}

		err := queries.CreateUser(r.Context(), db.CreateUserParams{
			Username: input.Username,
			Password: input.Password,
		})
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				http.Error(w, "Username already taken", http.StatusConflict)
			} else {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
