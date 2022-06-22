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

	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/store"
	"orblivion/lbry-id/wallet"
)

// Implementing interfaces for stubbed out packages

type TestAuth struct {
	TestNewTokenString auth.TokenString
	FailGenToken       bool
}

func (a *TestAuth) NewToken(userId auth.UserId, deviceId auth.DeviceId, scope auth.AuthScope) (*auth.AuthToken, error) {
	if a.FailGenToken {
		return nil, fmt.Errorf("Test error: fail to generate token")
	}
	return &auth.AuthToken{Token: a.TestNewTokenString, UserId: userId, DeviceId: deviceId, Scope: scope}, nil
}

type SetWalletCall struct {
	EncryptedWallet wallet.EncryptedWallet
	Sequence        wallet.Sequence
	Hmac            wallet.WalletHmac
}

// Whether functions are called, and sometimes what they're called with
type TestStoreFunctionsCalled struct {
	SaveToken     auth.TokenString
	GetToken      auth.TokenString
	GetUserId     bool
	CreateAccount bool
	SetWallet     SetWalletCall
	GetWallet     bool
}

type TestStoreFunctionsErrors struct {
	SaveToken     error
	GetToken      error
	GetUserId     error
	CreateAccount error
	SetWallet     error
	GetWallet     error
}

type TestStore struct {
	// Fake store functions will set these to `true` as they are called
	Called TestStoreFunctionsCalled

	// Fake store functions will return the errors (including `nil`) specified in
	// the test setup
	Errors TestStoreFunctionsErrors

	TestAuthToken auth.AuthToken

	TestEncryptedWallet wallet.EncryptedWallet
	TestSequence        wallet.Sequence
	TestHmac            wallet.WalletHmac
	TestSequenceCorrect bool
}

func (s *TestStore) SaveToken(authToken *auth.AuthToken) error {
	s.Called.SaveToken = authToken.Token
	return s.Errors.SaveToken
}

func (s *TestStore) GetToken(token auth.TokenString) (*auth.AuthToken, error) {
	s.Called.GetToken = token
	return &s.TestAuthToken, s.Errors.GetToken
}

func (s *TestStore) GetUserId(auth.Email, auth.Password) (auth.UserId, error) {
	s.Called.GetUserId = true
	return 0, s.Errors.GetUserId
}

func (s *TestStore) CreateAccount(auth.Email, auth.Password) error {
	s.Called.CreateAccount = true
	return s.Errors.CreateAccount
}

func (s *TestStore) SetWallet(
	UserId auth.UserId,
	encryptedWallet wallet.EncryptedWallet,
	sequence wallet.Sequence,
	hmac wallet.WalletHmac,
) (latestEncryptedWallet wallet.EncryptedWallet, latestSequence wallet.Sequence, latestHmac wallet.WalletHmac, sequenceCorrect bool, err error) {
	s.Called.SetWallet = SetWalletCall{encryptedWallet, sequence, hmac}
	err = s.Errors.SetWallet
	if err == nil {
		latestEncryptedWallet = s.TestEncryptedWallet
		latestSequence = s.TestSequence
		latestHmac = s.TestHmac
		sequenceCorrect = s.TestSequenceCorrect
	}
	return
}

func (s *TestStore) GetWallet(userId auth.UserId) (encryptedWallet wallet.EncryptedWallet, sequence wallet.Sequence, hmac wallet.WalletHmac, err error) {
	s.Called.GetWallet = true
	err = s.Errors.GetWallet
	if err == nil {
		encryptedWallet = s.TestEncryptedWallet
		sequence = s.TestSequence
		hmac = s.TestHmac
	}
	return
}

// expectStatusCode: A helper to call in functions that test that request
// handlers responded with a certain status code. Cuts down on noise.
func expectStatusCode(t *testing.T, w *httptest.ResponseRecorder, expectedStatusCode int) {
	if want, got := expectedStatusCode, w.Result().StatusCode; want != got {
		t.Errorf("StatusCode: expected %s (%d), got %s (%d)", http.StatusText(want), want, http.StatusText(got), got)
	}
}

// expectErrorString: A helper to call in functions that test that request
// handlers failed with a certain error string. Cuts down on noise.
func expectErrorString(t *testing.T, w *httptest.ResponseRecorder, expectedErrorString string) {
	body, _ := ioutil.ReadAll(w.Body)

	var result ErrorResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Error decoding error message: %s: `%s`", err, body)
	}

	if want, got := expectedErrorString, result.Error; want != got {
		t.Errorf("Error String: expected %s, got %s", want, got)
	}
}

func TestServerHelperCheckAuth(t *testing.T) {
	tt := []struct {
		name          string
		requiredScope auth.AuthScope
		userScope     auth.AuthScope

		tokenExpected       bool
		expectedStatusCode  int
		expectedErrorString string

		storeErrors TestStoreFunctionsErrors
	}{
		{
			name: "success",
			// Just check that scope checks exist. The more detailed specific tests
			// go in the auth module
			requiredScope: auth.AuthScope("banana"),
			userScope:     auth.AuthScope("*"),

			// not that it's a full request but as of now no error yet means 200 by default
			expectedStatusCode: 200,
			tokenExpected:      true,
		}, {
			name:          "auth token not found",
			requiredScope: auth.AuthScope("banana"),
			userScope:     auth.AuthScope("*"),

			expectedStatusCode:  http.StatusUnauthorized,
			expectedErrorString: http.StatusText(http.StatusUnauthorized) + ": Token Not Found",

			storeErrors: TestStoreFunctionsErrors{GetToken: store.ErrNoToken},
		}, {
			name:          "unknown auth token db error",
			requiredScope: auth.AuthScope("banana"),
			userScope:     auth.AuthScope("*"),

			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrorString: http.StatusText(http.StatusInternalServerError),

			storeErrors: TestStoreFunctionsErrors{GetToken: fmt.Errorf("Some random DB Error!")},
		}, {
			name:          "auth scope failure",
			requiredScope: auth.AuthScope("banana"),
			userScope:     auth.AuthScope("carrot"),

			expectedStatusCode:  http.StatusForbidden,
			expectedErrorString: http.StatusText(http.StatusForbidden) + ": Scope",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			testStore := TestStore{
				Errors:        tc.storeErrors,
				TestAuthToken: auth.AuthToken{Token: auth.TokenString("seekrit"), Scope: tc.userScope},
			}
			s := Server{&TestAuth{}, &testStore}

			w := httptest.NewRecorder()
			authToken := s.checkAuth(w, testStore.TestAuthToken.Token, tc.requiredScope)
			if tc.tokenExpected && (*authToken != testStore.TestAuthToken) {
				t.Errorf("Expected checkAuth to return a valid AuthToken")
			}
			if !tc.tokenExpected && (authToken != nil) {
				t.Errorf("Expected checkAuth not to return a valid AuthToken")
			}
			expectStatusCode(t, w, tc.expectedStatusCode)
			if len(tc.expectedErrorString) > 0 {
				expectErrorString(t, w, tc.expectedErrorString)
			}
		})
	}
}

func TestServerHelperGetGetDataSuccess(t *testing.T) {
	t.Fatalf("Test me: getGetData success")
}
func TestServerHelperGetGetDataErrors(t *testing.T) {
	t.Fatalf("Test me: getGetData failure")
}

type TestReqStruct struct{ key string }

func (t *TestReqStruct) validate() bool { return t.key != "" }

func TestServerHelperGetPostDataSuccess(t *testing.T) {
	requestBody := []byte(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBuffer(requestBody))
	w := httptest.NewRecorder()
	success := getPostData(w, req, &TestReqStruct{key: "hi"})
	if !success {
		t.Errorf("getPostData failed unexpectedly")
	}
}

// Test getPostData, including requestOverhead and any other mini-helpers it calls.
func TestServerHelperGetPostDataErrors(t *testing.T) {
	tt := []struct {
		name                string
		method              string
		requestBody         string
		expectedStatusCode  int
		expectedErrorString string
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
			requestBody:         fmt.Sprintf(`{"key": "%s"}`, strings.Repeat("a", 100000)),
			expectedStatusCode:  http.StatusRequestEntityTooLarge,
			expectedErrorString: http.StatusText(http.StatusRequestEntityTooLarge),
		},
		{
			name:                "malformed request body JSON",
			method:              http.MethodPost,
			requestBody:         "{",
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrorString: http.StatusText(http.StatusBadRequest) + ": Error parsing JSON",
		},
		{
			name:                "body JSON failed validation",
			method:              http.MethodPost,
			requestBody:         "{}",
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrorString: http.StatusText(http.StatusBadRequest) + ": Request failed validation",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// Make request
			req := httptest.NewRequest(tc.method, PathAuthToken, bytes.NewBuffer([]byte(tc.requestBody)))
			w := httptest.NewRecorder()

			success := getPostData(w, req, &TestReqStruct{})
			if success {
				t.Errorf("getPostData succeeded unexpectedly")
			}

			expectStatusCode(t, w, tc.expectedStatusCode)
			expectErrorString(t, w, tc.expectedErrorString)
		})
	}
}
