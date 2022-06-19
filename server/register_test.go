package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/store"
)

func TestServerRegisterSuccess(t *testing.T) {
	testAuth := TestAuth{TestToken: auth.TokenString("seekrit")}
	testStore := TestStore{}
	s := Server{&testAuth, &testStore}

	requestBody := []byte(`{"email": "abc@example.com", "password": "123"}`)

	req := httptest.NewRequest(http.MethodPost, PathRegister, bytes.NewBuffer(requestBody))
	w := httptest.NewRecorder()

	s.register(w, req)
	body, _ := ioutil.ReadAll(w.Body)

	if want, got := http.StatusCreated, w.Result().StatusCode; want != got {
		t.Errorf("StatusCode: expected %s (%d), got %s (%d)", http.StatusText(want), want, http.StatusText(got), got)
	}

	if string(body) != "{}" {
		t.Errorf("Expected register response to be \"{}\": result: %+v", string(body))
	}

	if !testStore.Called.CreateAccount {
		t.Errorf("Expected Store.CreateAccount to be called")
	}
}

func TestServerRegisterErrors(t *testing.T) {
	tt := []struct {
		name                string
		requestBody         string
		expectedStatusCode  int
		expectedErrorString string

		storeErrors TestStoreFunctionsErrors
	}{
		{
			name:                "existing account",
			expectedStatusCode:  http.StatusConflict,
			expectedErrorString: http.StatusText(http.StatusConflict) + ": Error registering",

			storeErrors: TestStoreFunctionsErrors{CreateAccount: store.ErrDuplicateEmail},
		},
		{
			name:                "unspecified account creation failure",
			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrorString: http.StatusText(http.StatusInternalServerError),

			storeErrors: TestStoreFunctionsErrors{CreateAccount: fmt.Errorf("TestStore.CreateAccount fail")},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			// Set this up to fail according to specification
			testAuth := TestAuth{TestToken: auth.TokenString("seekrit")}
			testStore := TestStore{Errors: tc.storeErrors}
			server := Server{&testAuth, &testStore}

			// Make request
			requestBody := `{ "email": "abc@example.com", "password": "123"}`
			req := httptest.NewRequest(http.MethodPost, PathAuthToken, bytes.NewBuffer([]byte(requestBody)))
			w := httptest.NewRecorder()

			server.register(w, req)

			expectErrorResponse(t, w, tc.expectedStatusCode, tc.expectedErrorString)
		})
	}
}

func TestServerValidateRegisterRequest(t *testing.T) {
	registerRequest := RegisterRequest{Email: "joe@example.com", Password: "aoeu"}
	if !registerRequest.validate() {
		t.Fatalf("Expected valid RegisterRequest to successfully validate")
	}

	registerRequest = RegisterRequest{Email: "joe-example.com", Password: "aoeu"}
	if registerRequest.validate() {
		t.Fatalf("Expected RegisterRequest with invalid email to not successfully validate")
	}

	// Note that Golang's email address parser, which I use, will accept
	// "Joe <joe@example.com>" so we need to make sure to avoid accepting it. See
	// the implementation.
	registerRequest = RegisterRequest{Email: "Joe <joe@example.com>", Password: "aoeu"}
	if registerRequest.validate() {
		t.Fatalf("Expected RegisterRequest with email with unexpected formatting to not successfully validate")
	}

	registerRequest = RegisterRequest{Password: "aoeu"}
	if registerRequest.validate() {
		t.Fatalf("Expected RegisterRequest with missing email to not successfully validate")
	}

	registerRequest = RegisterRequest{Email: "joe@example.com"}
	if registerRequest.validate() {
		t.Fatalf("Expected RegisterRequest with missing password to not successfully validate")
	}
}
