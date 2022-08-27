package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"lbryio/wallet-sync-server/auth"
	"lbryio/wallet-sync-server/metrics"
	"lbryio/wallet-sync-server/store"
	"lbryio/wallet-sync-server/wallet"

	"github.com/prometheus/client_golang/prometheus"
)

type ChangePasswordRequest struct {
	EncryptedWallet wallet.EncryptedWallet `json:"encryptedWallet"`
	Sequence        wallet.Sequence        `json:"sequence"`
	Hmac            wallet.WalletHmac      `json:"hmac"`
	Email           auth.Email             `json:"email"`
	OldPassword     auth.Password          `json:"oldPassword"`
	NewPassword     auth.Password          `json:"newPassword"`
	ClientSaltSeed  auth.ClientSaltSeed    `json:"clientSaltSeed"`
}

func (r *ChangePasswordRequest) validate() error {
	// The wallet should be here or not. Not partially here.
	walletPresent := (r.EncryptedWallet != "" && r.Hmac != "" && r.Sequence > 0)
	walletAbsent := (r.EncryptedWallet == "" && r.Hmac == "" && r.Sequence == 0)

	if !r.Email.Validate() {
		return fmt.Errorf("Invalid or missing 'email'")
	}
	if !r.OldPassword.Validate() {
		return fmt.Errorf("Invalid or missing 'oldPassword'")
	}
	if !r.NewPassword.Validate() {
		return fmt.Errorf("Invalid or missing 'newPassword'")
	}
	// Too bad we can't do this so easily with clientSaltSeed
	if r.OldPassword == r.NewPassword {
		return fmt.Errorf("'oldPassword' and 'newPassword' should not be the same")
	}
	if !r.ClientSaltSeed.Validate() {
		return fmt.Errorf("Invalid or missing 'clientSaltSeed'")
	}
	if !walletPresent && !walletAbsent {
		return fmt.Errorf("Fields 'encryptedWallet', 'sequence', and 'hmac' should be all non-empty and non-zero, or all omitted")
	}
	return nil
}

func (s *Server) changePassword(w http.ResponseWriter, req *http.Request) {
	var changePasswordRequest ChangePasswordRequest
	if !getPostData(w, req, &changePasswordRequest) {
		return
	}

	// To be cautious, we will block password changes for unverified accounts.
	// The only reason I can think of for allowing them is if the user
	// accidentally put in a bad password that they desperately want to change,
	// and the verification email isn't working. However unlikely such a scenario
	// is, with the salting and the KDF and all that, it seems all the less a big
	// deal.
	//
	// Changing a password when unverified as such isn't a big deal, but I'm
	// concerned with wallet creation. This endpoint currently doesn't allow you
	// to _create_ a wallet if you don't already have one, so as of now we don't
	// strictly need this restriction. However this seems too precarious and
	// tricky. We might forget about it and allow wallet creation here later.
	// Someone might find a loophole I'm not thinking of. So I'm just blocking
	// unverified accounts here for simplicity.

	var err error
	var userId auth.UserId
	if changePasswordRequest.EncryptedWallet != "" {
		userId, err = s.store.ChangePasswordWithWallet(
			changePasswordRequest.Email,
			changePasswordRequest.OldPassword,
			changePasswordRequest.NewPassword,
			changePasswordRequest.ClientSaltSeed,
			changePasswordRequest.EncryptedWallet,
			changePasswordRequest.Sequence,
			changePasswordRequest.Hmac)
		if err == store.ErrWrongSequence {
			errorJson(w, http.StatusConflict, "Bad sequence number or wallet does not exist")
			return
		}
	} else {
		userId, err = s.store.ChangePasswordNoWallet(
			changePasswordRequest.Email,
			changePasswordRequest.OldPassword,
			changePasswordRequest.NewPassword,
			changePasswordRequest.ClientSaltSeed,
		)
		if err == store.ErrUnexpectedWallet {
			errorJson(w, http.StatusConflict, "Wallet exists; need an updated wallet when changing password")
			return
		}
	}
	if err == store.ErrWrongCredentials {
		errorJson(w, http.StatusUnauthorized, "No match for email and/or password")
		return
	}
	if err == store.ErrNotVerified {
		errorJson(w, http.StatusUnauthorized, "Account is not verified")
		return
	}
	if err != nil {
		internalServiceErrorJson(w, err, "Error changing password")
		return
	}

	// TODO - A socket connection request using an old auth token could still
	// succeed in a race condition:
	// * websocket handler: checkAuth passes with token
	// * password change handler: change password, invalidate token
	// * password change handler: send userRemove message
	// * websocket manager: process userRemove message, ending all websocket connections for user
	// * websocket handler: new websocket connection is established
	//
	// It would require the websocket handler to be very slow, but I don't want to
	// rule it out.
	//
	// But a much more likely scenario could happen: the buffer on the userRemove
	// channel could get full and it could time out, and not boot any of the
	// users' clients.
	//
	// These aren't horribly important now since the only message is a
	// notification that a new wallet version exists, but who knows what we
	// could use websockets for. Maybe we start doing something crazy like
	// updating the wallet over the channel, in which case we absolutely want
	// to prevent an old client from doing so after a password change on
	// another client.
	//
	// We'd have to think a fair amount about how to make these foolproof if it
	// becomes important. Maybe we just pass the auth token to the websocket
	// writer, and pass it to every wallet update db call, and have it check
	// the auth token within the same transaction as the wallet update.

	timeout := time.NewTicker(100 * time.Millisecond)
	select {
	case s.userRemove <- wsClientForUser{userId, nil}:
	case <-timeout.C:
		metrics.ErrorsCount.With(prometheus.Labels{"details": "websocket user remove chan buffer full"}).Inc()
		return
	}
	timeout.Stop()

	var changePasswordResponse struct{} // no data to respond with, but keep it JSON
	var response []byte
	response, err = json.Marshal(changePasswordResponse)

	if err != nil {
		internalServiceErrorJson(w, err, "Error generating change password response")
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(response))
	log.Printf("User %s has changed their password", changePasswordRequest.Email)
}
