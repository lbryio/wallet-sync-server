package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"lbryio/wallet-sync-server/auth"
	"lbryio/wallet-sync-server/env"
	"lbryio/wallet-sync-server/mail"
	"lbryio/wallet-sync-server/server/paths"
	"lbryio/wallet-sync-server/store"
	"lbryio/wallet-sync-server/wallet"
)

const maxBodySize = 100000

// Message sent from the wallet POST request handler to the websocket manager,
// indicating that a user's client should receive a (different) message that
// their wallet has an update on the server.
type walletUpdateMsg struct {
	userId   auth.UserId
	sequence wallet.Sequence
}

type Server struct {
	auth  auth.AuthInterface
	store store.StoreInterface
	env   env.EnvInterface
	mail  mail.MailInterface
	port  int

	clientAdd     chan wsClientForUser
	clientRemove  chan wsClientForUser
	userRemove    chan wsClientForUser
	walletUpdates chan walletUpdateMsg
}

func Init(
	authInterface auth.AuthInterface,
	storeInterface store.StoreInterface,
	envInterface env.EnvInterface,
	mailInterface mail.MailInterface,
	port int,
) *Server {
	return &Server{
		auth:  authInterface,
		store: storeInterface,
		env:   envInterface,
		mail:  mailInterface,
		port:  port,

		// Anything that could get backed up by a lot of requests, let's just
		// give it a buffer. Starting small until we start to see dashboard
		// stats on this. I want a sense of how this grows with the number of
		// users or whatnot.
		clientAdd:     make(chan wsClientForUser),
		clientRemove:  make(chan wsClientForUser),
		userRemove:    make(chan wsClientForUser, 5),
		walletUpdates: make(chan walletUpdateMsg, 5),
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

// Confirm it's a Post request, various overhead, decode the json, validate the struct
func getPostData(w http.ResponseWriter, req *http.Request, reqStruct PostRequest) bool {
	if !requestOverhead(w, req, http.MethodPost) {
		return false
	}

	// Make the limit 100k. Increase from there as needed. I'd rather block some
	// people's large wallets and increase the limit than OOM for everybody and
	// decrease the limit.
	req.Body = http.MaxBytesReader(w, req.Body, maxBodySize)
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&reqStruct)
	switch {
	case err == nil:
		break
	case err.Error() == "http: request body too large":
		errorJson(w, http.StatusRequestEntityTooLarge, "")
		return false
	case strings.HasPrefix(err.Error(), "json: unknown field"):
		// The error is coming straight out of the json decoder. I think the prefix
		// we check for determines what it is pretty reliably. I'd think it's safe
		// to give back to the requesting client (unlike an arbitrary error
		// message).
		errorJson(w, http.StatusBadRequest, err.Error())
		return false
	default:
		// Maybe we can suss out more specific errors later. Need to study what
		// errors come from Decode.
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
// deviceId.
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

// Useful for any request where token is the only GET param to get
// TODO - There's probably a struct-based solution here like with POST/PUT.
func getTokenParam(req *http.Request) (token auth.AuthTokenString, err error) {
	tokenSlice, hasTokenSlice := req.URL.Query()["token"]

	if !hasTokenSlice || tokenSlice[0] == "" {
		err = fmt.Errorf("Missing token parameter")
	}

	if err == nil {
		token = auth.AuthTokenString(tokenSlice[0])
	}

	return
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

func serve(server *http.Server, done chan bool) {
	log.Print("Server start")
	server.ListenAndServe()
	log.Print("Server finish")

	done <- true
}

func (s *Server) Serve() {
	http.HandleFunc(paths.PathAuthToken, s.getAuthToken)
	http.HandleFunc(paths.PathWallet, s.handleWallet)
	http.HandleFunc(paths.PathRegister, s.register)
	http.HandleFunc(paths.PathPassword, s.changePassword)
	http.HandleFunc(paths.PathVerify, s.verify)
	http.HandleFunc(paths.PathResendVerify, s.resendVerifyEmail)
	http.HandleFunc(paths.PathClientSaltSeed, s.getClientSaltSeed)
	http.HandleFunc(paths.PathWebsocket, s.websocket)

	http.HandleFunc(paths.PathUnknownEndpoint, s.unknownEndpoint)
	http.HandleFunc(paths.PathWrongApiVersion, s.wrongApiVersion)

	http.Handle(paths.PathPrometheus, promhttp.Handler())

	log.Printf("Serving at localhost:%d\n", s.port)

	// Signal *to* socket manager that it should finish (we use server.Shutdown
	// to tell the server to finish)
	socketsFinish := make(chan bool)

	// Signal *from* server and socket manager that they are done:
	serverDone := make(chan bool)
	socketsDone := make(chan bool)

	go s.manageSockets(socketsDone, socketsFinish)

	server := http.Server{Addr: fmt.Sprintf("localhost:%d", s.port)}
	go serve(&server, serverDone)

	// Make sure that both the server and the websocket manager close properly on interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait for the interrupt signal
	<-interrupt

	// Tell the server to finish and wait for it to do so. We want it to finish
	// to guarantee no more incoming sockets before we turn off the socket
	// manager.
	server.Shutdown(context.Background())
	<-serverDone

	// The socket manager's cleanup procedure assumes that there will be no new
	// socket connections. Now that the server is done, no new socket
	// connections will be coming in, so we can close the socket manager.
	socketsFinish <- true
	<-socketsDone

	log.Printf("All done")
}
