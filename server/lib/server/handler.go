package server

import (
	"encoding/json"
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

	username := rqst.Username
	password := rqst.Password

	// Fields input on registering for every new user

	switch {
	// --- USERNAME ---
	case !usernameReg.MatchString(username):
		{
			log.Printf("Invalid Fields: letters, numbers, underscore only, 16 max characters only")
			http.Error(w, "[INVALID FIELDS] letters, numbers, underscore only, 16 max characters only", http.StatusUnprocessableEntity)
			return
		}
	case noWhitespace(username):
		{
			log.Printf("Invalid Fields: No Whitespace on Username Allowed")
			http.Error(w, "[INVALID FIELDS] No Whitespace on Username Allowed", http.StatusUnprocessableEntity)
		}

	// --- PASSWORD ---
	case len(password) > 32:
		{
			log.Printf("Invalid Fields: Password too long, Max. 32 characters only")
			http.Error(w, "[INVALID FIELDS] Exceeded Max. Character (32)", http.StatusUnprocessableEntity)
			return
		}
	case len(password) < 8:
		{
			log.Printf("Invalid Fields: Password too short, Atleast Min. of 8 characters")
			http.Error(w, "[INVALID FIELDS] Atleast Min. 8 Characters", http.StatusUnprocessableEntity)
			return
		}
	case noWhitespace(password):
		{
			log.Printf("Invalid Fields: Password Must not Contain Whitespace")
			http.Error(w, "[INVALID FIELDS] No Whitespaces Allowed", http.StatusUnprocessableEntity)
			return
		}

	// --- USERNAME & PASSWORD ---

	case !onlyASCII(username) || !onlyASCII(password):
		{
			log.Printf("Invalid Fields: Username and password must only use ASCII")
			http.Error(w, "[INVALID FIELDS] ASCIII Only", http.StatusUnprocessableEntity)
		}
	default:
		{
			log.Printf("[REGISTER] Success!")
		}
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

	log.Printf("[REGISTER] Ready to insert user into DB: username=%s apikey=%s", rqst.Username, apikey)

	err = nh.Queries.CreateUser(r.Context(), db.CreateUserParams{
		Username: rqst.Username,
		Password: string(hashed),
		ApiKey:   apikey,
	})

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			http.Error(w, "Username is already taken", http.StatusConflict)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		log.Printf("[REGISTER] DB error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("[REGISTER] Success: username=%s apikey=%s", username, apikey)

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

	log.Printf("[LOGIN] Username: %s", rqst.Username)

	// -- LOGIN LOGIC --
	user, err := nh.Queries.GetUserByUsername(r.Context(), rqst.Username)
	if err != nil {
		log.Printf("Invalid Username")
		http.Error(w, "[INVALID USERNAME] Username does not exist", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(rqst.Password)); err != nil {
		log.Printf("Invalid Password")
		http.Error(w, "[INVALID PASSWORD] Password does not match", http.StatusUnauthorized)
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
	if err != nil {
		log.Printf("Username not found")
		http.Error(w, "[INVALID USERNAME] Username not found", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(rqst.Password)); err != nil {
		http.Error(w, "[INVALID PASSWORD] Password mismatch", http.StatusUnauthorized)
		return
	}

	// --- REGENENARATION LOGIC ---
	newAPIKey, err := sqlitedb.GenerateAPIKey()
	if err != nil {
		log.Printf("Failed to generate new API KEY")
		http.Error(w, "[FAILURE] Failed to generate new API KEY", http.StatusInternalServerError)
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
	if err != nil {
		log.Printf("%s not found within the system", rqst.Username)
		http.Error(w, "[INVALID USERNAME] Username not found", http.StatusUnauthorized)
		return
	}

	// --- PASSWORD LOGIC ---
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(rqst.OldPassword)); err != nil {
		log.Printf("Password does not match account")
		http.Error(w, "[FAILURE TO CHANGE PASSWORD]Password does not match", http.StatusUnauthorized)
		return
	}

	switch {
	case len(rqst.NewPassword) > 32:
		{
			log.Printf("Password exceed 32 character max")
			http.Error(w, "[INVALID PASSWORD] Password shouldn't exceed more than 32 characters", http.StatusUnprocessableEntity)
			return
		}
	case len(rqst.NewPassword) < 8:
		{
			log.Printf("Password should be atleast 8 chracters")
			http.Error(w, "[INVALLID PASSWORD] Password should atleast be 8 characters", http.StatusUnprocessableEntity)
			return
		}
	case noWhitespace(rqst.NewPassword):
		{
			log.Printf("Password shouldn't contain spaces")
			http.Error(w, "[INVALID PASSWORD] Password must not contain Whiitespace", http.StatusUnprocessableEntity)
			return
		}
	case (rqst.NewPassword) == (rqst.OldPassword):
		{
			log.Printf("Password must not be the same as the old password")
			http.Error(w, "[INVALID PASSSWORD] Password already used", http.StatusUnprocessableEntity)
			return
		}
	case !onlyASCII(rqst.NewPassword):
		{
			log.Printf("Password has invalid characters")
			http.Error(w, "[INVALID PASSWORD] Password invalid characters", http.StatusUnprocessableEntity)
			return
		}
	default:
		log.Printf("Suceesfully changed password")
	}

	// Hash new password
	hashed, err := bcrypt.GenerateFromPassword([]byte(rqst.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	// --- UPDATED PASSWORD ---
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
