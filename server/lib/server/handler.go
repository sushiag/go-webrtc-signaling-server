package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/sushiag/go-webrtc-signaling-server/server/lib/db"
	sqlitedb "github.com/sushiag/go-webrtc-signaling-server/server/lib/register"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	Queries *db.Queries
}

func NewHandler(nh *db.Queries) *Handler {
	return &Handler{Queries: nh}
}

var (
	usernameReg = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,15}$`)
)

func onlyASCII(in string) bool {
	for _, r := range in {
		if r < 32 || r > 126 { // no chinese char allowed
			return false
		}
	}
	return true
}

func noWhitespace(in string) bool {
	for _, r := range in {
		if unicode.IsSpace(r) { // no white allowed
			return true
		}
	}
	return false
}

func checkUsernameField(username string) error {
	switch {
	case len(username) < 8 || len(username) > 16:
		log.Printf("Username field: Username must be 8–16 characters")
		return errors.New("username must be between 8 and 16 characters")
	case !usernameReg.MatchString(username):
		log.Printf("Username field: invalid characters")
		return errors.New("username should only contain alphanumeric/underscore and max 16 characters")
	case noWhitespace(username):
		log.Printf("Username field: shouldn't contain whitespace")
		return errors.New("username shouldn't have whitespaces")
	case !onlyASCII(username):
		log.Printf("Username field: only ASCII allowed")
		return errors.New("username must only be contain ASCII")
	case username == "":
		log.Printf("Username field: must not be blank")
		return errors.New("username shouldn't be blank")

	}
	return nil
}

func checkPasswordField(password string) error {
	switch {
	case len(password) < 8 || len(password) > 32:
		log.Printf("Password field: should only be 8-32 characters")
		return errors.New("password should only be 8-32 characters only")
	case noWhitespace(password):
		log.Printf("Password field: no whitespace allowed")
		return errors.New("password should not contain whitespaces")
	case !onlyASCII(password):
		log.Printf("Username field: ONLYASCII allowed")
		return errors.New("password must only be ACSII")
	case password == "":
		log.Printf("Password field: must not be blank")
		return errors.New("password must not be blank")
	}
	return nil
}

// --- Registration ---

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

func (nh *Handler) loginUser(w http.ResponseWriter, r *http.Request) {
	var rqst struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	log.Printf("[LOGIN] Attemping to Login")

	if err := json.NewDecoder(r.Body).Decode(&rqst); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	// -- LOGIN LOGIC --

	user, err := nh.Queries.GetUserByUsername(r.Context(), rqst.Username)
	if err != nil {
		log.Printf("[LOGIN] Username does not exist")
		http.Error(w, "Invalid username", http.StatusUnauthorized)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(rqst.Password)) != nil {
		log.Printf("[LOGIN] Password does not match username")
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	log.Printf("[LOGIN] Login Sucessful, Welcome Back user %s", user.Username)

	// Return API key
	resp := struct {
		APIKey string `json:"api_key"`
	}{
		APIKey: user.ApiKey.(string),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[LOGIN] Failed to send response: %v", err)
	}
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
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(rqst.OldPassword)); err != nil {
		log.Printf("[PASSWORD] Old password mismatch")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(rqst.OldPassword)); err != nil {
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
