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

	"lbryio/wallet-sync-server/server/paths"
	"lbryio/wallet-sync-server/store"
)

// TODO - maybe this test could just be one of the TestServerRegisterAccountVerification tests now
func TestServerRegisterSuccess(t *testing.T) {
	testStore := &TestStore{}
	env := map[string]string{
		"ACCOUNT_VERIFICATION_MODE": "EmailVerify",
	}
	testMail := TestMail{}
	testAuth := TestAuth{TestNewVerifyTokenString: "abcd1234abcd1234abcd1234abcd1234"}
	s := Server{&testAuth, testStore, &TestEnv{env}, &testMail, TestPort}

	requestBody := []byte(`{"email": "abc@example.com", "password": "123", "clientSaltSeed": "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234" }`)

	req := httptest.NewRequest(http.MethodPost, paths.PathRegister, bytes.NewBuffer(requestBody))
	w := httptest.NewRecorder()

	s.register(w, req)
	body, _ := ioutil.ReadAll(w.Body)

	expectStatusCode(t, w, http.StatusCreated)

	var result RegisterResponse
	err := json.Unmarshal(body, &result)

	expectedResponse := RegisterResponse{Verified: false}
	if err != nil || result != expectedResponse {
		t.Errorf("Unexpected value for register response. Want: %+v Got: %+v Err: %+v", expectedResponse, result, err)
	}

	if testStore.Called.CreateAccount == nil {
		t.Errorf("Expected Store.CreateAccount to be called")
	}

	if testMail.SendVerificationEmailCall == nil {
		// We're doing EmailVerify for this test.
		t.Fatalf("Expected Store.SendVerificationEmail to be called")
	}
}

func TestServerRegisterErrors(t *testing.T) {
	tt := []struct {
		name                              string
		email                             string
		expectedStatusCode                int
		expectedErrorString               string
		expectedCallSendVerificationEmail bool
		expectedCallCreateAccount         bool

		storeErrors  TestStoreFunctionsErrors
		mailError    error
		failGenToken bool
	}{
		{
			name:                              "validation error", // missing email address
			email:                             "",
			expectedStatusCode:                http.StatusBadRequest,
			expectedErrorString:               http.StatusText(http.StatusBadRequest) + ": Request failed validation: Invalid or missing 'email'",
			expectedCallSendVerificationEmail: false,
			expectedCallCreateAccount:         false,

			// Just check one validation error (missing email address) to make sure the
			// validate function is called. We'll check the rest of the validation
			// errors in the other test below.
		},
		{
			name:                              "existing account",
			email:                             "abc@example.com",
			expectedStatusCode:                http.StatusConflict,
			expectedErrorString:               http.StatusText(http.StatusConflict) + ": Error registering",
			expectedCallSendVerificationEmail: false,
			expectedCallCreateAccount:         true,

			storeErrors: TestStoreFunctionsErrors{CreateAccount: store.ErrDuplicateEmail},
		},
		{
			name:                              "unspecified account creation failure",
			email:                             "abc@example.com",
			expectedStatusCode:                http.StatusInternalServerError,
			expectedErrorString:               http.StatusText(http.StatusInternalServerError),
			expectedCallSendVerificationEmail: false,
			expectedCallCreateAccount:         true,

			storeErrors: TestStoreFunctionsErrors{CreateAccount: fmt.Errorf("TestStore.CreateAccount fail")},
		},
		{
			name:                              "fail to generate verifiy token",
			email:                             "abc@example.com",
			expectedStatusCode:                http.StatusInternalServerError,
			expectedErrorString:               http.StatusText(http.StatusInternalServerError),
			expectedCallSendVerificationEmail: false,
			expectedCallCreateAccount:         false,

			failGenToken: true,
		},
		{
			name:                              "fail to generate verification email",
			email:                             "abc@example.com",
			expectedStatusCode:                http.StatusInternalServerError,
			expectedErrorString:               http.StatusText(http.StatusInternalServerError),
			expectedCallSendVerificationEmail: true,
			expectedCallCreateAccount:         true,

			mailError: fmt.Errorf("TestEmail.SendVerificationEmail fail"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			env := map[string]string{
				"ACCOUNT_VERIFICATION_MODE": "EmailVerify",
			}

			// Set this up to fail according to specification
			testAuth := TestAuth{TestNewVerifyTokenString: "abcd1234abcd1234abcd1234abcd1234", FailGenToken: tc.failGenToken}
			testMail := TestMail{SendVerificationEmailError: tc.mailError}
			testStore := TestStore{Errors: tc.storeErrors}
			s := Server{&testAuth, &testStore, &TestEnv{env}, &testMail, TestPort}

			// Make request
			requestBody := fmt.Sprintf(`{"email": "%s", "password": "123", "clientSaltSeed": "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234"}`, tc.email)
			req := httptest.NewRequest(http.MethodPost, paths.PathAuthToken, bytes.NewBuffer([]byte(requestBody)))
			w := httptest.NewRecorder()

			s.register(w, req)

			body, _ := ioutil.ReadAll(w.Body)

			expectStatusCode(t, w, tc.expectedStatusCode)
			expectErrorString(t, body, tc.expectedErrorString)

			if tc.expectedCallCreateAccount && testStore.Called.CreateAccount == nil {
				t.Errorf("Expected Store.CreateAccount to be called")
			}
			if !tc.expectedCallCreateAccount && testStore.Called.CreateAccount != nil {
				t.Errorf("Expected Store.CreateAccount not to be called")
			}

			if tc.expectedCallSendVerificationEmail && testMail.SendVerificationEmailCall == nil {
				t.Errorf("Expected Store.SendVerificationEmail to be called")
			}
			if !tc.expectedCallSendVerificationEmail && testMail.SendVerificationEmailCall != nil {
				t.Errorf("Expected Store.SendVerificationEmail not to be called")
			}
		})
	}
}

func TestServerRegisterAccountVerification(t *testing.T) {
	tt := []struct {
		name string

		env                               map[string]string
		expectSuccess                     bool
		expectedVerified                  bool
		expectedStatusCode                int
		expectedCallSendVerificationEmail bool
	}{
		{
			name: "allow all",

			env: map[string]string{
				"ACCOUNT_VERIFICATION_MODE": "AllowAll",
			},

			expectedVerified:                  true,
			expectSuccess:                     true,
			expectedStatusCode:                http.StatusCreated,
			expectedCallSendVerificationEmail: false,
		},
		{
			name: "whitelist allowed",

			env: map[string]string{
				"ACCOUNT_VERIFICATION_MODE": "Whitelist",
				"ACCOUNT_WHITELIST":         "abc@example.com",
			},

			expectedVerified:                  true,
			expectSuccess:                     true,
			expectedStatusCode:                http.StatusCreated,
			expectedCallSendVerificationEmail: false,
		},
		{
			name: "whitelist disallowed",

			env: map[string]string{
				"ACCOUNT_VERIFICATION_MODE": "Whitelist",
				"ACCOUNT_WHITELIST":         "something-else@example.com",
			},

			expectedVerified:                  false,
			expectSuccess:                     false,
			expectedStatusCode:                http.StatusForbidden,
			expectedCallSendVerificationEmail: false,
		},
		{
			name: "email verify",

			env: map[string]string{
				"ACCOUNT_VERIFICATION_MODE": "EmailVerify",
			},

			expectedVerified:                  false,
			expectSuccess:                     true,
			expectedStatusCode:                http.StatusCreated,
			expectedCallSendVerificationEmail: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			testStore := &TestStore{}
			testAuth := TestAuth{TestNewVerifyTokenString: "abcd1234abcd1234abcd1234abcd1234"}
			testMail := TestMail{}
			s := Server{&testAuth, testStore, &TestEnv{tc.env}, &testMail, TestPort}

			requestBody := []byte(`{"email": "abc@example.com", "password": "123", "clientSaltSeed": "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234" }`)

			req := httptest.NewRequest(http.MethodPost, paths.PathRegister, bytes.NewBuffer(requestBody))
			w := httptest.NewRecorder()

			s.register(w, req)
			body, _ := ioutil.ReadAll(w.Body)

			expectStatusCode(t, w, tc.expectedStatusCode)

			if tc.expectSuccess {
				if testStore.Called.CreateAccount == nil {
					t.Fatalf("Expected CreateAccount to be called")
				}
				tokenPassedIn := testStore.Called.CreateAccount.VerifyToken != nil
				if tc.expectedVerified && tokenPassedIn {
					t.Errorf("Expected new account to be verified, thus expected verifyToken *not to be passed in* to call to CreateAccount.")
				}
				if !tc.expectedVerified && !tokenPassedIn {
					t.Errorf("Expected new account not to be verified, thus expected verifyToken not *to be passed in* to call to CreateAccount.")
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

			if tc.expectedCallSendVerificationEmail && testMail.SendVerificationEmailCall == nil {
				t.Errorf("Expected Store.SendVerificationEmail to be called")
			}
			if !tc.expectedCallSendVerificationEmail && testMail.SendVerificationEmailCall != nil {
				t.Errorf("Expected Store.SendVerificationEmail not to be called")
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

func TestServerResendVerifyEmailSuccess(t *testing.T) {
	testStore := TestStore{}
	testMail := TestMail{}

	env := map[string]string{
		"ACCOUNT_VERIFICATION_MODE": "EmailVerify",
	}
	s := Server{&TestAuth{}, &testStore, &TestEnv{env}, &testMail, TestPort}

	requestBody := []byte(`{"email": "abc@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, paths.PathVerify, bytes.NewBuffer(requestBody))
	w := httptest.NewRecorder()

	s.resendVerifyEmail(w, req)
	body, _ := ioutil.ReadAll(w.Body)

	expectStatusCode(t, w, http.StatusOK)

	if string(body) != "{}" {
		t.Errorf("Expected register response to be \"{}\": result: %+v", string(body))
	}

	if !testStore.Called.UpdateVerifyTokenString {
		t.Errorf("Expected Store.UpdateVerifyTokenString to be called")
	}

	if testMail.SendVerificationEmailCall == nil {
		// We're doing EmailVerify for this test.
		t.Fatalf("Expected Store.SendVerificationEmail to be called")
	}
}

func TestServerResendVerifyEmailErrors(t *testing.T) {
	tt := []struct {
		name                    string
		omitEmailAddress        bool
		accountVerificationMode string

		expectedStatusCode                  int
		expectedErrorString                 string
		expectedCallSendVerificationEmail   bool
		expectedCallUpdateVerifyTokenString bool

		storeErrors TestStoreFunctionsErrors
		mailError   error
	}{

		{
			name:                                "wrong account verification mode",
			accountVerificationMode:             "Whitelist",
			expectedStatusCode:                  http.StatusForbidden,
			expectedErrorString:                 http.StatusText(http.StatusForbidden) + ": Account verification mode is not set to EmailVerify",
			expectedCallSendVerificationEmail:   false,
			expectedCallUpdateVerifyTokenString: false,
		},
		{
			name:                                "validation error",
			accountVerificationMode:             "EmailVerify",
			omitEmailAddress:                    true,
			expectedStatusCode:                  http.StatusBadRequest,
			expectedErrorString:                 http.StatusText(http.StatusBadRequest) + ": Request failed validation: Invalid or missing 'email'",
			expectedCallSendVerificationEmail:   false,
			expectedCallUpdateVerifyTokenString: false,
		},

		{
			name:                                "not found email",
			accountVerificationMode:             "EmailVerify",
			expectedStatusCode:                  http.StatusUnauthorized,
			expectedErrorString:                 http.StatusText(http.StatusUnauthorized) + ": No match for email",
			storeErrors:                         TestStoreFunctionsErrors{UpdateVerifyTokenString: store.ErrWrongCredentials},
			expectedCallSendVerificationEmail:   false,
			expectedCallUpdateVerifyTokenString: true,
		},
		{
			name:                                "assorted db error",
			accountVerificationMode:             "EmailVerify",
			expectedStatusCode:                  http.StatusInternalServerError,
			expectedErrorString:                 http.StatusText(http.StatusInternalServerError),
			storeErrors:                         TestStoreFunctionsErrors{UpdateVerifyTokenString: fmt.Errorf("TestStore.UpdateVerifyTokenString fail")},
			expectedCallSendVerificationEmail:   false,
			expectedCallUpdateVerifyTokenString: true,
		},
		{
			name:                                "fail to generate verification email",
			accountVerificationMode:             "EmailVerify",
			expectedStatusCode:                  http.StatusInternalServerError,
			expectedErrorString:                 http.StatusText(http.StatusInternalServerError),
			expectedCallSendVerificationEmail:   true,
			expectedCallUpdateVerifyTokenString: true,

			mailError: fmt.Errorf("TestEmail.SendVerificationEmail fail"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			env := map[string]string{
				"ACCOUNT_VERIFICATION_MODE": tc.accountVerificationMode,
			}

			// Set this up to fail according to specification
			testStore := TestStore{Errors: tc.storeErrors}
			testMail := TestMail{SendVerificationEmailError: tc.mailError}
			s := Server{&TestAuth{}, &testStore, &TestEnv{env}, &testMail, TestPort}

			// Make request
			var requestBody []byte
			if tc.omitEmailAddress {
				requestBody = []byte(`{}`)
			} else {
				requestBody = []byte(`{"email": "abc@example.com"}`)
			}
			req := httptest.NewRequest(http.MethodPost, paths.PathVerify, bytes.NewBuffer(requestBody))
			w := httptest.NewRecorder()

			s.resendVerifyEmail(w, req)
			body, _ := ioutil.ReadAll(w.Body)

			expectStatusCode(t, w, tc.expectedStatusCode)
			expectErrorString(t, body, tc.expectedErrorString)

			if tc.expectedCallUpdateVerifyTokenString && !testStore.Called.UpdateVerifyTokenString {
				t.Errorf("Expected Store.UpdateVerifyTokenString to be called")
			}
			if !tc.expectedCallUpdateVerifyTokenString && testStore.Called.UpdateVerifyTokenString {
				t.Errorf("Expected Store.UpdateVerifyTokenString not to be called")
			}

			if tc.expectedCallSendVerificationEmail && testMail.SendVerificationEmailCall == nil {
				// We're doing EmailVerify for this test.
				t.Fatalf("Expected Store.SendVerificationEmail to be called")
			}
			if !tc.expectedCallSendVerificationEmail && testMail.SendVerificationEmailCall != nil {
				// We're doing EmailVerify for this test.
				t.Fatalf("Expected Store.SendVerificationEmail not to be called")
			}
		})
	}
}

func TestServerVerifyAccountSuccess(t *testing.T) {
	testStore := TestStore{}
	s := Server{&TestAuth{}, &testStore, &TestEnv{}, &TestMail{}, TestPort}

	req := httptest.NewRequest(http.MethodGet, paths.PathVerify, nil)
	q := req.URL.Query()
	q.Add("verifyToken", "abcd1234abcd1234abcd1234abcd1234")
	req.URL.RawQuery = q.Encode()
	w := httptest.NewRecorder()

	s.verify(w, req)
	body, _ := ioutil.ReadAll(w.Body)

	expectStatusCode(t, w, http.StatusOK)

	if string(body) != "Your account has been verified." {
		t.Errorf("Expected register response to be \"{}\": result: %+v", string(body))
	}

	if !testStore.Called.VerifyAccount {
		t.Errorf("Expected Store.VerifyAccount to be called")
	}
}

func TestServerVerifyAccountErrors(t *testing.T) {
	tt := []struct {
		name                      string
		token                     string
		expectedStatusCode        int
		expectedBody              string
		expectedCallVerifyAccount bool

		storeErrors TestStoreFunctionsErrors
	}{
		{
			name:                      "missing token",
			token:                     "",
			expectedStatusCode:        http.StatusBadRequest,
			expectedBody:              "There seems to be a problem with this URL: Missing verifyToken parameter",
			expectedCallVerifyAccount: false,
		},
		{
			name:                      "token not found", // including expired
			token:                     "abcd1234abcd1234abcd1234abcd1234",
			expectedStatusCode:        http.StatusForbidden,
			expectedBody:              "The verification token was not found, already used, or expired. If you want to try again, generate a new one from your app.",
			storeErrors:               TestStoreFunctionsErrors{VerifyAccount: store.ErrNoTokenForUser},
			expectedCallVerifyAccount: true,
		},
		{
			name:                      "assorted db error",
			token:                     "abcd1234abcd1234abcd1234abcd1234",
			expectedStatusCode:        http.StatusInternalServerError,
			expectedBody:              "Something went wrong trying to verify your account.",
			storeErrors:               TestStoreFunctionsErrors{VerifyAccount: fmt.Errorf("TestStore.VerifyAccount fail")},
			expectedCallVerifyAccount: true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			// Set this up to fail according to specification
			testStore := TestStore{Errors: tc.storeErrors}
			s := Server{&TestAuth{}, &testStore, &TestEnv{}, &TestMail{}, TestPort}

			// Make request
			req := httptest.NewRequest(http.MethodGet, paths.PathVerify, nil)
			q := req.URL.Query()
			q.Add("verifyToken", tc.token)
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()

			s.verify(w, req)
			body, _ := ioutil.ReadAll(w.Body)

			expectStatusCode(t, w, tc.expectedStatusCode)
			if want, got := tc.expectedBody, strings.TrimSpace(string(body)); want != got {
				t.Errorf("Body: expected `%s`, got `%s`", want, got)
			}

			if tc.expectedCallVerifyAccount && !testStore.Called.VerifyAccount {
				t.Errorf("Expected Store.VerifyAccount to be called")
			}
			if !tc.expectedCallVerifyAccount && testStore.Called.VerifyAccount {
				t.Errorf("Expected Store.VerifyAccount not to be called")
			}
		})
	}
}
