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
	"time"

	"lbryio/wallet-sync-server/auth"
	"lbryio/wallet-sync-server/server/paths"
	"lbryio/wallet-sync-server/store"
	"lbryio/wallet-sync-server/wallet"
)

const TestPort = 8090

// Implementing interfaces for stubbed out packages

type SendVerificationEmailCall struct {
	Email auth.Email
	Token auth.VerifyTokenString
}

type TestMail struct {
	SendVerificationEmailError error
	SendVerificationEmailCall  *SendVerificationEmailCall
}

func (m *TestMail) SendVerificationEmail(email auth.Email, token auth.VerifyTokenString) error {
	m.SendVerificationEmailCall = &SendVerificationEmailCall{email, token}
	return m.SendVerificationEmailError
}

type TestEnv struct {
	env map[string]string
}

func (e *TestEnv) Getenv(key string) string {
	return e.env[key]
}

type TestAuth struct {
	TestNewAuthTokenString   auth.AuthTokenString
	TestNewVerifyTokenString auth.VerifyTokenString
	FailGenToken             bool
}

func (a *TestAuth) NewAuthToken(userId auth.UserId, deviceId auth.DeviceId, scope auth.AuthScope) (*auth.AuthToken, error) {
	if a.FailGenToken {
		return nil, fmt.Errorf("Test error: fail to generate token")
	}
	return &auth.AuthToken{Token: a.TestNewAuthTokenString, UserId: userId, DeviceId: deviceId, Scope: scope}, nil
}

func (a *TestAuth) NewVerifyTokenString() (auth.VerifyTokenString, error) {
	if a.FailGenToken {
		return "", fmt.Errorf("Test error: fail to generate token")
	}
	return a.TestNewVerifyTokenString, nil
}

type SetWalletCall struct {
	EncryptedWallet wallet.EncryptedWallet
	Sequence        wallet.Sequence
	Hmac            wallet.WalletHmac
}

type ChangePasswordNoWalletCall struct {
	Email          auth.Email
	OldPassword    auth.Password
	NewPassword    auth.Password
	ClientSaltSeed auth.ClientSaltSeed
}

type ChangePasswordWithWalletCall struct {
	EncryptedWallet wallet.EncryptedWallet
	Sequence        wallet.Sequence
	Hmac            wallet.WalletHmac
	Email           auth.Email
	OldPassword     auth.Password
	NewPassword     auth.Password
	ClientSaltSeed  auth.ClientSaltSeed
}

type CreateAccountCall struct {
	Email          auth.Email
	Password       auth.Password
	ClientSaltSeed auth.ClientSaltSeed
	VerifyToken    *auth.VerifyTokenString
}

// Whether functions are called, and sometimes what they're called with
type TestStoreFunctionsCalled struct {
	SaveToken                auth.AuthTokenString
	GetToken                 auth.AuthTokenString
	GetUserId                bool
	CreateAccount            *CreateAccountCall
	UpdateVerifyTokenString  bool
	VerifyAccount            bool
	SetWallet                SetWalletCall
	GetWallet                bool
	ChangePasswordWithWallet ChangePasswordWithWalletCall
	ChangePasswordNoWallet   ChangePasswordNoWalletCall
	GetClientSaltSeed        auth.Email
}

type TestStoreFunctionsErrors struct {
	SaveToken                error
	GetToken                 error
	GetUserId                error
	CreateAccount            error
	UpdateVerifyTokenString  error
	VerifyAccount            error
	SetWallet                error
	GetWallet                error
	ChangePasswordWithWallet error
	ChangePasswordNoWallet   error
	GetClientSaltSeed        error
}

type TestStore struct {
	// Fake store functions will set these to `true` as they are called
	Called TestStoreFunctionsCalled

	// Fake store functions will return the errors (including `nil`) specified in
	// the test setup
	Errors TestStoreFunctionsErrors

	TestAuthToken auth.AuthToken
	TestUserId    auth.UserId

	TestEncryptedWallet wallet.EncryptedWallet
	TestSequence        wallet.Sequence
	TestHmac            wallet.WalletHmac

	TestClientSaltSeed auth.ClientSaltSeed
}

func (s *TestStore) SaveToken(authToken *auth.AuthToken) error {
	s.Called.SaveToken = authToken.Token
	return s.Errors.SaveToken
}

func (s *TestStore) GetToken(token auth.AuthTokenString) (*auth.AuthToken, error) {
	s.Called.GetToken = token
	return &s.TestAuthToken, s.Errors.GetToken
}

func (s *TestStore) GetUserId(auth.Email, auth.Password) (auth.UserId, error) {
	s.Called.GetUserId = true
	return 0, s.Errors.GetUserId
}

func (s *TestStore) CreateAccount(email auth.Email, password auth.Password, seed auth.ClientSaltSeed, verifyToken *auth.VerifyTokenString) error {
	s.Called.CreateAccount = &CreateAccountCall{
		Email:          email,
		Password:       password,
		ClientSaltSeed: seed,
		VerifyToken:    verifyToken,
	}
	return s.Errors.CreateAccount
}

func (s *TestStore) UpdateVerifyTokenString(auth.Email, auth.VerifyTokenString) (err error) {
	s.Called.UpdateVerifyTokenString = true
	return s.Errors.UpdateVerifyTokenString
}

func (s *TestStore) VerifyAccount(auth.VerifyTokenString) (err error) {
	s.Called.VerifyAccount = true
	return s.Errors.VerifyAccount
}

func (s *TestStore) SetWallet(
	UserId auth.UserId,
	encryptedWallet wallet.EncryptedWallet,
	sequence wallet.Sequence,
	hmac wallet.WalletHmac,
) (err error) {
	s.Called.SetWallet = SetWalletCall{encryptedWallet, sequence, hmac}
	return s.Errors.SetWallet
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

func (s *TestStore) ChangePasswordWithWallet(
	email auth.Email,
	oldPassword auth.Password,
	newPassword auth.Password,
	clientSaltSeed auth.ClientSaltSeed,
	encryptedWallet wallet.EncryptedWallet,
	sequence wallet.Sequence,
	hmac wallet.WalletHmac,
) (auth.UserId, error) {
	s.Called.ChangePasswordWithWallet = ChangePasswordWithWalletCall{
		EncryptedWallet: encryptedWallet,
		Sequence:        sequence,
		Hmac:            hmac,
		Email:           email,
		OldPassword:     oldPassword,
		NewPassword:     newPassword,
		ClientSaltSeed:  clientSaltSeed,
	}
	return s.TestUserId, s.Errors.ChangePasswordWithWallet
}

func (s *TestStore) ChangePasswordNoWallet(
	email auth.Email,
	oldPassword auth.Password,
	newPassword auth.Password,
	clientSaltSeed auth.ClientSaltSeed,
) (auth.UserId, error) {
	s.Called.ChangePasswordNoWallet = ChangePasswordNoWalletCall{
		Email:          email,
		OldPassword:    oldPassword,
		NewPassword:    newPassword,
		ClientSaltSeed: clientSaltSeed,
	}
	return s.TestUserId, s.Errors.ChangePasswordNoWallet
}

func (s *TestStore) GetClientSaltSeed(email auth.Email) (seed auth.ClientSaltSeed, err error) {
	s.Called.GetClientSaltSeed = email
	err = s.Errors.GetClientSaltSeed
	if err == nil {
		seed = s.TestClientSaltSeed
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
func expectErrorString(t *testing.T, body []byte, expectedErrorString string) {
	if len(body) == 0 {
		// Nothing to decode
		if expectedErrorString == "" {
			return // Nothing expected either, we're all good
		}
		t.Errorf("Error String: expected %s, got an empty body (no JSON to decode)", expectedErrorString)
	}

	var result ErrorResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Error decoding error message: %s: `%s`", err, body)
	}

	if want, got := expectedErrorString, result.Error; want != got {
		t.Errorf("Error String: expected %s, got %s", want, got)
	}
}

type wsMockManager struct {
	s    *Server
	done chan bool

	addedClientUserId   auth.UserId
	removedClientUserId auth.UserId
	removedUserId       auth.UserId
	walletUpdateUserId  auth.UserId
	noMessage           bool
}

func (m *wsMockManager) getOneMessage(timeout time.Duration) {
	t := time.NewTicker(timeout)
	select {
	case msg := <-m.s.clientAdd:
		m.addedClientUserId = msg.userId
	case msg := <-m.s.clientRemove:
		m.removedClientUserId = msg.userId
	case msg := <-m.s.userRemove:
		m.removedUserId = msg.userId
	case msg := <-m.s.walletUpdates:
		m.walletUpdateUserId = msg.userId
	case <-t.C:
		m.noMessage = true
	}
	t.Stop()
	m.done <- true
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

			storeErrors: TestStoreFunctionsErrors{GetToken: store.ErrNoTokenForUserDevice},
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
				TestAuthToken: auth.AuthToken{Token: auth.AuthTokenString("seekrit"), Scope: tc.userScope},
			}
			s := Init(&TestAuth{}, &testStore, &TestEnv{}, &TestMail{}, TestPort)

			w := httptest.NewRecorder()
			authToken := s.checkAuth(w, testStore.TestAuthToken.Token, tc.requiredScope)
			if tc.tokenExpected && (*authToken != testStore.TestAuthToken) {
				t.Errorf("Expected checkAuth to return a valid AuthToken")
			}
			if !tc.tokenExpected && (authToken != nil) {
				t.Errorf("Expected checkAuth not to return a valid AuthToken")
			}
			body, _ := ioutil.ReadAll(w.Body)

			expectStatusCode(t, w, tc.expectedStatusCode)
			expectErrorString(t, body, tc.expectedErrorString)
		})
	}
}

func TestServerHelperGetGetDataSuccess(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	success := getGetData(w, req)
	if !success {
		t.Errorf("getGetData failed unexpectedly")
	}
}
func TestServerHelperGetGetDataErrors(t *testing.T) {
	// Only error right now is if you do a POST request
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	success := getGetData(w, req)
	if success {
		t.Errorf("getGetData succeeded unexpectedly")
	}
}

type TestReqStruct struct{ key string }

func (t *TestReqStruct) validate() error {
	if t.key == "" {
		return fmt.Errorf("TestReq Error")
	} else {
		return nil
	}
}

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
			expectedErrorString: http.StatusText(http.StatusBadRequest) + ": Request failed validation: TestReq Error",
		},
		{
			name:                "body JSON has unknown field",
			method:              http.MethodPost,
			requestBody:         `{"lol": "wut"}`,
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrorString: http.StatusText(http.StatusBadRequest) + `: json: unknown field "lol"`,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// Make request
			req := httptest.NewRequest(tc.method, paths.PathAuthToken, bytes.NewBuffer([]byte(tc.requestBody)))
			w := httptest.NewRecorder()

			success := getPostData(w, req, &TestReqStruct{})
			if success {
				t.Errorf("getPostData succeeded unexpectedly")
			}
			body, _ := ioutil.ReadAll(w.Body)

			expectStatusCode(t, w, tc.expectedStatusCode)
			expectErrorString(t, body, tc.expectedErrorString)
		})
	}
}
