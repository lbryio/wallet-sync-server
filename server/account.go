package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"lbryio/wallet-sync-server/auth"
	"lbryio/wallet-sync-server/env"
	"lbryio/wallet-sync-server/store"
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
	if !r.Password.Validate() {
		return fmt.Errorf("Invalid or missing 'password'")
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

	var token *auth.VerifyTokenString

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
		newToken, err := s.auth.NewVerifyTokenString()
		token = &newToken

		if err != nil {
			internalServiceErrorJson(w, err, "Error generating verify token string")
			return
		}
	}

	err = s.store.CreateAccount(
		registerRequest.Email,
		registerRequest.Password,
		registerRequest.ClientSaltSeed,
		token, // if it's not set, the user is marked as verified
	)

	if err != nil {
		if err == store.ErrDuplicateEmail || err == store.ErrDuplicateAccount {
			errorJson(w, http.StatusConflict, "Error registering")
		} else {
			internalServiceErrorJson(w, err, "Error registering")
		}
		return
	}

	if token != nil {
		err = s.mail.SendVerificationEmail(registerRequest.Email, *token)
	}

	if err != nil {
		internalServiceErrorJson(w, err, "Error sending verification email")
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
	log.Printf("User %s has registered", registerRequest.Email)
}

// TODO - There's probably a struct-based solution here like with POST/PUT.
// We could put that struct up top as well.
func getVerifyParams(req *http.Request) (token auth.VerifyTokenString, err error) {
	tokenSlice, hasTokenSlice := req.URL.Query()["verifyToken"]

	if !hasTokenSlice || tokenSlice[0] == "" {
		err = fmt.Errorf("Missing verifyToken parameter")
	}

	if err == nil {
		token = auth.VerifyTokenString(tokenSlice[0])
	}

	return
}

type ResendVerifyEmailRequest struct {
	Email auth.Email `json:"email"`
}

func (r *ResendVerifyEmailRequest) validate() error {
	if !r.Email.Validate() {
		return fmt.Errorf("Invalid or missing 'email'")
	}
	return nil
}

func (s *Server) resendVerifyEmail(w http.ResponseWriter, req *http.Request) {
	verificationMode, err := env.GetAccountVerificationMode(s.env)
	if err != nil {
		internalServiceErrorJson(w, err, "Error getting account verification mode")
		return
	}
	if verificationMode != env.AccountVerificationModeEmailVerify {
		errorJson(w, http.StatusForbidden, "Account verification mode is not set to EmailVerify")
		return
	}

	var resendVerifyEmailRequest ResendVerifyEmailRequest
	if !getPostData(w, req, &resendVerifyEmailRequest) {
		return
	}

	token, err := s.auth.NewVerifyTokenString()

	if err != nil {
		internalServiceErrorJson(w, err, "Error generating verify token string")
		return
	}

	err = s.store.UpdateVerifyTokenString(resendVerifyEmailRequest.Email, token)
	if err == store.ErrWrongCredentials {
		errorJson(w, http.StatusUnauthorized, "No match for email")
		return
	}
	if err != nil {
		internalServiceErrorJson(w, err, "Error updating verify token string")
		return
	}

	err = s.mail.SendVerificationEmail(resendVerifyEmailRequest.Email, token)

	if err != nil {
		internalServiceErrorJson(w, err, "Error re-sending verification email")
		return
	}

	var verifyResponse struct{}
	response, err := json.Marshal(verifyResponse)

	if err != nil {
		internalServiceErrorJson(w, err, "Error generating verify response")
		return
	}

	fmt.Fprintf(w, string(response))
}

func (s *Server) verify(w http.ResponseWriter, req *http.Request) {
	if !getGetData(w, req) {
		return
	}

	token, paramsErr := getVerifyParams(req)

	if paramsErr != nil {
		// In this specific case, the error is limited to values that are safe to
		// give to the user.
		http.Error(w, "There seems to be a problem with this URL: "+paramsErr.Error(), http.StatusBadRequest)
		return
	}

	err := s.store.VerifyAccount(token)

	if err == store.ErrNoTokenForUser {
		http.Error(w, "The verification token was not found, already used, or expired. If you want to try again, generate a new one from your app.", http.StatusForbidden)
		return
	} else if err != nil {
		http.Error(w, "Something went wrong trying to verify your account.", http.StatusInternalServerError)
		log.Printf("%s: %+v\n", "Error verifying account", err)
		return
	}

	fmt.Fprintf(w, "Your account has been verified.")

	// if we really want to log the user's email at some point
	// we can put in the effort then to fetch it
	log.Printf("User has been verified with token %s", token)
}
