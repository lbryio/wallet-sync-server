package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"lbryio/wallet-sync-server/auth"
	"lbryio/wallet-sync-server/metrics"
	"lbryio/wallet-sync-server/store"
	"lbryio/wallet-sync-server/wallet"
)

type WalletRequest struct {
	Token           auth.AuthTokenString   `json:"token"`
	EncryptedWallet wallet.EncryptedWallet `json:"encryptedWallet"`
	Sequence        wallet.Sequence        `json:"sequence"`
	Hmac            wallet.WalletHmac      `json:"hmac"`
}

func (r *WalletRequest) validate() error {
	if r.Token == "" {
		return fmt.Errorf("Missing 'token'")
	}
	if r.EncryptedWallet == "" {
		return fmt.Errorf("Missing 'encryptedWallet'")
	}
	if r.Hmac == "" {
		return fmt.Errorf("Missing 'hmac'")
	}
	if r.Sequence < store.InitialWalletSequence {
		return fmt.Errorf("Missing or zero-value 'sequence'")
	}
	return nil
}

type WalletResponse struct {
	EncryptedWallet wallet.EncryptedWallet `json:"encryptedWallet"`
	Sequence        wallet.Sequence        `json:"sequence"`
	Hmac            wallet.WalletHmac      `json:"hmac"`
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

func (s *Server) getWallet(w http.ResponseWriter, req *http.Request) {
	metrics.RequestsCount.With(prometheus.Labels{"method": "GET", "endpoint": "wallet"}).Inc()

	if !getGetData(w, req) {
		return
	}

	token, paramsErr := getTokenParam(req)

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

// Response Code:
//   200: Update successful
//   409: Update unsuccessful due to new wallet's sequence not being 1 +
//     current wallet's sequence
//   500: Update unsuccessful for unanticipated reasons
func (s *Server) postWallet(w http.ResponseWriter, req *http.Request) {
	metrics.RequestsCount.With(prometheus.Labels{"method": "POST", "endpoint": "wallet"}).Inc()

	var walletRequest WalletRequest
	if !getPostData(w, req, &walletRequest) {
		return
	}

	authToken := s.checkAuth(w, walletRequest.Token, auth.ScopeFull)
	if authToken == nil {
		return
	}

	err := s.store.SetWallet(authToken.UserId, walletRequest.EncryptedWallet, walletRequest.Sequence, walletRequest.Hmac)

	if err == store.ErrWrongSequence {
		errorJson(w, http.StatusConflict, "Bad sequence number")
		return
	} else if err != nil {
		// Something other than sequence error
		internalServiceErrorJson(w, err, "Error saving or getting wallet")
		return
	}

	var response []byte
	var walletResponse struct{} // no data to respond with, but keep it JSON
	response, err = json.Marshal(walletResponse)

	if err != nil {
		internalServiceErrorJson(w, err, "Error generating walletResponse")
		return
	}

	fmt.Fprintf(w, string(response))
	if walletRequest.Sequence == store.InitialWalletSequence {
		log.Printf("Initial wallet created for user id %d", authToken.UserId)
	}

	// Inform the other clients over websockets. If we can't do it within 100
	// milliseconds, don't bother. It's a nice-to-have, not mission critical.
	// But, count the misses on the dashboard. If it happens a lot we should
	// probably increase the buffer on the notify chans for the clients. Those
	// will be a bottleneck within the socket manager.
	timeout := time.NewTicker(100 * time.Millisecond)
	select {
	case s.walletUpdates <- walletUpdateMsg{authToken.UserId, walletRequest.Sequence}:
	case <-timeout.C:
		metrics.ErrorsCount.With(prometheus.Labels{"error_type": "ws-client-notify"}).Inc()
	}
	timeout.Stop()
}
