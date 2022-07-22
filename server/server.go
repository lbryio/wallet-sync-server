package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/mail"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"lbryio/lbry-id/auth"
	"lbryio/lbry-id/store"
)

// TODO proper doc comments!

const ApiVersion = "3"
const PathPrefix = "/api/" + ApiVersion

const PathPrometheus = "/metrics"

const PathAuthToken = PathPrefix + "/auth/full"
const PathRegister = PathPrefix + "/signup"
const PathPassword = PathPrefix + "/password"
const PathWallet = PathPrefix + "/wallet"
const PathClientSaltSeed = PathPrefix + "/client-salt-seed"

const PathUnknownEndpoint = PathPrefix + "/"
const PathWrongApiVersion = "/api/"

type Server struct {
	auth  auth.AuthInterface
	store store.StoreInterface
}

func Init(
	auth auth.AuthInterface,
	store store.StoreInterface,
) *Server {
	return &Server{
		auth:  auth,
		store: store,
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func errorJson(w http.ResponseWriter, code int, extra string) {
	errorStr := http.StatusText(code)
	if extra != "" {
		errorStr = errorStr + ": " + extra
	}
	authErrorJson, err := json.Marshal(ErrorResponse{Error: errorStr})
	if err != nil {
		// In case something really stupid happens
		http.Error(w, `{"error": "error when JSON-encoding error message"}`, code)
	}
	http.Error(w, string(authErrorJson), code)
	return
}

// Don't report any details to the user. Log it instead.
func internalServiceErrorJson(w http.ResponseWriter, serverErr error, errContext string) {
	errorStr := http.StatusText(http.StatusInternalServerError)
	authErrorJson, err := json.Marshal(ErrorResponse{Error: errorStr})
	if err != nil {
		// In case something really stupid happens
		http.Error(w, `{"error": "error when JSON-encoding error message"}`, http.StatusInternalServerError)
		log.Printf("error when JSON-encoding error message")
		return
	}
	http.Error(w, string(authErrorJson), http.StatusInternalServerError)
	log.Printf("%s: %+v\n", errContext, serverErr)

	return
}

//////////////////
// Handler Helpers
//////////////////

// Cut down on code repetition. No need to return errors since it can all be
// handled here. Just return a bool to indicate success.
// TODO the names `getPostData` and `getGetData` don't fully describe what they do

func requestOverhead(w http.ResponseWriter, req *http.Request, method string) bool {
	if req.Method != method {
		errorJson(w, http.StatusMethodNotAllowed, "")
		return false
	}

	return true
}

// All structs representing incoming json request body should implement this
// The contents of `error` should be safe for an API response (public-facing)
type PostRequest interface {
	validate() error
}

// TODO decoder.DisallowUnknownFields?
// TODO GET params too large (like StatusRequestEntityTooLarge)? Or is that
//   somehow handled by the http library due to a size limit in the http spec?

// Confirm it's a Post request, various overhead, decode the json, validate the struct
func getPostData(w http.ResponseWriter, req *http.Request, reqStruct PostRequest) bool {
	if !requestOverhead(w, req, http.MethodPost) {
		return false
	}

	// Make the limit 100k. Increase from there as needed. I'd rather block some
	// people's large wallets and increase the limit than OOM for everybody and
	// decrease the limit.
	req.Body = http.MaxBytesReader(w, req.Body, 100000)
	err := json.NewDecoder(req.Body).Decode(&reqStruct)
	switch {
	case err == nil:
		break
	case err.Error() == "http: request body too large":
		errorJson(w, http.StatusRequestEntityTooLarge, "")
		return false
	default:
		// Maybe we can suss out specific errors later. Need to study what errors
		// come from Decode.
		errorJson(w, http.StatusBadRequest, "Error parsing JSON")
		return false
	}

	err = reqStruct.validate()
	if err != nil {
		errorJson(w, http.StatusBadRequest, "Request failed validation: "+err.Error())
		return false
	}

	return true
}

// Confirm it's a Get request, various overhead
func getGetData(w http.ResponseWriter, req *http.Request) bool {
	return requestOverhead(w, req, http.MethodGet)
}

// TODO - probably don't return all of authToken since we only need userId and
// deviceId. Also this is apparently not idiomatic go error handling.
func (s *Server) checkAuth(
	w http.ResponseWriter,
	token auth.TokenString,
	scope auth.AuthScope,
) *auth.AuthToken {
	authToken, err := s.store.GetToken(token)
	if err == store.ErrNoToken {
		errorJson(w, http.StatusUnauthorized, "Token Not Found")
		return nil
	}
	if err != nil {
		internalServiceErrorJson(w, err, "Error getting Token")
		return nil
	}

	if !authToken.ScopeValid(scope) {
		errorJson(w, http.StatusForbidden, "Scope")
		return nil
	}

	return authToken
}

func validateEmail(email auth.Email) bool {
	e, err := mail.ParseAddress(string(email))
	if err != nil {
		return false
	}
	// "Joe <joe@example.com>" is valid according to ParseAddress. Likewise
	// " joe@example.com". Etc. We only want the exact address, "joe@example.com"
	// to be valid. ParseAddress will extract the exact address as e.Address. So
	// we'll take the input email, put it through ParseAddress, see if it parses
	// successfully, and then compare the input email to e.Address to make sure
	// that it was an exact address to begin with.
	return string(email) == e.Address
}

func validateClientSaltSeed(clientSaltSeed auth.ClientSaltSeed) bool {
	_, err := hex.DecodeString(string(clientSaltSeed))
	const seedHexLength = auth.ClientSaltSeedLength * 2
	return len(clientSaltSeed) == seedHexLength && err == nil
}

// TODO - both wallet and token requests should be PUT, not POST.
// PUT = "...creates a new resource or replaces a representation of the target resource with the request payload."

func (s *Server) unknownEndpoint(w http.ResponseWriter, req *http.Request) {
	errorJson(w, http.StatusNotFound, "Unknown Endpoint")
	return
}

func (s *Server) wrongApiVersion(w http.ResponseWriter, req *http.Request) {
	errorJson(w, http.StatusNotFound, "Wrong API version. Current version is "+ApiVersion+".")
	return
}

func (s *Server) Serve() {
	http.HandleFunc(PathAuthToken, s.getAuthToken)
	http.HandleFunc(PathWallet, s.handleWallet)
	http.HandleFunc(PathRegister, s.register)
	http.HandleFunc(PathPassword, s.changePassword)
	http.HandleFunc(PathClientSaltSeed, s.getClientSaltSeed)

	http.HandleFunc(PathUnknownEndpoint, s.unknownEndpoint)
	http.HandleFunc(PathWrongApiVersion, s.wrongApiVersion)

	http.Handle(PathPrometheus, promhttp.Handler())

	fmt.Println("Serving at localhost:8090")
	http.ListenAndServe("localhost:8090", nil)
}
