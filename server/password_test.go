package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lbryio/lbry-id/auth"
	"lbryio/lbry-id/store"
	"lbryio/lbry-id/wallet"
)

func TestServerChangePassword(t *testing.T) {
	tt := []struct {
		name string

		expectedStatusCode  int
		expectedErrorString string

		// Whether we expect the call to ChangePassword*Wallet to happen
		expectChangePasswordCall bool

		// `new...` refers to what is being passed into the via POST request (and
		//   what we expect to get passed into SetWallet for the *non-error* cases
		//   below)
		newEncryptedWallet wallet.EncryptedWallet
		newSequence        wallet.Sequence
		newHmac            wallet.WalletHmac

		email auth.Email

		storeErrors TestStoreFunctionsErrors
	}{
		{
			name: "success with wallet",

			expectedStatusCode: http.StatusOK,

			expectChangePasswordCall: true,

			newEncryptedWallet: "my-enc-wallet",
			newSequence:        2,
			newHmac:            "my-hmac",

			email: "abc@example.com",
		}, {
			name: "success no wallet",

			expectedStatusCode: http.StatusOK,

			expectChangePasswordCall: true,

			email: "abc@example.com",
		}, {
			name:                "conflict with wallet",
			expectedStatusCode:  http.StatusConflict,
			expectedErrorString: http.StatusText(http.StatusConflict) + ": Bad sequence number or wallet does not exist",

			expectChangePasswordCall: true,

			newEncryptedWallet: "my-enc-wallet",
			newSequence:        2,
			newHmac:            "my-hmac",

			email: "abc@example.com",

			storeErrors: TestStoreFunctionsErrors{ChangePasswordWithWallet: store.ErrWrongSequence},
		}, {
			name:                "conflict no wallet",
			expectedStatusCode:  http.StatusConflict,
			expectedErrorString: http.StatusText(http.StatusConflict) + ": Wallet exists; need an updated wallet when changing password",

			expectChangePasswordCall: true,

			email: "abc@example.com",

			storeErrors: TestStoreFunctionsErrors{ChangePasswordNoWallet: store.ErrUnexpectedWallet},
		}, {
			name:                "incorrect email with wallet",
			expectedStatusCode:  http.StatusUnauthorized,
			expectedErrorString: http.StatusText(http.StatusUnauthorized) + ": No match for email and password",

			expectChangePasswordCall: true,

			newEncryptedWallet: "my-enc-wallet",
			newSequence:        2,
			newHmac:            "my-hmac",

			email: "abc@example.com",

			storeErrors: TestStoreFunctionsErrors{ChangePasswordWithWallet: store.ErrWrongCredentials},
		}, {
			name:                "incorrect email no wallet",
			expectedStatusCode:  http.StatusUnauthorized,
			expectedErrorString: http.StatusText(http.StatusUnauthorized) + ": No match for email and password",

			expectChangePasswordCall: true,

			email: "abc@example.com",

			storeErrors: TestStoreFunctionsErrors{ChangePasswordNoWallet: store.ErrWrongCredentials},
		}, {
			name:                "validation error",
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrorString: http.StatusText(http.StatusBadRequest) + ": Request failed validation: Invalid or missing 'email'",

			// Just check one validation error (missing email address) to make sure
			// the validate function is called. We'll check the rest of the
			// validation errors in the other test below.

			expectChangePasswordCall: false,
		}, {
			name:                     "db error changing password with wallet",
			expectedStatusCode:       http.StatusInternalServerError,
			expectedErrorString:      http.StatusText(http.StatusInternalServerError),
			expectChangePasswordCall: true,

			// Putting in valid data here so it's clear that this isn't what causes
			// the error
			newEncryptedWallet: "my-encrypted-wallet",
			newSequence:        2,
			newHmac:            "my-hmac",

			email: "abc@example.com",

			// What causes the error
			storeErrors: TestStoreFunctionsErrors{ChangePasswordWithWallet: fmt.Errorf("Some random db problem")},
		}, {
			name:                     "db error changing password no wallet",
			expectedStatusCode:       http.StatusInternalServerError,
			expectedErrorString:      http.StatusText(http.StatusInternalServerError),
			expectChangePasswordCall: true,

			email: "abc@example.com",

			// What causes the error
			storeErrors: TestStoreFunctionsErrors{ChangePasswordNoWallet: fmt.Errorf("Some random db problem")},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			testAuth := TestAuth{}
			testStore := TestStore{Errors: tc.storeErrors}
			s := Server{&testAuth, &testStore}

			// Whether we passed in wallet fields (these test cases should be passing
			// in all of them or none of them, so we only test EncryptedWallet). This
			// determines whether we expect a call to ChangePasswordWithWallet (as
			// opposed to ChangePasswordNoWallet).
			withWallet := (tc.newEncryptedWallet != "")

			const oldPassword = "old password"
			const newPassword = "new password"
			const clientSaltSeed = "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234"

			requestBody := []byte(
				fmt.Sprintf(`{
          "encryptedWallet": "%s",
          "sequence":         %d ,
          "hmac":            "%s",
          "email":           "%s",
          "oldPassword":     "%s",
          "newPassword":     "%s",
          "clientSaltSeed":  "%s"
        }`, tc.newEncryptedWallet, tc.newSequence, tc.newHmac, tc.email, oldPassword, newPassword, clientSaltSeed),
			)

			req := httptest.NewRequest(http.MethodPost, PathPassword, bytes.NewBuffer(requestBody))
			w := httptest.NewRecorder()

			s.changePassword(w, req)

			body, _ := ioutil.ReadAll(w.Body)

			expectStatusCode(t, w, tc.expectedStatusCode)
			expectErrorString(t, body, tc.expectedErrorString)

			if tc.expectedErrorString == "" && string(body) != "{}" {
				t.Errorf("Expected change password response to be \"{}\": result: %+v", string(body))
			}

			if tc.expectChangePasswordCall {
				if withWallet {
					// Called ChangePasswordWithWallet with the expected parameters
					if want, got := (ChangePasswordWithWalletCall{
						EncryptedWallet: tc.newEncryptedWallet,
						Sequence:        tc.newSequence,
						Hmac:            tc.newHmac,
						Email:           tc.email,
						OldPassword:     oldPassword,
						NewPassword:     newPassword,
						ClientSaltSeed:  clientSaltSeed,
					}), testStore.Called.ChangePasswordWithWallet; want != got {
						t.Errorf("Store.ChangePasswordWithWallet called with: expected %+v, got %+v", want, got)
					}

					// Did *not* call ChangePasswordNoWallet
					if want, got := (ChangePasswordNoWalletCall{}), testStore.Called.ChangePasswordNoWallet; want != got {
						t.Errorf("Store.ChangePasswordNoWallet unexpectly called with: %+v", got)
					}
				} else {
					// Called ChangePasswordNoWallet with the expected parameters
					if want, got := (ChangePasswordNoWalletCall{
						Email:          tc.email,
						OldPassword:    oldPassword,
						NewPassword:    newPassword,
						ClientSaltSeed: clientSaltSeed,
					}), testStore.Called.ChangePasswordNoWallet; want != got {
						t.Errorf("Store.ChangePasswordNoWallet called with: expected %+v, got %+v", want, got)
					}

					// Did *not* call ChangePasswordWithWallet
					if want, got := (ChangePasswordWithWalletCall{}), testStore.Called.ChangePasswordWithWallet; want != got {
						t.Errorf("Store.ChangePasswordWithWallet unexpectly called with: %+v", got)
					}
				}
			} else {
				if want, got := (ChangePasswordWithWalletCall{}), testStore.Called.ChangePasswordWithWallet; want != got {
					t.Errorf("Store.ChangePasswordWithWallet unexpectly called with: %+v", got)
				}
				if want, got := (ChangePasswordNoWalletCall{}), testStore.Called.ChangePasswordNoWallet; want != got {
					t.Errorf("Store.ChangePasswordNoWallet unexpectly called with: %+v", got)
				}
			}
		})
	}
}

func TestServerValidateChangePasswordRequest(t *testing.T) {
	changePasswordRequest := ChangePasswordRequest{
		EncryptedWallet: "my-encrypted-wallet",
		Hmac:            "my-hmac",
		Sequence:        2,
		Email:           "abc@example.com",
		OldPassword:     "123",
		NewPassword:     "456",
		ClientSaltSeed:  "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
	}
	if changePasswordRequest.validate() != nil {
		t.Errorf("Expected valid ChangePasswordRequest with wallet fields to successfully validate")
	}

	changePasswordRequest = ChangePasswordRequest{
		Email:          "abc@example.com",
		OldPassword:    "123",
		NewPassword:    "456",
		ClientSaltSeed: "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
	}
	if changePasswordRequest.validate() != nil {
		t.Errorf("Expected valid ChangePasswordRequest without wallet fields to successfully validate")
	}

	tt := []struct {
		changePasswordRequest ChangePasswordRequest
		expectedErrorSubstr   string
		failureDescription    string
	}{
		{
			ChangePasswordRequest{
				EncryptedWallet: "my-encrypted-wallet",
				Hmac:            "my-hmac",
				Sequence:        2,
				Email:           "abc-example.com",
				OldPassword:     "123",
				NewPassword:     "456",
				ClientSaltSeed:  "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
			},
			"email",
			"Expected ChangePasswordRequest with invalid email to return an appropriate error",
		}, {
			// Note that Golang's email address parser, which I use, will accept
			// "Abc <abc@example.com>" so we need to make sure to avoid accepting it. See
			// the implementation.
			ChangePasswordRequest{
				EncryptedWallet: "my-encrypted-wallet",
				Hmac:            "my-hmac",
				Sequence:        2,
				Email:           "Abc <abc@example.com>",
				OldPassword:     "123",
				NewPassword:     "456",
				ClientSaltSeed:  "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
			},
			"email",
			"Expected ChangePasswordRequest with email with unexpected formatting to return an appropriate error",
		}, {
			ChangePasswordRequest{
				EncryptedWallet: "my-encrypted-wallet",
				Hmac:            "my-hmac",
				Sequence:        2,
				OldPassword:     "123",
				NewPassword:     "456",
				ClientSaltSeed:  "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
			},
			"email",
			"Expected ChangePasswordRequest with missing email to return an appropriate error",
		}, {
			ChangePasswordRequest{
				EncryptedWallet: "my-encrypted-wallet",
				Hmac:            "my-hmac",
				Sequence:        2,
				Email:           "abc@example.com",
				NewPassword:     "456",
				ClientSaltSeed:  "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
			},
			"oldPassword",
			"Expected ChangePasswordRequest with missing old password to return an appropriate error",
		}, {
			ChangePasswordRequest{
				EncryptedWallet: "my-encrypted-wallet",
				Hmac:            "my-hmac",
				Sequence:        2,
				Email:           "abc@example.com",
				OldPassword:     "123",
				ClientSaltSeed:  "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
			},
			"newPassword",
			"Expected ChangePasswordRequest with missing new password to return an appropriate error",
		}, {
			ChangePasswordRequest{
				EncryptedWallet: "my-encrypted-wallet",
				Hmac:            "my-hmac",
				Sequence:        2,
				Email:           "abc@example.com",
				OldPassword:     "123",
				NewPassword:     "456",
			},
			"clientSaltSeed",
			"Expected ChangePasswordRequest with missing clientSaltSeed to return an appropriate error",
		}, {
			ChangePasswordRequest{
				EncryptedWallet: "my-encrypted-wallet",
				Hmac:            "my-hmac",
				Sequence:        2,
				Email:           "abc@example.com",
				OldPassword:     "123",
				NewPassword:     "456",
				ClientSaltSeed:  "abcd1234abcd1234abcd1234abcd1234",
			},
			"clientSaltSeed",
			"Expected ChangePasswordRequest with clientSaltSeed of wrong length to return an appropriate error",
		}, {
			ChangePasswordRequest{
				EncryptedWallet: "my-encrypted-wallet",
				Hmac:            "my-hmac",
				Sequence:        2,
				Email:           "abc@example.com",
				OldPassword:     "123",
				NewPassword:     "456",
				ClientSaltSeed:  "xxxx1234xxxx1234xxxx1234xxxx1234xxxx1234xxxx1234xxxx1234xxxx1234",
			},
			"clientSaltSeed",
			"Expected ChangePasswordRequest with clientSaltSeed with a non-hex string to return an appropriate error",
		}, {
			ChangePasswordRequest{
				Hmac:           "my-hmac",
				Sequence:       2,
				Email:          "abc@example.com",
				OldPassword:    "123",
				NewPassword:    "456",
				ClientSaltSeed: "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
			},
			"'encryptedWallet', 'sequence', and 'hmac'", // More likely to fail when we change the error message but whatever
			"Expected ChangePasswordRequest with missing encrypted wallet (but with other wallet fields present) to return an appropriate error",
		}, {
			ChangePasswordRequest{
				EncryptedWallet: "my-encrypted-wallet",
				Sequence:        2,
				Email:           "abc@example.com",
				OldPassword:     "123",
				NewPassword:     "456",
				ClientSaltSeed:  "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
			},
			"'encryptedWallet', 'sequence', and 'hmac'", // More likely to fail when we change the error message but whatever
			"Expected ChangePasswordRequest with missing hmac (but with other wallet fields present) to return an appropriate error",
		}, {
			ChangePasswordRequest{
				EncryptedWallet: "my-encrypted-wallet",
				Hmac:            "my-hmac",
				Sequence:        0,
				Email:           "abc@example.com",
				OldPassword:     "123",
				NewPassword:     "456",
				ClientSaltSeed:  "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
			},
			"'encryptedWallet', 'sequence', and 'hmac'", // More likely to fail when we change the error message but whatever
			"Expected ChangePasswordRequest with sequence < 1 (but with other wallet fields present) to return an appropriate error",
		}, {
			ChangePasswordRequest{
				EncryptedWallet: "my-encrypted-wallet",
				Hmac:            "my-hmac",
				Sequence:        2,
				Email:           "abc@example.com",
				OldPassword:     "123",
				NewPassword:     "123",
				ClientSaltSeed:  "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
			},
			"should not be the same",
			"Expected ChangePasswordRequest with password that does not change to return an appropriate error",
		},
	}
	for _, tc := range tt {
		err := tc.changePasswordRequest.validate()
		if !strings.Contains(err.Error(), tc.expectedErrorSubstr) {
			t.Errorf(tc.failureDescription)
		}
	}
}
