package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lbryio/lbry-id/store"
)

func TestServerRegisterSuccess(t *testing.T) {
	testAuth := TestAuth{}
	testStore := TestStore{}
	s := Server{&testAuth, &testStore}

	requestBody := []byte(`{"email": "abc@example.com", "password": "123", "clientSaltSeed": "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234" }`)

	req := httptest.NewRequest(http.MethodPost, PathRegister, bytes.NewBuffer(requestBody))
	w := httptest.NewRecorder()

	s.register(w, req)
	body, _ := ioutil.ReadAll(w.Body)

	expectStatusCode(t, w, http.StatusCreated)

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
		email               string
		expectedStatusCode  int
		expectedErrorString string

		storeErrors TestStoreFunctionsErrors
	}{
		{
			name:                "validation error", // missing email address
			email:               "",
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrorString: http.StatusText(http.StatusBadRequest) + ": Request failed validation: Invalid or missing 'email'",

			// Just check one validation error (missing email address) to make sure the
			// validate function is called. We'll check the rest of the validation
			// errors in the other test below.
		},
		{
			name:                "existing account",
			email:               "abc@example.com",
			expectedStatusCode:  http.StatusConflict,
			expectedErrorString: http.StatusText(http.StatusConflict) + ": Error registering",

			storeErrors: TestStoreFunctionsErrors{CreateAccount: store.ErrDuplicateEmail},
		},
		{
			name:                "unspecified account creation failure",
			email:               "abc@example.com",
			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrorString: http.StatusText(http.StatusInternalServerError),

			storeErrors: TestStoreFunctionsErrors{CreateAccount: fmt.Errorf("TestStore.CreateAccount fail")},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			// Set this up to fail according to specification
			testAuth := TestAuth{}
			testStore := TestStore{Errors: tc.storeErrors}
			server := Server{&testAuth, &testStore}

			// Make request
			requestBody := fmt.Sprintf(`{"email": "%s", "password": "123", "clientSaltSeed": "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234"}`, tc.email)
			req := httptest.NewRequest(http.MethodPost, PathAuthToken, bytes.NewBuffer([]byte(requestBody)))
			w := httptest.NewRecorder()

			server.register(w, req)

			body, _ := ioutil.ReadAll(w.Body)

			expectStatusCode(t, w, tc.expectedStatusCode)
			expectErrorString(t, body, tc.expectedErrorString)
		})
	}
}

func TestServerValidateRegisterRequest(t *testing.T) {
	registerRequest := RegisterRequest{Email: "joe@example.com", Password: "aoeu", ClientSaltSeed: "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234"}
	if registerRequest.validate() != nil {
		t.Errorf("Expected valid RegisterRequest to successfully validate")
	}

	registerRequest = RegisterRequest{Email: "joe-example.com", Password: "aoeu", ClientSaltSeed: "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234"}
	err := registerRequest.validate()
	if !strings.Contains(err.Error(), "email") {
		t.Errorf("Expected RegisterRequest with invalid email to return an appropriate error")
	}

	// Note that Golang's email address parser, which I use, will accept
	// "Joe <joe@example.com>" so we need to make sure to avoid accepting it. See
	// the implementation.
	registerRequest = RegisterRequest{Email: "Joe <joe@example.com>", Password: "aoeu", ClientSaltSeed: "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234"}
	err = registerRequest.validate()
	if !strings.Contains(err.Error(), "email") {
		t.Errorf("Expected RegisterRequest with email with unexpected formatting to return an appropriate error")
	}

	registerRequest = RegisterRequest{Password: "aoeu", ClientSaltSeed: "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234"}
	err = registerRequest.validate()
	if !strings.Contains(err.Error(), "email") {
		t.Errorf("Expected RegisterRequest with missing email to return an appropriate error")
	}

	registerRequest = RegisterRequest{Email: "joe@example.com", ClientSaltSeed: "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234"}
	err = registerRequest.validate()
	if !strings.Contains(err.Error(), "password") {
		t.Errorf("Expected RegisterRequest with missing password to return an appropriate error")
	}

	registerRequest = RegisterRequest{Email: "joe@example.com", Password: "aoeu"}
	err = registerRequest.validate()
	if !strings.Contains(err.Error(), "clientSaltSeed") {
		t.Errorf("Expected RegisterRequest with missing clientSaltSeed to return an appropriate error")
	}

	registerRequest = RegisterRequest{Email: "joe@example.com", Password: "aoeu", ClientSaltSeed: "abcd1234abcd1234abcd1234abcd1234"}
	err = registerRequest.validate()
	if !strings.Contains(err.Error(), "clientSaltSeed") {
		t.Errorf("Expected RegisterRequest with clientSaltSeed of wrong length to return an appropriate error")
	}

	registerRequest = RegisterRequest{Email: "joe@example.com", Password: "aoeu", ClientSaltSeed: "xxxx1234xxxx1234xxxx1234xxxx1234xxxx1234xxxx1234xxxx1234xxxx1234"}
	err = registerRequest.validate()
	if !strings.Contains(err.Error(), "clientSaltSeed") {
		t.Errorf("Expected RegisterRequest with clientSaltSeed with a non-hex string to return an appropriate error")
	}
}
