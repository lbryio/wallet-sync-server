package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"lbryio/wallet-sync-server/auth"
	"lbryio/wallet-sync-server/store"
	"lbryio/wallet-sync-server/wallet"
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
	if changePasswordRequest.EncryptedWallet != "" {
		err = s.store.ChangePasswordWithWallet(
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
		err = s.store.ChangePasswordNoWallet(
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

	var changePasswordResponse struct{} // no data to respond with, but keep it JSON
	var response []byte
	response, err = json.Marshal(changePasswordResponse)

	if err != nil {
		internalServiceErrorJson(w, err, "Error generating change password response")
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(response))
}
