package server

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"orblivion/lbry-id/auth"
	"testing"
)

func TestServerAuthHandlerSuccess(t *testing.T) {
	testAuth := TestAuth{TestToken: auth.TokenString("seekrit")}
	testStore := TestStore{}
	s := Server{&testAuth, &testStore}

	requestBody := []byte(`{"deviceId": "dev-1", "email": "abc@example.com", "password": "123"}`)

	req := httptest.NewRequest(http.MethodPost, PathAuthToken, bytes.NewBuffer(requestBody))
	w := httptest.NewRecorder()

	s.getAuthToken(w, req)
	body, _ := ioutil.ReadAll(w.Body)

	var result auth.AuthToken

	if want, got := http.StatusOK, w.Result().StatusCode; want != got {
		t.Errorf("StatusCode: expected %s (%d), got %s (%d)", http.StatusText(want), want, http.StatusText(got), got)
	}

	err := json.Unmarshal(body, &result)

	if err != nil || result.Token != testAuth.TestToken {
		t.Errorf("Expected auth response to contain token: result: %+v err: %+v", string(body), err)
	}

	if !testStore.Called.SaveToken {
		t.Errorf("Expected Store.SaveToken to be called")
	}
}

func TestServerAuthHandlerErrors(t *testing.T) {
	tt := []struct {
		name                string
		expectedStatusCode  int
		expectedErrorString string

		storeFailures    TestStoreFunctions
		authFailGenToken bool
	}{
		{
			name: "login fail",
			// so long as the JSON is well-formed, the content doesn't matter here since the password check will be stubbed out
			expectedStatusCode:  http.StatusUnauthorized,
			expectedErrorString: http.StatusText(http.StatusUnauthorized) + ": No match for email and password",

			storeFailures: TestStoreFunctions{GetUserId: true},
		},
		{
			name:                "generate token fail",
			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrorString: http.StatusText(http.StatusInternalServerError),

			authFailGenToken: true,
		},
		{
			name:                "save token fail",
			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrorString: http.StatusText(http.StatusInternalServerError),

			storeFailures: TestStoreFunctions{SaveToken: true},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			// Set this up to fail according to specification
			testAuth := TestAuth{TestToken: auth.TokenString("seekrit")}
			testStore := TestStore{Failures: tc.storeFailures}
			if tc.authFailGenToken { // TODO - TestAuth{Failures:authFailures}
				testAuth.FailGenToken = true
			}
			server := Server{&testAuth, &testStore}

			// Make request
			requestBody := `{"deviceId": "dev-1", "email": "abc@example.com", "password": "123"}`
			req := httptest.NewRequest(http.MethodPost, PathAuthToken, bytes.NewBuffer([]byte(requestBody)))
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
	authRequest := AuthRequest{DeviceId: "dId", Email: "joe@example.com", Password: "aoeu"}
	if !authRequest.validate() {
		t.Fatalf("Expected valid AuthRequest to successfully validate")
	}

	authRequest = AuthRequest{Email: "joe@example.com", Password: "aoeu"}
	if authRequest.validate() {
		t.Fatalf("Expected AuthRequest with missing device to not successfully validate")
	}

	authRequest = AuthRequest{DeviceId: "dId", Email: "joe-example.com", Password: "aoeu"}
	if authRequest.validate() {
		t.Fatalf("Expected AuthRequest with invalid email to not successfully validate")
	}

	// Note that Golang's email address parser, which I use, will accept
	// "Joe <joe@example.com>" so we need to make sure to avoid accepting it. See
	// the implementation.
	authRequest = AuthRequest{DeviceId: "dId", Email: "Joe <joe@example.com>", Password: "aoeu"}
	if authRequest.validate() {
		t.Fatalf("Expected AuthRequest with email with unexpected formatting to not successfully validate")
	}

	authRequest = AuthRequest{DeviceId: "dId", Password: "aoeu"}
	if authRequest.validate() {
		t.Fatalf("Expected AuthRequest with missing email to not successfully validate")
	}

	authRequest = AuthRequest{DeviceId: "dId", Email: "joe@example.com"}
	if authRequest.validate() {
		t.Fatalf("Expected AuthRequest with missing password to not successfully validate")
	}
}
