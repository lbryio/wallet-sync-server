package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"lbryio/lbry-id/auth"
	"lbryio/lbry-id/store"
)

func TestServerRegisterSuccess(t *testing.T) {
	testStore := &TestStore{}
	env := map[string]string{
		"ACCOUNT_VERIFICATION_MODE": "AllowAll",
	}
	s := Server{&TestAuth{}, testStore, &TestEnv{env}}

	requestBody := []byte(`{"email": "abc@example.com", "password": "123", "clientSaltSeed": "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234" }`)

	req := httptest.NewRequest(http.MethodPost, PathRegister, bytes.NewBuffer(requestBody))
	w := httptest.NewRecorder()

	s.register(w, req)
	body, _ := ioutil.ReadAll(w.Body)

	expectStatusCode(t, w, http.StatusCreated)

	var result RegisterResponse
	err := json.Unmarshal(body, &result)

	expectedResponse := RegisterResponse{Verified: true}
	if err != nil || !reflect.DeepEqual(&result, &expectedResponse) {
		t.Errorf("Unexpected value for register response. Want: %+v Got: %+v Err: %+v", expectedResponse, result, err)
	}

	if testStore.Called.CreateAccount == nil {
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

			env := map[string]string{
				"ACCOUNT_VERIFICATION_MODE": "AllowAll",
			}

			// Set this up to fail according to specification
			s := Server{&TestAuth{}, &TestStore{Errors: tc.storeErrors}, &TestEnv{env}}

			// Make request
			requestBody := fmt.Sprintf(`{"email": "%s", "password": "123", "clientSaltSeed": "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234"}`, tc.email)
			req := httptest.NewRequest(http.MethodPost, PathAuthToken, bytes.NewBuffer([]byte(requestBody)))
			w := httptest.NewRecorder()

			s.register(w, req)

			body, _ := ioutil.ReadAll(w.Body)

			expectStatusCode(t, w, tc.expectedStatusCode)
			expectErrorString(t, body, tc.expectedErrorString)
		})
	}
}

func TestServerRegisterAccountVerification(t *testing.T) {
	tt := []struct {
		name string

		env                map[string]string
		expectSuccess      bool
		expectedVerified   bool
		expectedStatusCode int
	}{
		{
			name: "allow all",

			env: map[string]string{
				"ACCOUNT_VERIFICATION_MODE": "AllowAll",
			},

			expectedVerified:   true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusCreated,
		},
		{
			name: "whitelist allowed",

			env: map[string]string{
				"ACCOUNT_VERIFICATION_MODE": "Whitelist",
				"ACCOUNT_WHITELIST":         "abc@example.com",
			},

			expectedVerified:   true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusCreated,
		},
		{
			name: "whitelist disallowed",

			env: map[string]string{
				"ACCOUNT_VERIFICATION_MODE": "Whitelist",
				"ACCOUNT_WHITELIST":         "something-else@example.com",
			},

			expectedVerified:   false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name: "email verify",

			env: map[string]string{
				"ACCOUNT_VERIFICATION_MODE": "EmailVerify",
			},

			expectedVerified:   false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusCreated,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			testStore := &TestStore{}
			s := Server{&TestAuth{}, testStore, &TestEnv{tc.env}}

			requestBody := []byte(`{"email": "abc@example.com", "password": "123", "clientSaltSeed": "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234" }`)

			req := httptest.NewRequest(http.MethodPost, PathRegister, bytes.NewBuffer(requestBody))
			w := httptest.NewRecorder()

			s.register(w, req)
			body, _ := ioutil.ReadAll(w.Body)

			expectStatusCode(t, w, tc.expectedStatusCode)

			if tc.expectSuccess {
				if testStore.Called.CreateAccount == nil {
					t.Fatalf("Expected CreateAccount to be called")
				}
				if tc.expectedVerified != testStore.Called.CreateAccount.Verified {
					t.Errorf("Unexpected value in call to CreateAccount for `verified`. Want: %+v Got: %+v", tc.expectedVerified, testStore.Called.CreateAccount.Verified)
				}
				var result RegisterResponse
				err := json.Unmarshal(body, &result)

				if err != nil || tc.expectedVerified != result.Verified {
					t.Errorf("Unexpected value in register response for `verified`. Want: %+v Got: %+v Err: %+v", tc.expectedVerified, result.Verified, err)
				}
			} else {
				if testStore.Called.CreateAccount != nil {
					t.Errorf("Expected CreateAccount not to be called")
				}
			}

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

func TestServerVerifyAccountSuccess(t *testing.T) {
	testStore := TestStore{TestVerifyTokenString: "abcd1234abcd1234abcd1234abcd1234"}
	s := Server{&TestAuth{}, &testStore, &TestEnv{}}

	req := httptest.NewRequest(http.MethodGet, PathVerify, nil)
	q := req.URL.Query()
	q.Add("verifyToken", string(testStore.TestVerifyTokenString))
	req.URL.RawQuery = q.Encode()
	w := httptest.NewRecorder()

	s.verify(w, req)
	body, _ := ioutil.ReadAll(w.Body)

	expectStatusCode(t, w, http.StatusOK)

	if string(body) != "{}" {
		t.Errorf("Expected register response to be \"{}\": result: %+v", string(body))
	}

	if !testStore.Called.VerifyAccount {
		t.Errorf("Expected Store.VerifyAccount to be called")
	}
}

func TestServerVerifyAccountErrors(t *testing.T) {
	tt := []struct {
		name                      string
		token                     auth.VerifyTokenString
		expectedStatusCode        int
		expectedErrorString       string
		expectedCallVerifyAccount bool

		storeErrors TestStoreFunctionsErrors
	}{
		{
			name:                      "missing token",
			token:                     "",
			expectedStatusCode:        http.StatusBadRequest,
			expectedErrorString:       http.StatusText(http.StatusBadRequest) + ": Missing verifyToken parameter",
			expectedCallVerifyAccount: false,
		},
		{
			name:                      "not found token", // including expired
			token:                     "abcd1234abcd1234abcd1234abcd1234",
			expectedStatusCode:        http.StatusForbidden,
			expectedErrorString:       http.StatusText(http.StatusForbidden) + ": Verification token not found or expired",
			storeErrors:               TestStoreFunctionsErrors{VerifyAccount: store.ErrNoTokenForUser},
			expectedCallVerifyAccount: true,
		},
		{
			name:                      "assorted db error",
			token:                     "abcd1234abcd1234abcd1234abcd1234",
			expectedStatusCode:        http.StatusInternalServerError,
			expectedErrorString:       http.StatusText(http.StatusInternalServerError),
			storeErrors:               TestStoreFunctionsErrors{VerifyAccount: fmt.Errorf("TestStore.VerifyAccount fail")},
			expectedCallVerifyAccount: true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			// Set this up to fail according to specification
			testStore := TestStore{Errors: tc.storeErrors, TestVerifyTokenString: tc.token}
			s := Server{&TestAuth{}, &testStore, &TestEnv{}}

			// Make request
			req := httptest.NewRequest(http.MethodGet, PathVerify, nil)
			q := req.URL.Query()
			q.Add("verifyToken", string(testStore.TestVerifyTokenString))
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()

			s.verify(w, req)
			body, _ := ioutil.ReadAll(w.Body)

			expectStatusCode(t, w, tc.expectedStatusCode)
			expectErrorString(t, body, tc.expectedErrorString)

			if tc.expectedCallVerifyAccount != testStore.Called.VerifyAccount {
				t.Errorf("Expected Store.VerifyAccount not to be called")
			}
		})
	}
}
