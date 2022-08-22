package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"lbryio/wallet-sync-server/auth"
	"lbryio/wallet-sync-server/env"
	"lbryio/wallet-sync-server/mail"
	"lbryio/wallet-sync-server/server/paths"
	"lbryio/wallet-sync-server/store"
)

const maxBodySize = 100000

type Server struct {
	auth  auth.AuthInterface
	store store.StoreInterface
	env   env.EnvInterface
	mail  mail.MailInterface
	port  int
}

// TODO If I capitalize the `auth` `store` and `env` fields of Store{} I can
// create Store{} structs directly from main.go.
func Init(
	auth auth.AuthInterface,
	store store.StoreInterface,
	env env.EnvInterface,
	mail mail.MailInterface,
	port int,
) *Server {
	return &Server{auth, store, env, mail, port}
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
	req.Body = http.MaxBytesReader(w, req.Body, maxBodySize)
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
	token auth.AuthTokenString,
	scope auth.AuthScope,
) *auth.AuthToken {
	authToken, err := s.store.GetToken(token)
	if err == store.ErrNoTokenForUserDevice {
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

// TODO - both wallet and token requests should be PUT, not POST.
// PUT = "...creates a new resource or replaces a representation of the target resource with the request payload."

func (s *Server) unknownEndpoint(w http.ResponseWriter, req *http.Request) {
	errorJson(w, http.StatusNotFound, "Unknown Endpoint")
	return
}

func (s *Server) wrongApiVersion(w http.ResponseWriter, req *http.Request) {
	errorJson(w, http.StatusNotFound, "Wrong API version. Current version is "+paths.ApiVersion+".")
	return
}

func (s *Server) Serve() {
	http.HandleFunc(paths.PathAuthToken, s.getAuthToken)
	http.HandleFunc(paths.PathWallet, s.handleWallet)
	http.HandleFunc(paths.PathRegister, s.register)
	http.HandleFunc(paths.PathPassword, s.changePassword)
	http.HandleFunc(paths.PathVerify, s.verify)
	http.HandleFunc(paths.PathResendVerify, s.resendVerifyEmail)
	http.HandleFunc(paths.PathClientSaltSeed, s.getClientSaltSeed)

	http.HandleFunc(paths.PathUnknownEndpoint, s.unknownEndpoint)
	http.HandleFunc(paths.PathWrongApiVersion, s.wrongApiVersion)

	http.Handle(paths.PathPrometheus, promhttp.Handler())

	log.Printf("Serving at localhost:%d\n", s.port)
	http.ListenAndServe(fmt.Sprintf("localhost:%d", s.port), nil)
}
