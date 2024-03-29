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

	"lbryio/wallet-sync-server/auth"
	"lbryio/wallet-sync-server/server/paths"
	"lbryio/wallet-sync-server/store"
)

func TestServerAuthHandlerSuccess(t *testing.T) {
	testAuth := TestAuth{TestNewAuthTokenString: auth.AuthTokenString("seekrit")}
	testStore := TestStore{}
	s := Init(&testAuth, &testStore, &TestEnv{}, &TestMail{}, TestPort)

	requestBody := []byte(`{"deviceId": "dev-1", "email": "abc@example.com", "password": "12345678"}`)

	req := httptest.NewRequest(http.MethodPost, paths.PathAuthToken, bytes.NewBuffer(requestBody))
	w := httptest.NewRecorder()

	s.getAuthToken(w, req)
	body, _ := ioutil.ReadAll(w.Body)

	expectStatusCode(t, w, http.StatusOK)

	var result auth.AuthToken
	err := json.Unmarshal(body, &result)

	if err != nil || result.Token != testAuth.TestNewAuthTokenString {
		t.Errorf("Expected auth response to contain token: result: %+v err: %+v", string(body), err)
	}

	if testStore.Called.SaveToken != testAuth.TestNewAuthTokenString {
		t.Errorf("Expected Store.SaveToken to be called with %s", testAuth.TestNewAuthTokenString)
	}
}

func TestServerAuthHandlerErrors(t *testing.T) {
	tt := []struct {
		name                string
		email               string
		expectedStatusCode  int
		expectedErrorString string

		storeErrors      TestStoreFunctionsErrors
		authFailGenToken bool
	}{
		{
			name:                "validation error", // missing email address
			email:               "",
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrorString: http.StatusText(http.StatusBadRequest) + ": Request failed validation: Invalid 'email'",

			// Just check one validation error (missing email address) to make sure the
			// validate function is called. We'll check the rest of the validation
			// errors in the other test below.
		},
		{
			name:                "login fail",
			email:               "abc@example.com",
			expectedStatusCode:  http.StatusUnauthorized,
			expectedErrorString: http.StatusText(http.StatusUnauthorized) + ": No match for email and/or password",

			storeErrors: TestStoreFunctionsErrors{GetUserId: store.ErrWrongCredentials},
		},
		{
			name:                "unverified account",
			email:               "abc@example.com",
			expectedStatusCode:  http.StatusUnauthorized,
			expectedErrorString: http.StatusText(http.StatusUnauthorized) + ": Account is not verified",

			storeErrors: TestStoreFunctionsErrors{GetUserId: store.ErrNotVerified},
		},
		{
			name:                "generate token fail",
			email:               "abc@example.com",
			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrorString: http.StatusText(http.StatusInternalServerError),

			authFailGenToken: true,
		},
		{
			name:                "save token fail",
			email:               "abc@example.com",
			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrorString: http.StatusText(http.StatusInternalServerError),

			storeErrors: TestStoreFunctionsErrors{SaveToken: fmt.Errorf("TestStore.SaveToken fail")},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			// Set this up to fail according to specification
			testAuth := TestAuth{TestNewAuthTokenString: auth.AuthTokenString("seekrit")}
			testStore := TestStore{Errors: tc.storeErrors}
			if tc.authFailGenToken { // TODO - TestAuth{Errors:authErrors}
				testAuth.FailGenToken = true
			}
			server := Init(&testAuth, &testStore, &TestEnv{}, &TestMail{}, TestPort)

			// Make request
			// So long as the JSON is well-formed, the content doesn't matter here since the password check will be stubbed out
			requestBody := fmt.Sprintf(`{"deviceId": "dev-1", "email": "%s", "password": "12345678"}`, tc.email)
			req := httptest.NewRequest(http.MethodPost, paths.PathAuthToken, bytes.NewBuffer([]byte(requestBody)))
			w := httptest.NewRecorder()

			server.getAuthToken(w, req)

			body, _ := ioutil.ReadAll(w.Body)

			expectStatusCode(t, w, tc.expectedStatusCode)
			expectErrorString(t, body, tc.expectedErrorString)
		})
	}
}

func TestServerValidateAuthRequest(t *testing.T) {
	authRequest := AuthRequest{DeviceId: "dId", Email: "joe@example.com", Password: "12345678"}
	if authRequest.validate() != nil {
		t.Errorf("Expected valid AuthRequest to successfully validate")
	}

	tt := []struct {
		authRequest         AuthRequest
		expectedErrorSubstr string
		failureDescription  string
	}{
		{
			AuthRequest{Email: "joe@example.com", Password: "12345678"},
			"deviceId",
			"Expected AuthRequest with missing device to not successfully validate",
		}, {
			AuthRequest{DeviceId: "dId", Email: "joe-example.com", Password: "12345678"},
			"email",
			"Expected AuthRequest with invalid email to not successfully validate",
		}, {
			// Note that Golang's email address parser, which I use, will accept
			// "Joe <joe@example.com>" so we need to make sure to avoid accepting it. See
			// the implementation.
			AuthRequest{DeviceId: "dId", Email: "Joe <joe@example.com>", Password: "12345678"},
			"email",
			"Expected AuthRequest with email with unexpected formatting to not successfully validate",
		}, {
			AuthRequest{DeviceId: "dId", Password: "12345678"},
			"email",
			"Expected AuthRequest with missing email to not successfully validate",
		}, {
			AuthRequest{DeviceId: "dId", Email: "joe@example.com"},
			"password",
			"Expected AuthRequest with missing password to not successfully validate",
		},
	}
	for _, tc := range tt {
		err := tc.authRequest.validate()
		if !strings.Contains(err.Error(), tc.expectedErrorSubstr) {
			t.Errorf(tc.failureDescription)
		}
	}
}
