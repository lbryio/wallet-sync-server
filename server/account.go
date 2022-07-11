package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/store"
)

// TODO email verification cycle

type RegisterRequest struct {
	Email    auth.Email    `json:"email"`
	Password auth.Password `json:"password"`
}

func (r *RegisterRequest) validate() error {
	if !validateEmail(r.Email) {
		return fmt.Errorf("Invalid 'email'")
	}
	if r.Password == "" {
		return fmt.Errorf("Missing 'password'")
	}
	return nil
}

func (s *Server) register(w http.ResponseWriter, req *http.Request) {
	var registerRequest RegisterRequest
	if !getPostData(w, req, &registerRequest) {
		return
	}

	err := s.store.CreateAccount(registerRequest.Email, registerRequest.Password)

	if err != nil {
		if err == store.ErrDuplicateEmail || err == store.ErrDuplicateAccount {
			errorJson(w, http.StatusConflict, "Error registering")
		} else {
			internalServiceErrorJson(w, err, "Error registering")
		}
		return
	}

	var registerResponse struct{} // no data to respond with, but keep it JSON
	var response []byte
	response, err = json.Marshal(registerResponse)

	if err != nil {
		internalServiceErrorJson(w, err, "Error generating register response")
		return
	}

	// TODO StatusCreated also for first wallet and/or for get auth token?
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, string(response))
}
