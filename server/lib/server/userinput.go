package server

import (
	"errors"
	"log"
	"regexp"
	"unicode"

	"github.com/sushiag/go-webrtc-signaling-server/server/lib/server/db"
)

type Handler struct {
	Queries *db.Queries
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
	case username == "":
		log.Printf("Username field: must not be blank")
		return errors.New("username shouldn't be blank")
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
	}
	return nil
}

func checkPasswordField(password string) error {
	switch {
	case password == "":
		log.Printf("Password field: must not be blank")
		return errors.New("password must not be blank")
	case len(password) < 8 || len(password) > 32:
		log.Printf("Password field: should only be 8-32 characters")
		return errors.New("password should only be 8-32 characters only")
	case noWhitespace(password):
		log.Printf("Password field: no whitespace allowed")
		return errors.New("password should not contain whitespaces")
	case !onlyASCII(password):
		log.Printf("Username field: ONLYASCII allowed")
		return errors.New("password must only be ACSII")
	}
	return nil
}
