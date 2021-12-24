package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

////////////////
// TODO move to testing helper file

type TestStore struct {
	FailSave bool

	SaveTokenCalled bool
}

type TestAuth struct {
	TestToken    AuthTokenString
	FailSigCheck bool
	FailGenToken bool
}

func (a *TestAuth) NewFullToken(pubKey PublicKey, tokenRequest *TokenRequest) (*AuthToken, error) {
	if a.FailGenToken {
		return nil, fmt.Errorf("Test error: fail to generate token")
	}
	return &AuthToken{Token: a.TestToken}, nil
}

func (a *TestAuth) IsValidSignature(pubKey PublicKey, payload string, signature string) bool {
	return !a.FailSigCheck
}

func (s *TestStore) SaveToken(token *AuthToken) error {
	if s.FailSave {
		return fmt.Errorf("TestStore.SaveToken fail")
	}
	s.SaveTokenCalled = true
	return nil
}

////////////////

func TestServerAuthHandlerSuccess(t *testing.T) {
	testAuth := TestAuth{TestToken: AuthTokenString("seekrit")}
	testStore := TestStore{}
	s := Server{
		&testAuth,
		&testStore,
	}

	requestBody := []byte(`
	{
	  "tokenRequestJSON": "{}"
  }
	`)

	req := httptest.NewRequest(http.MethodPost, PathGetAuthToken, bytes.NewBuffer(requestBody))
	w := httptest.NewRecorder()

	s.getAuthToken(w, req)
	body, _ := ioutil.ReadAll(w.Body)

	var result AuthToken

	if want, got := http.StatusOK, w.Result().StatusCode; want != got {
		t.Errorf("StatusCode: expected %s (%d), got %s (%d)", http.StatusText(want), want, http.StatusText(got), got)
	}

	err := json.Unmarshal(body, &result)

	if err != nil || result.Token != testAuth.TestToken {
		t.Errorf("Expected auth response to contain token: result: %+v err: %+v", string(body), err)
	}

	if !testStore.SaveTokenCalled {
		t.Errorf("Expected Store.SaveToken to be called")
	}
}

func TestServerAuthHandlerErrors(t *testing.T) {
	tt := []struct {
		name                string
		method              string
		requestBody         string
		expectedStatusCode  int
		expectedErrorString string

		authFailSigCheck bool
		authFailGenToken bool
		storeFailSave    bool
	}{
		{
			name:                "bad method",
			method:              http.MethodGet,
			requestBody:         "",
			expectedStatusCode:  http.StatusMethodNotAllowed,
			expectedErrorString: http.StatusText(http.StatusMethodNotAllowed),
		},
		{
			name:                "request body too large",
			method:              http.MethodPost,
			requestBody:         fmt.Sprintf("{\"tokenRequestJSON\": \"%s\"}", strings.Repeat("a", 10000)),
			expectedStatusCode:  http.StatusRequestEntityTooLarge,
			expectedErrorString: http.StatusText(http.StatusRequestEntityTooLarge),
		},
		{
			name:                "malformed request body JSON",
			method:              http.MethodPost,
			requestBody:         "{",
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrorString: http.StatusText(http.StatusBadRequest) + ": Malformed request body JSON",
		},
		{
			name:   "signature check fail",
			method: http.MethodPost,
			// so long as the JSON is well-formed, the content doesn't matter here since the signature check will be stubbed out
			requestBody:         "{\"tokenRequestJSON\": \"{}\"}",
			expectedStatusCode:  http.StatusForbidden,
			expectedErrorString: http.StatusText(http.StatusForbidden) + ": Bad signature",

			authFailSigCheck: true,
		},
		{
			name:                "malformed tokenRequest JSON",
			method:              http.MethodPost,
			requestBody:         "{\"tokenRequestJSON\": \"{\"}",
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrorString: http.StatusText(http.StatusBadRequest) + ": Malformed tokenRequest JSON",
		},
		{
			name:                "generate token fail",
			method:              http.MethodPost,
			requestBody:         "{\"tokenRequestJSON\": \"{}\"}",
			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrorString: http.StatusText(http.StatusInternalServerError) + ": Error generating auth token",

			authFailGenToken: true,
		},
		{
			name:                "save token fail",
			method:              http.MethodPost,
			requestBody:         "{\"tokenRequestJSON\": \"{}\"}",
			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrorString: http.StatusText(http.StatusInternalServerError) + ": Error saving auth token",

			storeFailSave: true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			// Set this up to fail according to specification
			testAuth := TestAuth{TestToken: AuthTokenString("seekrit")}
			testStore := TestStore{}
			if tc.authFailSigCheck {
				testAuth.FailSigCheck = true
			} else if tc.authFailGenToken {
				testAuth.FailGenToken = true
			} else if tc.storeFailSave {
				testStore.FailSave = true
			} else {
				testAuth.TestToken = AuthTokenString("seekrit")
			}
			server := Server{&testAuth, &testStore}

			// Make request
			req := httptest.NewRequest(tc.method, PathGetAuthToken, bytes.NewBuffer([]byte(tc.requestBody)))
			w := httptest.NewRecorder()

			server.getAuthToken(w, req)

			if want, got := tc.expectedStatusCode, w.Result().StatusCode; want != got {
				t.Errorf("StatusCode: expected %d, got %d", want, got)
			}

			body, _ := ioutil.ReadAll(w.Body)

			var result ErrorResponse
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatalf("Error decoding error message %s: `%s`", err, body)
			}

			if want, got := tc.expectedErrorString, result.Error; want != got {
				t.Errorf("Error String: expected %s, got %s", want, got)
			}
		})
	}
}

func TestServerValidateAuthRequest(t *testing.T) {
	// also add a basic test case for this in TestAuthHandlerErrors to make sure it's called at all
	// Maybe 401 specifically for missing signature?
	t.Fatalf("Implement and test validateAuthRequest")
}

func TestServerValidateTokenRequest(t *testing.T) {
	// also add a basic test case for this in TestAuthHandlerErrors to make sure it's called at all
	t.Fatalf("Implement and test validateTokenRequest")
}

func TestServerGetWalletSuccess(t *testing.T) {
	t.Fatalf("GetWallet succeeds")
}

func TestServerGetWalletErrors(t *testing.T) {
	t.Fatalf("GetWallet fails for various reasons (malformed, auth, db fail)")
}

func TestServerPutWalletSuccess(t *testing.T) {
	t.Fatalf("GetWallet succeeds")
}

func TestServerPutWalletErrors(t *testing.T) {
	t.Fatalf("GetWallet fails for various reasons (malformed, auth, db fail)")
}
