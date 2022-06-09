package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/store"
	"orblivion/lbry-id/wallet"
)

const CURRENT_VERSION = 1

type WalletRequest struct {
	Version         int                    `json:"version"`
	Token           auth.TokenString       `json:"token"`
	EncryptedWallet wallet.EncryptedWallet `json:"encryptedWallet"`
	Sequence        wallet.Sequence        `json:"sequence"`
	Hmac            wallet.WalletHmac      `json:"hmac"`
}

func (r *WalletRequest) validate() bool {
	return (r.Version == CURRENT_VERSION &&
		r.Token != auth.TokenString("") &&
		r.EncryptedWallet != wallet.EncryptedWallet("") &&
		r.Hmac != wallet.WalletHmac("") &&
		r.Sequence >= wallet.Sequence(1))
}

type WalletResponse struct {
	Version         int                    `json:"version"`
	EncryptedWallet wallet.EncryptedWallet `json:"encryptedWallet"`
	Sequence        wallet.Sequence        `json:"sequence"`
	Hmac            wallet.WalletHmac      `json:"hmac"`
	Error           string                 `json:"error"` // in case of 409 Conflict responses. TODO - make field not show up if it's empty, to avoid confusion
}

func (s *Server) handleWallet(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		s.getWallet(w, req)
	} else if req.Method == http.MethodPost {
		s.postWallet(w, req)
	} else {
		errorJson(w, http.StatusMethodNotAllowed, "")
	}
}

// TODO - There's probably a struct-based solution here like with POST/PUT.
// We could put that struct up top as well.
func getWalletParams(req *http.Request) (token auth.TokenString, err error) {
	tokenSlice, hasTokenSlice := req.URL.Query()["token"]

	if !hasTokenSlice {
		err = fmt.Errorf("Missing token parameter")
	}

	if err == nil {
		token = auth.TokenString(tokenSlice[0])
	}

	return
}

func (s *Server) getWallet(w http.ResponseWriter, req *http.Request) {
	if !getGetData(w, req) {
		return
	}

	token, paramsErr := getWalletParams(req)

	if paramsErr != nil {
		// In this specific case, the error is limited to values that are safe to
		// give to the user.
		errorJson(w, http.StatusBadRequest, paramsErr.Error())
		return
	}

	authToken := s.checkAuth(w, token, auth.ScopeFull)

	if authToken == nil {
		return
	}

	latestEncryptedWallet, latestSequence, latestHmac, err := s.store.GetWallet(authToken.UserId)

	if err == store.ErrNoWallet {
		errorJson(w, http.StatusNotFound, "No wallet")
		return
	} else if err != nil {
		internalServiceErrorJson(w, err, "Error retrieving wallet")
		return
	}

	walletResponse := WalletResponse{
		Version:         CURRENT_VERSION,
		EncryptedWallet: latestEncryptedWallet,
		Sequence:        latestSequence,
		Hmac:            latestHmac,
	}

	var response []byte
	response, err = json.Marshal(walletResponse)

	if err != nil {
		internalServiceErrorJson(w, err, "Error generating wallet response")
		return
	}

	fmt.Fprintf(w, string(response))
}

func (s *Server) postWallet(w http.ResponseWriter, req *http.Request) {
	var walletRequest WalletRequest
	if !getPostData(w, req, &walletRequest) {
		return
	}

	authToken := s.checkAuth(w, walletRequest.Token, auth.ScopeFull)
	if authToken == nil {
		return
	}

	latestEncryptedWallet, latestSequence, latestHmac, sequenceCorrect, err := s.store.SetWallet(
		authToken.UserId,
		walletRequest.EncryptedWallet,
		walletRequest.Sequence,
		walletRequest.Hmac,
	)

	var response []byte

	if err == store.ErrNoWallet {
		// We failed to update, and when we tried pulling the latest wallet,
		// there was nothing there. This should only happen if the client sets
		// sequence != 1 for the first wallet, which would be a bug.
		// TODO - figure out better error messages and/or document this
		errorJson(w, http.StatusConflict, "Bad sequence number (No existing wallet)")
		return
	} else if err != nil {
		// Something other than sequence error
		internalServiceErrorJson(w, err, "Error saving wallet")
		return
	}

	walletResponse := WalletResponse{
		Version:         CURRENT_VERSION,
		EncryptedWallet: latestEncryptedWallet,
		Sequence:        latestSequence,
		Hmac:            latestHmac,
	}

	if !sequenceCorrect {
		// TODO - should we even call this an error?
		walletResponse.Error = "Bad sequence number"
	}
	response, err = json.Marshal(walletResponse)

	if err != nil {
		internalServiceErrorJson(w, err, "Error generating walletResponse")
		return
	}

	// Response Code:
	//   200: Update successful
	//   409: Update unsuccessful, probably due to new wallet's
	//      sequence not being 1 + current wallet's sequence
	//
	// Response Body:
	//   Current wallet (if it exists). If update successful, we just return
	//   the same one passed in. If update not successful, return the latest one
	//   from the db for the client to merge.
	if sequenceCorrect {
		fmt.Fprintf(w, string(response))
	} else {
		http.Error(w, string(response), http.StatusConflict)
	}
}
