package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// TODO proper doc comments!

const PathGetAuthToken = "/auth"

// Server: The interface definition for the Server module
type Server struct {
	auth  AuthInterface
	store StoreInterface
}

////
// Requests
////

/*
TODO - consider reworking the naming convention

Rename `AuthRequest` struct to:

type TokenRequest struct {
	BodyJSON  string // or maybe PayloadJSON
	PubKey    PublicKey
	Signature string
}

And then rename the existing `TokenRequest` to `TokenRequestBody` (this is what `BodyJSON` unmarhals to).

The reason is that we'll need this format for walletState eventually. The walletState as such, saved on devices,
passed around, etc, should contain the signature and the public key, but of course the signed portion itself
cannot contain the signature.

So, the part not containing the signature we'll similarly call something lke `WalletStateBody`. `WaletStateBody`
will still contain other metadata about the encrypted wallet such as DeviceID and Sequence. We only keep the
signature and PubKey outside because of the verification process. We could keep the PubKey inside the Body but
it's more convenient this way, and the signature process will verify it along with the body.
*/

type AuthRequest struct {
	// TokenRequestJSON: json string within json, so that the string representation is
	//   unambiguous for the purposes of signing. This means we need to deserialize the
	//   request body twice.
	TokenRequestJSON string    `json:"tokenRequestJSON"`
	PubKey           PublicKey `json:"publicKey"`
	Signature        string    `json:"signature"`
}

func (s *Server) validateAuthRequest(payload *AuthRequest) bool {
	// TODO
	return true
}

////
// Responses
////

type ErrorResponse struct {
	Error string `json:"error"`
}

func errorJSON(w http.ResponseWriter, code int, extra string) {
	errorStr := http.StatusText(code)
	if extra != "" {
		errorStr = errorStr + ": " + extra
	}
	authErrorJSON, err := json.Marshal(ErrorResponse{Error: errorStr})
	if err != nil {
		// In case something really stupid happens
		http.Error(w, `{"error": "error when JSON-encoding error message"}`, code)
	}
	http.Error(w, string(authErrorJSON), code)
	return
}

////
// Handlers
////

func (s *Server) getAuthToken(w http.ResponseWriter, req *http.Request) {
	/*
	  (This comment may only be needed for WIP)

	   Server should be in charge of such things as:
	   * Request body size check (in particular to not tie up signature check)
	   * JSON validation/deserialization

	   Auth should be in charge of such things as:
	   * Checking signatures
	   * Generating tokens

	   The order of events:
	   * Server checks the request body size
	   * Server deserializes and then validates the AuthRequest
	   * Auth checks the signature of authRequest.TokenRequestJSON
	     * This the awkward bit, since Auth is being passed a (serialized) JSON string.
	       However, it's not deserializing it. It's ONLY checking the signature of it
	       as a string per se. (The same function will be used for signed walletState)
	   * Server deserializes and then validates the TokenRequest
	   * Auth takes TokenRequest and PubKey and generates a token
	   * DataStore stores the token. The pair (PubKey, TokenRequest.DeviceID) is the primary key.
	       We should have one token for each device.
	*/

	if req.Method != http.MethodPost {
		errorJSON(w, http.StatusMethodNotAllowed, "")
		return
	}

	/*
		TODO - http.StatusRequestEntityTooLarge for some arbitrary large size
		see:
		* MaxBytesReader or LimitReader
		* https://pkg.go.dev/net/http#Request.ParseForm
		* some library/framework that handles it (along with req.Method)
	*/

	var authRequest AuthRequest
	if err := json.NewDecoder(req.Body).Decode(&authRequest); err != nil {
		errorJSON(w, http.StatusBadRequest, "Malformed request body JSON")
		return
	}

	if s.validateAuthRequest(&authRequest) {
		// TODO
	}

	if !s.auth.IsValidSignature(authRequest.PubKey, authRequest.TokenRequestJSON, authRequest.Signature) {
		errorJSON(w, http.StatusForbidden, "Bad signature")
		return
	}

	var tokenRequest TokenRequest
	if err := json.Unmarshal([]byte(authRequest.TokenRequestJSON), &tokenRequest); err != nil {
		errorJSON(w, http.StatusBadRequest, "Malformed tokenRequest JSON")
		return
	}

	if s.validateTokenRequest(&tokenRequest) {
		// TODO
	}

	authToken, err := s.auth.NewToken(authRequest.PubKey, &tokenRequest)

	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "Error generating auth token")
		log.Print(err)
		return
	}

	response, err := json.Marshal(&authToken)

	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "Error generating auth token")
		log.Print(err)
		return
	}

	if err := s.store.SaveToken(authToken); err != nil {
		errorJSON(w, http.StatusInternalServerError, "Error saving auth token")
		log.Print(err)
		return
	}

	fmt.Fprintf(w, string(response))
}

func (s *Server) getWalletState(w http.ResponseWriter, req *http.Request) {
	// TODO
	// GET request only
	// !(AuthToken.Valid && (AuthToken.Scope == "*" || AuthToken.Scope == "download")) -> http.StatusNotAllowed
}

func (s *Server) putWalletState(w http.ResponseWriter, req *http.Request) {
	// TODO
	// POST request only
	// !(AuthToken.Valid && AuthToken.Scope == "*") -> http.StatusNotAllowed
}

func main() {
	server := Server{&Auth{}, &Store{}}

	http.HandleFunc(PathGetAuthToken, server.getAuthToken)

	// TODO
	//http.HandleFunc("/get-wallet-state", server.getWalletState)
	//http.HandleFunc("/put-wallet-state", server.putWalletState)

	http.ListenAndServe(":8090", nil)
}
