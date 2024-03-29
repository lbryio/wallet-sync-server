package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"lbryio/wallet-sync-server/auth"
	"lbryio/wallet-sync-server/store"
)

// DeviceId is decided by the device. UserId is decided by the server, and is
// gatekept by Email/Password
type AuthRequest struct {
	DeviceId auth.DeviceId `json:"deviceId"`
	Email    auth.Email    `json:"email"`
	Password auth.Password `json:"password"`
}

func (r *AuthRequest) validate() error {
	if !r.Email.Validate() {
		return fmt.Errorf("Invalid 'email'")
	}
	if !r.Password.Validate() {
		return fmt.Errorf("Invalid or missing 'password'")
	}
	if r.DeviceId == "" {
		return fmt.Errorf("Missing 'deviceId'")
	}
	return nil
}

func (s *Server) getAuthToken(w http.ResponseWriter, req *http.Request) {
	var authRequest AuthRequest
	if !getPostData(w, req, &authRequest) {
		return
	}

	userId, err := s.store.GetUserId(authRequest.Email, authRequest.Password)
	if err == store.ErrWrongCredentials {
		errorJson(w, http.StatusUnauthorized, "No match for email and/or password")
		return
	}
	if err == store.ErrNotVerified {
		errorJson(w, http.StatusUnauthorized, "Account is not verified")
		return
	}
	if err != nil {
		internalServiceErrorJson(w, err, "Error getting User Id")
		return
	}

	authToken, err := s.auth.NewAuthToken(userId, authRequest.DeviceId, auth.ScopeFull)

	if err != nil {
		internalServiceErrorJson(w, err, "Error generating auth token")
		return
	}

	response, err := json.Marshal(&authToken)

	if err != nil {
		internalServiceErrorJson(w, err, "Error generating auth token")
		return
	}

	if err := s.store.SaveToken(authToken); err != nil {
		internalServiceErrorJson(w, err, "Error saving auth token")
		return
	}

	fmt.Fprintf(w, string(response))
}
