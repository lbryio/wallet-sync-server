package server

import (
	"bytes"
	"encoding/json"
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

func TestServerGetWallet(t *testing.T) {
	tt := []struct {
		name        string
		tokenString auth.TokenString

		expectedStatusCode  int
		expectedErrorString string

		storeErrors TestStoreFunctionsErrors
	}{
		{
			name:               "success",
			tokenString:        auth.TokenString("seekrit"),
			expectedStatusCode: http.StatusOK,
		},
		{
			name:                "validation error", // missing auth token
			tokenString:         auth.TokenString(""),
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrorString: http.StatusText(http.StatusBadRequest) + ": Missing token parameter",

			// Just check one validation error (missing auth token) to make sure the
			// validate function is called. We'll check the rest of the validation
			// errors in the other test below.
		},
		{
			name:        "auth error",
			tokenString: auth.TokenString("seekrit"),

			expectedStatusCode:  http.StatusUnauthorized,
			expectedErrorString: http.StatusText(http.StatusUnauthorized) + ": Token Not Found",

			storeErrors: TestStoreFunctionsErrors{GetToken: store.ErrNoTokenForUserDevice},
		},
		{
			name:        "db error getting wallet",
			tokenString: auth.TokenString("seekrit"),

			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrorString: http.StatusText(http.StatusInternalServerError),

			storeErrors: TestStoreFunctionsErrors{GetWallet: fmt.Errorf("Some random DB Error!")},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			testAuth := TestAuth{}
			testStore := TestStore{
				TestAuthToken: auth.AuthToken{
					Token: auth.TokenString(tc.tokenString),
					Scope: auth.ScopeFull,
				},

				TestEncryptedWallet: wallet.EncryptedWallet("my-encrypted-wallet"),
				TestSequence:        wallet.Sequence(2),
				TestHmac:            wallet.WalletHmac("my-hmac"),

				Errors: tc.storeErrors,
			}

			testEnv := TestEnv{}
			s := Server{&testAuth, &testStore, &testEnv}

			req := httptest.NewRequest(http.MethodGet, PathWallet, nil)
			q := req.URL.Query()
			q.Add("token", string(testStore.TestAuthToken.Token))
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()

			// test handleWallet while we're at it, which is a dispatch for get and post
			// wallet
			s.handleWallet(w, req)

			// Make sure we tried to get an auth based on the `token` param (whether or
			// not it was a valid `token`)
			// NOTE: For tests that set testStore.TestAuthToken.Token=="", this will
			// pass even if GetToken isn't called. But we don't care, we expect the
			// request to fail for other reasons at that point.
			if want, got := testStore.TestAuthToken.Token, testStore.Called.GetToken; want != got {
				t.Errorf("testStore.Called.GetToken called with: expected %s, got %s", want, got)
			}

			body, _ := ioutil.ReadAll(w.Body)

			expectStatusCode(t, w, tc.expectedStatusCode)
			expectErrorString(t, body, tc.expectedErrorString)

			// In this case, a wallet body is expected iff there is no error string
			expectWalletBody := len(tc.expectedErrorString) == 0

			if !expectWalletBody {
				return // The rest of the test does not apply
			}

			var result WalletResponse
			err := json.Unmarshal(body, &result)

			if err != nil ||
				result.EncryptedWallet != testStore.TestEncryptedWallet ||
				result.Hmac != testStore.TestHmac ||
				result.Sequence != testStore.TestSequence {
				t.Errorf("Expected wallet response to have the test wallet values: result: %+v err: %+v", string(body), err)
			}

			if !testStore.Called.GetWallet {
				t.Errorf("Expected Store.GetWallet to be called")
			}
		})
	}
}

func TestServerPostWallet(t *testing.T) {
	tt := []struct {
		name string

		expectedStatusCode  int
		expectedErrorString string
		expectSetWalletCall bool

		// This is getting messy, but in the case of validation failures, we don't
		// even get around to trying to get an auth token, since the token string is
		// part of what's being validated. So, we want to be able to skip that
		// check in that case.
		skipAuthCheck bool

		// `new...` refers to what is being passed into the via POST request (and
		//   what we expect to get passed into SetWallet for the *non-error* cases
		//   below)
		newEncryptedWallet wallet.EncryptedWallet
		newSequence        wallet.Sequence
		newHmac            wallet.WalletHmac

		storeErrors TestStoreFunctionsErrors
	}{
		{
			name:                "success",
			expectedStatusCode:  http.StatusOK,
			expectSetWalletCall: true,

			// Simulates a situation where the existing sequence is 1, the new
			// sequence is 2.

			newEncryptedWallet: wallet.EncryptedWallet("my-encrypted-wallet"),
			newSequence:        wallet.Sequence(2),
			newHmac:            wallet.WalletHmac("my-hmac"),
		}, {
			name:                "conflict",
			expectedStatusCode:  http.StatusConflict,
			expectedErrorString: http.StatusText(http.StatusConflict) + ": Bad sequence number",
			expectSetWalletCall: true,

			// Simulates a situation where the existing sequence is *not* 1, the new
			// proposed sequence is 2, and it thus fails with a conflict.

			newEncryptedWallet: wallet.EncryptedWallet("my-encrypted-wallet-new"),
			newSequence:        wallet.Sequence(2),
			newHmac:            wallet.WalletHmac("my-hmac-new"),

			storeErrors: TestStoreFunctionsErrors{SetWallet: store.ErrWrongSequence},
		}, {
			name:                "validation error",
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrorString: http.StatusText(http.StatusBadRequest) + ": Request failed validation: Missing 'encryptedWallet'",
			skipAuthCheck:       true, // we can't get an auth token without the data we just failed to validate

			// Just check one validation error (empty encrypted wallet) to make sure the
			// validate function is called. We'll check the rest of the validation
			// errors in the other test below.

			newEncryptedWallet: wallet.EncryptedWallet(""),
			newSequence:        wallet.Sequence(2),
			newHmac:            wallet.WalletHmac("my-hmac"),
		}, {
			name:                "auth error",
			expectedStatusCode:  http.StatusUnauthorized,
			expectedErrorString: http.StatusText(http.StatusUnauthorized) + ": Token Not Found",

			// Putting in valid data here so it's clear that this isn't what causes
			// the error
			newEncryptedWallet: wallet.EncryptedWallet("my-encrypted-wallet"),
			newSequence:        wallet.Sequence(2),
			newHmac:            wallet.WalletHmac("my-hmac"),

			// What causes the error
			storeErrors: TestStoreFunctionsErrors{GetToken: store.ErrNoTokenForUserDevice},
		}, {
			name:                "db error setting wallet",
			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrorString: http.StatusText(http.StatusInternalServerError),
			expectSetWalletCall: true,

			// Putting in valid data here so it's clear that this isn't what causes
			// the error
			newEncryptedWallet: wallet.EncryptedWallet("my-encrypted-wallet"),
			newSequence:        wallet.Sequence(2),
			newHmac:            wallet.WalletHmac("my-hmac"),

			// What causes the error
			storeErrors: TestStoreFunctionsErrors{SetWallet: fmt.Errorf("Some random db problem")},
		},

		// TODO
		// Future test case when we get lastSynced back: Error if
		// lastSynced.device_id doesn't match authToken.device_id

	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			testAuth := TestAuth{}
			testStore := TestStore{
				TestAuthToken: auth.AuthToken{
					Token: auth.TokenString("seekrit"),
					Scope: auth.ScopeFull,
				},

				Errors: tc.storeErrors,
			}

			s := Server{&testAuth, &testStore, &TestEnv{}}

			requestBody := []byte(
				fmt.Sprintf(`{
          "token": "%s",
          "encryptedWallet": "%s",
          "sequence": %d,
          "hmac": "%s"
        }`, testStore.TestAuthToken.Token, tc.newEncryptedWallet, tc.newSequence, tc.newHmac),
			)

			req := httptest.NewRequest(http.MethodPost, PathWallet, bytes.NewBuffer(requestBody))
			w := httptest.NewRecorder()

			// test handleWallet while we're at it, which is a dispatch for get and post
			// wallet
			s.handleWallet(w, req)

			// Make sure we tried to get an auth based on the `token` param (whether or
			// not it was a valid `token`)
			if want, got := testStore.TestAuthToken.Token, testStore.Called.GetToken; !tc.skipAuthCheck && want != got {
				t.Errorf("testStore.Called.GetToken called with: expected %s, got %s", want, got)
			}

			body, _ := ioutil.ReadAll(w.Body)

			expectStatusCode(t, w, tc.expectedStatusCode)
			expectErrorString(t, body, tc.expectedErrorString)

			if tc.expectedErrorString == "" && string(body) != "{}" {
				t.Errorf("Expected post wallet response to be \"{}\": result: %+v", string(body))
			}

			if want, got := (SetWalletCall{tc.newEncryptedWallet, tc.newSequence, tc.newHmac}), testStore.Called.SetWallet; tc.expectSetWalletCall && want != got {
				t.Errorf("Store.SetWallet called with: expected %+v, got %+v", want, got)
			}
		})
	}
}

func TestServerValidateWalletRequest(t *testing.T) {
	walletRequest := WalletRequest{Token: "seekrit", EncryptedWallet: "my-encrypted-wallet", Hmac: "my-hmac", Sequence: 2}
	if walletRequest.validate() != nil {
		t.Errorf("Expected valid WalletRequest to successfully validate")
	}

	tt := []struct {
		walletRequest       WalletRequest
		expectedErrorSubstr string
		failureDescription  string
	}{
		{
			WalletRequest{EncryptedWallet: "my-encrypted-wallet", Hmac: "my-hmac", Sequence: 2},
			"token",
			"Expected WalletRequest with missing token to not successfully validate",
		}, {
			WalletRequest{Token: "seekrit", Hmac: "my-hmac", Sequence: 2},
			"encryptedWallet",
			"Expected WalletRequest with missing encrypted wallet to not successfully validate",
		}, {
			WalletRequest{Token: "seekrit", EncryptedWallet: "my-encrypted-wallet", Sequence: 2},
			"hmac",
			"Expected WalletRequest with missing hmac to not successfully validate",
		}, {
			WalletRequest{Token: "seekrit", EncryptedWallet: "my-encrypted-wallet", Hmac: "my-hmac", Sequence: 0},
			"sequence",
			"Expected WalletRequest with sequence < 1 to not successfully validate",
		},
	}
	for _, tc := range tt {
		err := tc.walletRequest.validate()
		if err == nil || !strings.Contains(err.Error(), tc.expectedErrorSubstr) {
			t.Errorf(tc.failureDescription)
		}
	}
}
