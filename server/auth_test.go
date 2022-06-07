package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/wallet"
	"strings"
	"testing"
)

func TestServerAuthHandlerSuccess(t *testing.T) {
	testAuth := TestAuth{TestToken: auth.AuthTokenString("seekrit")}
	testStore := TestStore{}
	s := Server{&testAuth, &testStore, &wallet.WalletUtil{}}

	requestBody := []byte(`{"deviceId": "dev-1", "email": "abc@example.com", "password": "123"}`)

	req := httptest.NewRequest(http.MethodPost, PathAuthTokenFull, bytes.NewBuffer(requestBody))
	w := httptest.NewRecorder()

	s.getAuthTokenFull(w, req)
	body, _ := ioutil.ReadAll(w.Body)

	var result auth.AuthToken

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

		authFailLogin    bool
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
			requestBody:         fmt.Sprintf(`{"password": "%s"}`, strings.Repeat("a", 10000)),
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
			name:                "body JSON failed validation",
			method:              http.MethodPost,
			requestBody:         "{}",
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrorString: http.StatusText(http.StatusBadRequest) + ": Request failed validation",
		},
		{
			name:   "login fail",
			method: http.MethodPost,
			// so long as the JSON is well-formed, the content doesn't matter here since the password check will be stubbed out
			requestBody:         `{"deviceId": "dev-1", "email": "abc@example.com", "password": "123"}`,
			expectedStatusCode:  http.StatusUnauthorized,
			expectedErrorString: http.StatusText(http.StatusUnauthorized) + ": No match for email and password",

			authFailLogin: true,
		},
		{
			name:                "generate token fail",
			method:              http.MethodPost,
			requestBody:         `{"deviceId": "dev-1", "email": "abc@example.com", "password": "123"}`,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrorString: http.StatusText(http.StatusInternalServerError),

			authFailGenToken: true,
		},
		{
			name:                "save token fail",
			method:              http.MethodPost,
			requestBody:         `{"deviceId": "dev-1", "email": "abc@example.com", "password": "123"}`,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrorString: http.StatusText(http.StatusInternalServerError),

			storeFailSave: true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			// Set this up to fail according to specification
			testAuth := TestAuth{TestToken: auth.AuthTokenString("seekrit")}
			testStore := TestStore{}
			if tc.authFailLogin {
				testStore.FailLogin = true
			} else if tc.authFailGenToken {
				testAuth.FailGenToken = true
			} else if tc.storeFailSave {
				testStore.FailSave = true
			} else {
				testAuth.TestToken = auth.AuthTokenString("seekrit")
			}
			server := Server{&testAuth, &testStore, &wallet.WalletUtil{}}

			// Make request
			req := httptest.NewRequest(tc.method, PathAuthTokenFull, bytes.NewBuffer([]byte(tc.requestBody)))
			w := httptest.NewRecorder()

			server.getAuthTokenFull(w, req)

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

func TestServerValidateAuthFullRequest(t *testing.T) {
	t.Fatalf("Test me: Implement and test AuthFullRequest.validate()")
}

func TestServerValidateAuthForGetWalletStateRequest(t *testing.T) {
	t.Fatalf("Test me: Implement and test AuthForGetWalletStateRequest.validate()")
}

func TestServerAuthHandlerForGetWalletStateSuccess(t *testing.T) {
	t.Fatalf("Test me: getAuthTokenForGetWalletState success")
}

func TestServerAuthHandlerForGetWalletStateErrors(t *testing.T) {
	t.Fatalf("Test me: getAuthTokenForGetWalletState failure")
}
