package env

import (
	"fmt"
	"reflect"
	"testing"

	"lbryio/lbry-id/auth"
)

func TestAccountVerificationMode(t *testing.T) {
	tt := []struct {
		name string

		modeStr      string
		expectedMode AccountVerificationMode
		expectErr    bool
	}{
		{
			name: "allow all",

			modeStr:      "AllowAll",
			expectedMode: AccountVerificationModeAllowAll,
		},
		{
			name: "email verify",

			modeStr:      "EmailVerify",
			expectedMode: AccountVerificationModeEmailVerify,
		},
		{
			name: "whitelist",

			modeStr:      "Whitelist",
			expectedMode: AccountVerificationModeWhitelist,
		},
		{
			name: "blank",

			modeStr:      "",
			expectedMode: AccountVerificationModeWhitelist,
		},
		{
			name: "invalid",

			modeStr:   "Banana",
			expectErr: true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			mode, err := getAccountVerificationMode(tc.modeStr)
			if mode != tc.expectedMode {
				t.Errorf("Expected mode %s got %s", tc.expectedMode, mode)
			}
			if tc.expectErr && err == nil {
				t.Errorf("Expected err")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Unexpected err: %s", err.Error())
			}
		})
	}
}

func TestAccountWhitelist(t *testing.T) {
	tt := []struct {
		name string

		whitelist      string
		expectedEmails []auth.Email
		expectedErr    error
		mode           AccountVerificationMode
	}{
		{
			name: "empty",

			mode:           AccountVerificationModeWhitelist,
			whitelist:      "",
			expectedEmails: []auth.Email{},
		},
		{
			name: "invalid mode",

			mode:        AccountVerificationModeEmailVerify,
			whitelist:   "test1@example.com,test2@example.com",
			expectedErr: fmt.Errorf("Do not specify ACCOUNT_WHITELIST in env if ACCOUNT_VERIFICATION_MODE is not Whitelist"),
		},
		{
			name: "spaces in email",

			mode:        AccountVerificationModeWhitelist,
			whitelist:   "test1@example.com ,test2@example.com",
			expectedErr: fmt.Errorf("Emails in ACCOUNT_WHITELIST should be comma separated with no spaces."),
		},
		{
			name: "invalid email",

			mode:        AccountVerificationModeWhitelist,
			whitelist:   "test1@example.com,test2-example.com",
			expectedErr: fmt.Errorf("Invalid email in ACCOUNT_WHITELIST: test2-example.com"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			emails, err := getAccountWhitelist(tc.whitelist, tc.mode)
			if !reflect.DeepEqual(emails, tc.expectedEmails) {
				t.Errorf("Expected emails %+v got %+v", tc.expectedEmails, emails)
			}
			if fmt.Sprint(err) != fmt.Sprint(tc.expectedErr) {
				t.Errorf("Expected error `%s` got `%s`", tc.expectedErr, err.Error())
			}
		})
	}
}

func TestMailgunConfigs(t *testing.T) {
	tt := []struct {
		name string

		domain     string
		privateAPIKey string
		mode       AccountVerificationMode

		expectErr bool
	}{
		{
			name:       "success",
			mode:       AccountVerificationModeEmailVerify,
			domain:     "www.example.com",
			privateAPIKey: "my-private-api-key",
			expectErr:  false,
		},
		{
			name:      "wrong mode with domain",
			mode:      AccountVerificationModeWhitelist,
			domain:    "www.example.com",
			expectErr: true,
		},
		{
			name:       "wrong mode with private api key",
			mode:       AccountVerificationModeWhitelist,
			privateAPIKey: "my-private-api-key",
			expectErr:  true,
		},
		{
			name:       "missing domain",
			mode:       AccountVerificationModeEmailVerify,
			privateAPIKey: "my-private-api-key",
			expectErr:  true,
		},
		{
			name:      "missing private api key",
			mode:      AccountVerificationModeEmailVerify,
			domain:    "www.example.com",
			expectErr: true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			domain, privateAPIKey, err := getMailgunConfigs(tc.domain, tc.privateAPIKey, tc.mode)
			if tc.expectErr && err == nil {
				t.Errorf("Expected err")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Unexpected err: %s", err.Error())
			}
			if !tc.expectErr && tc.domain != domain {
				t.Errorf("Expected domain to be set")
			}
			if !tc.expectErr && tc.privateAPIKey != privateAPIKey {
				t.Errorf("Expected privateAPIKey to be set")
			}
		})
	}

}
