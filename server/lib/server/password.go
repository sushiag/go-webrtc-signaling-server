package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sushiag/go-webrtc-signaling-server/server/lib/db"
)

type ChangePasswordPayload struct {
	Username    string `json:"username"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func ChangePasswordHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var input ChangePasswordPayload
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil ||
			strings.TrimSpace(input.Username) == "" ||
			strings.TrimSpace(input.OldPassword) == "" ||
			strings.TrimSpace(input.NewPassword) == "" {
			http.Error(w, "Invalid input", http.StatusUnprocessableEntity)
			return
		}

		result, err := queries.UpdateUserPassword(r.Context(), db.UpdateUserPasswordParams{
			Password:    input.NewPassword,
			Username:    input.Username,
			OldPassword: input.OldPassword,
		})
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if result.RowsAffected == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
