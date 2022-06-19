package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/store"
	"orblivion/lbry-id/wallet"
	"strings"
	"testing"
)

// Implementing interfaces for stubbed out packages

type TestAuth struct {
	TestToken    auth.TokenString
	FailGenToken bool
}

func (a *TestAuth) NewToken(userId auth.UserId, deviceId auth.DeviceId, scope auth.AuthScope) (*auth.AuthToken, error) {
	if a.FailGenToken {
		return nil, fmt.Errorf("Test error: fail to generate token")
	}
	return &auth.AuthToken{Token: a.TestToken, UserId: userId, DeviceId: deviceId, Scope: scope}, nil
}

type TestStoreFunctions struct {
	SaveToken     bool
	GetToken      bool
	GetUserId     bool
	CreateAccount bool
	SetWallet     bool
	GetWallet     bool
}

type TestStore struct {
	// Fake store functions will set these to `true` as they are called
	Called TestStoreFunctions

	// Fake store functions will fail if these are set to `true` by the test
	// setup
	Failures TestStoreFunctions
}

func (s *TestStore) SaveToken(token *auth.AuthToken) error {
	if s.Failures.SaveToken {
		return fmt.Errorf("TestStore.SaveToken fail")
	}
	s.Called.SaveToken = true
	return nil
}

func (s *TestStore) GetToken(auth.TokenString) (*auth.AuthToken, error) {
	if s.Failures.GetToken {
		return nil, fmt.Errorf("TestStore.GetToken fail")
	}
	s.Called.GetToken = true
	return nil, nil
}

func (s *TestStore) GetUserId(auth.Email, auth.Password) (auth.UserId, error) {
	if s.Failures.GetUserId {
		return 0, store.ErrNoUId
	}
	s.Called.GetUserId = true
	return 0, nil
}

func (s *TestStore) CreateAccount(auth.Email, auth.Password) error {
	if s.Failures.CreateAccount {
		return fmt.Errorf("TestStore.CreateAccount fail")
	}
	s.Called.CreateAccount = true
	return nil
}

func (s *TestStore) SetWallet(
	UserId auth.UserId,
	encryptedWallet wallet.EncryptedWallet,
	sequence wallet.Sequence,
	hmac wallet.WalletHmac,
) (latestEncryptedWallet wallet.EncryptedWallet, latestSequence wallet.Sequence, latestHmac wallet.WalletHmac, sequenceCorrect bool, err error) {
	if s.Failures.SetWallet {
		err = fmt.Errorf("TestStore.SetWallet fail")
		return
	}
	s.Called.SetWallet = true
	return
}

func (s *TestStore) GetWallet(userId auth.UserId) (encryptedWallet wallet.EncryptedWallet, sequence wallet.Sequence, hmac wallet.WalletHmac, err error) {
	if s.Failures.GetWallet {
		err = fmt.Errorf("TestStore.GetWallet fail")
		return
	}
	s.Called.GetWallet = true
	return
}

func TestServerHelperCheckAuthSuccess(t *testing.T) {
	t.Fatalf("Test me: checkAuth success")
}

func TestServerHelperCheckAuthErrors(t *testing.T) {
	t.Fatalf("Test me: checkAuth failure")
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
			requestBody:         fmt.Sprintf(`{"key": "%s"}`, strings.Repeat("a", 10000)),
			expectedStatusCode:  http.StatusRequestEntityTooLarge,
			expectedErrorString: http.StatusText(http.StatusRequestEntityTooLarge),
		},
		{
			name:                "malformed request body JSON",
			method:              http.MethodPost,
			requestBody:         "{",
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrorString: http.StatusText(http.StatusBadRequest) + ": Request body JSON malformed or structure mismatch",
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

func TestServerHelperRequestOverheadSuccess(t *testing.T) {
	t.Fatalf("Test me: requestOverhead success")
}
func TestServerHelperRequestOverheadErrors(t *testing.T) {
	t.Fatalf("Test me: requestOverhead failures")
}
