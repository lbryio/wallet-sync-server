package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/store"
	"orblivion/lbry-id/wallet"
)

func TestServerGetWallet(t *testing.T) {
	tt := []struct {
		name string

		expectedStatusCode  int
		expectedErrorString string

		storeErrors TestStoreFunctionsErrors
	}{
		{
			name:               "success",
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "auth error",

			expectedStatusCode:  http.StatusUnauthorized,
			expectedErrorString: http.StatusText(http.StatusUnauthorized) + ": Token Not Found",

			storeErrors: TestStoreFunctionsErrors{GetToken: store.ErrNoToken},
		},
		{
			name: "db error getting wallet",

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
					Token: auth.TokenString("seekrit"),
					Scope: auth.ScopeFull,
				},

				TestEncryptedWallet: wallet.EncryptedWallet("my-encrypted-wallet"),
				TestSequence:        wallet.Sequence(2),
				TestHmac:            wallet.WalletHmac("my-hmac"),

				Errors: tc.storeErrors,
			}

			s := Server{&testAuth, &testStore}

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
			if testStore.Called.GetToken != testStore.TestAuthToken.Token {
				t.Errorf("Expected Store.GetToken to be called with %s. Got %s",
					testStore.TestAuthToken.Token,
					testStore.Called.GetToken)
			}

			expectStatusCode(t, w, tc.expectedStatusCode)

			if len(tc.expectedErrorString) > 0 {
				// Only check if we're expecting an error, since it reads the body
				expectErrorString(t, w, tc.expectedErrorString)
				return
			}

			body, _ := ioutil.ReadAll(w.Body)
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

func TestServerPostWalletTooLate(t *testing.T) {
	t.Fatalf("Test me: PostWallet fails for sequence being too low, returns the latest wallet")
}

func TestServerPostWalletErrors(t *testing.T) {
	// (malformed json, db fail, auth token not found, wallet metadata invalid (via stub, make sure the validation function is even called), sequence too high, device id doesn't match token device id)
	// Client sends sequence != 1 for first entry
	// Client sends sequence == x + 10 for xth entry or whatever
	t.Fatalf("Test me: PostWallet fails for various reasons")
}

func TestServerValidateWalletRequest(t *testing.T) {
	// also add a basic test case for this in TestServerAuthHandlerSuccess to make sure it's called at all
	t.Fatalf("Test me: Implement and test WalletRequest.validate()")
}

func TestServerHandleWallet(t *testing.T) {
	t.Fatalf("Test me: Call the get or post function as appropriate. Alternately: call handleWallet for the existing tests.")
}
