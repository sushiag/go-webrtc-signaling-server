package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/sushiag/go-webrtc-signaling-server/server/lib/server/db"
	sqlitedb "github.com/sushiag/go-webrtc-signaling-server/server/lib/server/register"
	"golang.org/x/crypto/bcrypt"
)

// Registration ---

func (nh *Handler) registerNewUser(w http.ResponseWriter, r *http.Request) {
	var rqst struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	log.Printf("[REGISTER] Request to create account received!")

	if err := json.NewDecoder(r.Body).Decode(&rqst); err != nil {
		http.Error(w, "Invalid Input", http.StatusUnprocessableEntity)
		return
	}

	log.Printf("[REGISTER] Received: username=%s password=%s", rqst.Username, rqst.Password)

	// --- USERNAME & PASSWORD ---
	if err := checkUsernameField(rqst.Username); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err := checkPasswordField(rqst.Password); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// --- HASH PASSWORD ---
	hashed, _ := bcrypt.GenerateFromPassword([]byte(rqst.Password), bcrypt.DefaultCost)

	// --- API-Key ---
	apikey, err := sqlitedb.GenerateAPIKey()
	if err != nil {
		log.Printf("[REGISTER] Error generating API key: %v", err)
		http.Error(w, "Unable to generate API Key", http.StatusInternalServerError)
		return
	}
	// --- INSERT TO DB ---
	err = nh.Queries.CreateUser(r.Context(), db.CreateUserParams{
		Username: rqst.Username,
		Password: string(hashed),
		ApiKey:   apikey,
	})

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			log.Printf("[FAILED TO REGISTER ACCOUNT] Username already taken")
			http.Error(w, "Username is already taken", http.StatusConflict)
		} else {
			log.Printf("[REGISTER USERNAME] Unable to register username this time: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	resp := struct {
		APIKey string `json:"api_key"`
	}{
		APIKey: apikey,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[REGISTER] Failed to write JSON response: %v", err)
	}
}

// --- LOGIN USER VIA API-KEY ---

func (nh *Handler) getUserFromAPIKey(r *http.Request) (*db.User, error) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		// fallback if some clients send Authorization header instead
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			apiKey = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	if apiKey == "" {
		return nil, fmt.Errorf("API key missing")
	}

	user, err := nh.Queries.GetUserByApikeys(r.Context(), apiKey)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// -- REGENERATE API Key ---

func (nh *Handler) regenerateNewAPIKeys(w http.ResponseWriter, r *http.Request) {
	var rqst struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&rqst); err != nil {
		http.Error(w, "Invalid input", http.StatusUnprocessableEntity)
		return
	}
	// --- ACCESS USER ACCOUNT IN DATABASE ---
	user, err := nh.Queries.GetUserByUsername(r.Context(), rqst.Username)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(rqst.Password)) != nil {
		log.Printf("[LOGIN FOR NEW API KEY] Username and Password does not match")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// --- REGENENARATION LOGIC ---
	newAPIKey, err := sqlitedb.GenerateAPIKey()
	if err != nil {
		log.Printf("Failed to generate new API KEY")
		http.Error(w, "[GENERATE NEW API KEY] Failed to generate new API KEY", http.StatusInternalServerError)
		return
	}

	err = nh.Queries.UpdateAPIKey(r.Context(), db.UpdateAPIKeyParams{ // iF username & password passes, update API KEY
		ApiKey:   newAPIKey,
		Username: rqst.Username,
	})

	if err != nil { // if fails to update
		log.Printf("Failed to update API Key")
		http.Error(w, "[FAILURE] Failed to update API Key", http.StatusInternalServerError)
	}

	log.Printf("[SUCESS] Sucess: New API Key Generated!, %s", newAPIKey)

	resp := struct {
		APIKey string `json:"api_key"`
	}{
		APIKey: newAPIKey,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)

}

func (nh *Handler) updatePassword(w http.ResponseWriter, r *http.Request) {
	var rqst struct {
		Username    string `json:"username"`
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	log.Printf("[PASSWORD] Attemping to update password")

	if err := json.NewDecoder(r.Body).Decode(&rqst); err != nil {
		http.Error(w, "Invalid Input", http.StatusBadRequest)
		return
	}
	// --- GET USERNAME FROM DATABASE ---
	user, err := nh.Queries.GetUserByUsername(r.Context(), rqst.Username)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(rqst.OldPassword)) != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if rqst.NewPassword == rqst.OldPassword {
		http.Error(w, "New password must differ from old password", http.StatusUnprocessableEntity)
		return
	}
	if err := checkPasswordField(rqst.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(rqst.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[ERROR] Failed to hash new password: %v", err)
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	err = nh.Queries.UpdateUserPassword(r.Context(), db.UpdateUserPasswordParams{
		Username: rqst.Username,
		Password: string(hashed),
	})
	if err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	log.Printf("[PASSWORD] Password updated for user: %s", rqst.Username)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Password updated successfully"))
}
