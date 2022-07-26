package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"lbryio/lbry-id/auth"
	"lbryio/lbry-id/env"
	"lbryio/lbry-id/store"
)

type RegisterRequest struct {
	Email          auth.Email          `json:"email"`
	Password       auth.Password       `json:"password"`
	ClientSaltSeed auth.ClientSaltSeed `json:"clientSaltSeed"`
}

type RegisterResponse struct {
	Verified bool `json:"verified"`
}

func (r *RegisterRequest) validate() error {
	if !r.Email.Validate() {
		return fmt.Errorf("Invalid or missing 'email'")
	}
	if r.Password == "" {
		return fmt.Errorf("Missing 'password'")
	}

	if !r.ClientSaltSeed.Validate() {
		return fmt.Errorf("Invalid or missing 'clientSaltSeed'")
	}
	return nil
}

func (s *Server) register(w http.ResponseWriter, req *http.Request) {
	var registerRequest RegisterRequest
	if !getPostData(w, req, &registerRequest) {
		return
	}

	verificationMode, err := env.GetAccountVerificationMode(s.env)
	if err != nil {
		internalServiceErrorJson(w, err, "Error getting account verification mode")
		return
	}
	accountWhitelist, err := env.GetAccountWhitelist(s.env, verificationMode)
	if err != nil {
		internalServiceErrorJson(w, err, "Error getting account whitelist")
		return
	}

	var registerResponse RegisterResponse

modes:
	switch verificationMode {
	case env.AccountVerificationModeAllowAll:
		// Always verified (for testers). No need to jump through email verify
		// hoops.
		registerResponse.Verified = true
	case env.AccountVerificationModeWhitelist:
		for _, whitelisteEmail := range accountWhitelist {
			if whitelisteEmail == registerRequest.Email {
				registerResponse.Verified = true
				break modes
			}
		}
		// If we have unverified users on whitelist setups, we'd need to create a way
		// to verify them. It's easier to just prevent account creation. It also will
		// make it easier for self-hosters to figure out that something is wrong
		// with their whitelist.
		errorJson(w, http.StatusForbidden, "Account not whitelisted")
		return
	case env.AccountVerificationModeEmailVerify:
		// Not verified until they click their email link.
		registerResponse.Verified = false
	}

	err = s.store.CreateAccount(
		registerRequest.Email,
		registerRequest.Password,
		registerRequest.ClientSaltSeed,
		registerResponse.Verified,
	)

	if err != nil {
		if err == store.ErrDuplicateEmail || err == store.ErrDuplicateAccount {
			errorJson(w, http.StatusConflict, "Error registering")
		} else {
			internalServiceErrorJson(w, err, "Error registering")
		}
		return
	}

	response, err := json.Marshal(registerResponse)

	if err != nil {
		internalServiceErrorJson(w, err, "Error generating register response")
		return
	}

	// TODO StatusCreated also for first wallet and/or for get auth token?
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, string(response))
}
