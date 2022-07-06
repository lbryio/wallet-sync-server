package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/store"
)

// DeviceId is decided by the device. UserId is decided by the server, and is
// gatekept by Email/Password
type AuthRequest struct {
	DeviceId auth.DeviceId `json:"deviceId"`
	Email    auth.Email    `json:"email"`
	Password auth.Password `json:"password"`
}

// TODO - validate funcs probably should return error rather than bool for
// idiomatic golang
func (r *AuthRequest) validate() bool {
	return r.DeviceId != "" && r.Password != auth.Password("") && validateEmail(r.Email)
}

func (s *Server) getAuthToken(w http.ResponseWriter, req *http.Request) {
	var authRequest AuthRequest
	if !getPostData(w, req, &authRequest) {
		return
	}

	userId, err := s.store.GetUserId(authRequest.Email, authRequest.Password)
	if err == store.ErrWrongCredentials {
		errorJson(w, http.StatusUnauthorized, "No match for email and password")
		return
	}
	if err != nil {
		internalServiceErrorJson(w, err, "Error getting User Id")
		return
	}

	authToken, err := s.auth.NewToken(userId, authRequest.DeviceId, auth.ScopeFull)

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
