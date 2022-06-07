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
type AuthFullRequest struct {
	DeviceId auth.DeviceId `json:"deviceId"`
	Email    auth.Email    `json:"email"`
	Password auth.Password `json:"password"`
}

func (r *AuthFullRequest) validate() bool {
	return (r.DeviceId != "" &&
		r.Email != auth.Email("") && // TODO email validation. Here or store. Stdlib does it: https://stackoverflow.com/a/66624104
		r.Password != auth.Password(""))
}

func (s *Server) getAuthTokenFull(w http.ResponseWriter, req *http.Request) {
	var authRequest AuthFullRequest
	if !getPostData(w, req, &authRequest) {
		return
	}

	userId, err := s.store.GetUserId(authRequest.Email, authRequest.Password)
	if err == store.ErrNoUId {
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
