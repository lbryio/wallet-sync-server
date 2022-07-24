package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"lbryio/lbry-id/auth"
	"lbryio/lbry-id/store"
)

// Thanks to Standard Notes. See:
// https://docs.standardnotes.com/specification/encryption/
//
// Auditor: I don't really understand how this system exactly works, and if
// I'm doing it right here, given that I don't understand it. In particular:
// Email address isn't sufficient for a secure salt, but it *is* somehow
// sufficient to keep the server from lying to us about passing the seed
// between clients? Is that the idea?

type ClientSaltSeedResponse struct {
	ClientSaltSeed auth.ClientSaltSeed `json:"clientSaltSeed"`
}

// TODO - There's probably a struct-based solution here like with POST/PUT.
// We could put that struct up top as well.
// TODO - maybe common code with getWalletParams?
func getClientSaltSeedParams(req *http.Request) (email auth.Email, err error) {
	emailSlice, hasEmailSlice := req.URL.Query()["email"]

	if !hasEmailSlice || emailSlice[0] == "" {
		err = fmt.Errorf("Missing email parameter")
	}

	if err == nil {
		decodedEmail, err := base64.StdEncoding.DecodeString(emailSlice[0])
		if err == nil {
			email = auth.Email(decodedEmail)
		}
	}

	if !email.Validate() {
		email = ""
		err = fmt.Errorf("Invalid email")
	}

	return
}

func (s *Server) getClientSaltSeed(w http.ResponseWriter, req *http.Request) {
	if !getGetData(w, req) {
		return
	}

	email, paramsErr := getClientSaltSeedParams(req)

	if paramsErr != nil {
		// In this specific case, the error is limited to values that are safe to
		// give to the user.
		errorJson(w, http.StatusBadRequest, paramsErr.Error())
		return
	}

	seed, err := s.store.GetClientSaltSeed(email)
	if err == store.ErrWrongCredentials {
		errorJson(w, http.StatusNotFound, "No match for email")
		return
	}
	if err != nil {
		internalServiceErrorJson(w, err, "Error getting client salt seed")
		return
	}

	clientSaltSeedResponse := ClientSaltSeedResponse{
		ClientSaltSeed: seed,
	}

	response, err := json.Marshal(clientSaltSeedResponse)

	if err != nil {
		internalServiceErrorJson(w, err, "Error generating client salt seed response")
		return
	}

	fmt.Fprintf(w, string(response))
}
