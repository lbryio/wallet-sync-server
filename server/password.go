package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/store"
	"orblivion/lbry-id/wallet"
)

type ChangePasswordRequest struct {
	EncryptedWallet wallet.EncryptedWallet `json:"encryptedWallet"`
	Sequence        wallet.Sequence        `json:"sequence"`
	Hmac            wallet.WalletHmac      `json:"hmac"`
	Email           auth.Email             `json:"email"`
	OldPassword     auth.Password          `json:"oldPassword"`
	NewPassword     auth.Password          `json:"newPassword"`
}

func (r *ChangePasswordRequest) validate() error {
	// The wallet should be here or not. Not partially here.
	walletPresent := (r.EncryptedWallet != "" && r.Hmac != "" && r.Sequence > 0)
	walletAbsent := (r.EncryptedWallet == "" && r.Hmac == "" && r.Sequence == 0)

	if !validateEmail(r.Email) {
		return fmt.Errorf("Invalid 'email'")
	}
	if r.OldPassword == "" {
		return fmt.Errorf("Missing 'oldPassword'")
	}
	if r.NewPassword == "" {
		return fmt.Errorf("Missing 'newPassword'")
	}
	if r.OldPassword == r.NewPassword {
		return fmt.Errorf("'oldPassword' and 'newPassword' should not be the same")
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

	var err error
	if changePasswordRequest.EncryptedWallet != "" {
		err = s.store.ChangePasswordWithWallet(
			changePasswordRequest.Email,
			changePasswordRequest.OldPassword,
			changePasswordRequest.NewPassword,
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
		)
		if err == store.ErrUnexpectedWallet {
			errorJson(w, http.StatusConflict, "Wallet exists; need an updated wallet when changing password")
			return
		}
	}
	if err == store.ErrWrongCredentials {
		errorJson(w, http.StatusUnauthorized, "No match for email and password")
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
