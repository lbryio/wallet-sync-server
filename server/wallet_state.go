package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/store"
	"orblivion/lbry-id/wallet"
)

type WalletStateRequest struct {
	Token           auth.AuthTokenString   `json:"token"`
	WalletStateJson string                 `json:"walletStateJson"`
	Hmac            wallet.WalletStateHmac `json:"hmac"`
}

func (r *WalletStateRequest) validate() bool {
	return (r.Token != auth.AuthTokenString("") &&
		r.WalletStateJson != "" &&
		r.Hmac != wallet.WalletStateHmac(""))
}

type WalletStateResponse struct {
	WalletStateJson string                 `json:"walletStateJson"`
	Hmac            wallet.WalletStateHmac `json:"hmac"`
	Error           string                 `json:"error"` // in case of 409 Conflict responses. TODO - make field not show up if it's empty, to avoid confusion
}

func (s *Server) handleWalletState(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		s.getWalletState(w, req)
	} else if req.Method == http.MethodPost {
		s.postWalletState(w, req)
	} else {
		errorJson(w, http.StatusMethodNotAllowed, "")
	}
}

// TODO - There's probably a struct-based solution here like with POST/PUT.
// We could put that struct up top as well.
func getWalletStateParams(req *http.Request) (token auth.AuthTokenString, err error) {
	tokenSlice, hasTokenSlice := req.URL.Query()["token"]

	if !hasTokenSlice {
		err = fmt.Errorf("Missing token parameter")
	}

	if err == nil {
		token = auth.AuthTokenString(tokenSlice[0])
	}

	return
}

func (s *Server) getWalletState(w http.ResponseWriter, req *http.Request) {
	if !getGetData(w, req) {
		return
	}

	token, paramsErr := getWalletStateParams(req)

	if paramsErr != nil {
		// In this specific case, the error is limited to values that are safe to
		// give to the user.
		errorJson(w, http.StatusBadRequest, paramsErr.Error())
		return
	}

	authToken := s.checkAuth(w, token, auth.ScopeGetWalletState)

	if authToken == nil {
		return
	}

	latestWalletStateJson, latestHmac, err := s.store.GetWalletState(authToken.UserId)

	var response []byte

	if err == store.ErrNoWalletState {
		errorJson(w, http.StatusNotFound, "No wallet state")
		return
	} else if err != nil {
		internalServiceErrorJson(w, err, "Error retrieving walletState")
		return
	}

	walletStateResponse := WalletStateResponse{
		WalletStateJson: latestWalletStateJson,
		Hmac:            latestHmac,
	}
	response, err = json.Marshal(walletStateResponse)

	if err != nil {
		internalServiceErrorJson(w, err, "Error generating latestWalletState response")
		return
	}

	fmt.Fprintf(w, string(response))
}

func (s *Server) postWalletState(w http.ResponseWriter, req *http.Request) {
	var walletStateRequest WalletStateRequest
	if !getPostData(w, req, &walletStateRequest) {
		return
	}

	var walletStateMetadata wallet.WalletStateMetadata
	if err := json.Unmarshal([]byte(walletStateRequest.WalletStateJson), &walletStateMetadata); err != nil {
		errorJson(w, http.StatusBadRequest, "Malformed walletStateJson")
		return
	}

	if s.walletUtil.ValidateWalletStateMetadata(&walletStateMetadata) {
		// TODO
	}

	authToken := s.checkAuth(w, walletStateRequest.Token, auth.ScopeFull)
	if authToken == nil {
		return
	}

	// TODO - We could do an extra check - pull from db, make sure the new
	// walletStateMetadata doesn't regress lastSynced for any given device.
	// This is primarily the responsibility of the clients, but we may want to
	// trade a db call here for a double-check against bugs in the client.
	// We do already do some validation checks here, but those doesn't require
	// new database calls.

	latestWalletStateJson, latestHmac, updated, err := s.store.SetWalletState(
		authToken.UserId,
		walletStateRequest.WalletStateJson,
		walletStateMetadata.Sequence(),
		walletStateRequest.Hmac,
	)

	var response []byte

	if err == store.ErrNoWalletState {
		// We failed to update, and when we tried pulling the latest wallet state,
		// there was nothing there. This should only happen if the client sets
		// sequence != 1 for the first walletState, which would be a bug.
		// TODO - figure out better error messages and/or document this
		errorJson(w, http.StatusConflict, "Bad sequence number (No existing wallet state)")
		return
	} else if err != nil {
		// Something other than sequence error
		internalServiceErrorJson(w, err, "Error saving walletState")
		return
	}

	walletStateResponse := WalletStateResponse{
		WalletStateJson: latestWalletStateJson,
		Hmac:            latestHmac,
	}
	if !updated {
		// TODO - should we even call this an error?
		walletStateResponse.Error = "Bad sequence number"
	}
	response, err = json.Marshal(walletStateResponse)

	if err != nil {
		internalServiceErrorJson(w, err, "Error generating walletStateResponse")
		return
	}

	// Response Code:
	//   200: Update successful
	//   409: Update unsuccessful, probably due to new walletState's
	//      sequence not being 1 + current walletState's sequence
	//
	// Response Body:
	//   Current walletState (if it exists). If update successful, we just return
	//   the same one passed in. If update not successful, return the latest one
	//   from the db for the client to merge.
	if updated {
		fmt.Fprintf(w, string(response))
	} else {
		http.Error(w, string(response), http.StatusConflict)
	}
}
