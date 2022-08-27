package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"lbryio/wallet-sync-server/auth"
	"lbryio/wallet-sync-server/server/paths"
	"lbryio/wallet-sync-server/store"
)

func TestServerGetClientSalt(t *testing.T) {
	tt := []struct {
		name              string
		emailGetParam     string
		emailCallExpected auth.Email

		expectedStatusCode  int
		expectedErrorString string

		storeErrors TestStoreFunctionsErrors
	}{
		{
			name:               "success",
			emailGetParam:      base64.StdEncoding.EncodeToString([]byte("good@example.com")),
			emailCallExpected:  "good@example.com",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:          "invalid email",
			emailGetParam: base64.StdEncoding.EncodeToString([]byte("bad-example.com")),

			expectedStatusCode:  http.StatusBadRequest,
			expectedErrorString: http.StatusText(http.StatusBadRequest) + ": Invalid email",
		},
		{
			name:                "account not found",
			emailGetParam:       base64.StdEncoding.EncodeToString([]byte("nonexistent@example.com")),
			emailCallExpected:   "nonexistent@example.com",
			expectedStatusCode:  http.StatusNotFound,
			expectedErrorString: http.StatusText(http.StatusNotFound) + ": No match for email",

			storeErrors: TestStoreFunctionsErrors{GetClientSaltSeed: store.ErrWrongCredentials},
		},
		{
			name:                "db error getting client salt",
			emailGetParam:       base64.StdEncoding.EncodeToString([]byte("good@example.com")),
			emailCallExpected:   "good@example.com",
			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrorString: http.StatusText(http.StatusInternalServerError),

			storeErrors: TestStoreFunctionsErrors{GetClientSaltSeed: fmt.Errorf("Some random DB Error!")},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			testAuth := TestAuth{}
			testStore := TestStore{
				TestClientSaltSeed: auth.ClientSaltSeed("abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234"),

				Errors: tc.storeErrors,
			}

			s := Init(&testAuth, &testStore, &TestEnv{}, &TestMail{}, TestPort)

			req := httptest.NewRequest(http.MethodGet, paths.PathClientSaltSeed, nil)
			q := req.URL.Query()
			q.Add("email", string(tc.emailGetParam))
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()

			s.getClientSaltSeed(w, req)

			body, _ := ioutil.ReadAll(w.Body)

			expectStatusCode(t, w, tc.expectedStatusCode)
			expectErrorString(t, body, tc.expectedErrorString)

			// In this case, a salt seed is expected iff there is no error string
			expectSaltSeed := len(tc.expectedErrorString) == 0

			if !expectSaltSeed {
				return // The rest of the test does not apply
			}

			var result ClientSaltSeedResponse
			err := json.Unmarshal(body, &result)

			if err != nil || result.ClientSaltSeed != testStore.TestClientSaltSeed {
				t.Errorf("Expected client salt seed response to have the test client salt secret: result: %+v err: %+v", string(body), err)
			}

			if testStore.Called.GetClientSaltSeed != tc.emailCallExpected {
				t.Errorf("Expected Store.GetClientSaltSeed to be called with %s got %s", tc.emailCallExpected, testStore.Called.GetClientSaltSeed)
			}
		})
	}
}
