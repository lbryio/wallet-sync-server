package env

import (
	"fmt"
	"os"
	"strings"

	"lbryio/lbry-id/auth"
)

const whitelistKey = "ACCOUNT_WHITELIST"
const verificationModeKey = "ACCOUNT_VERIFICATION_MODE"

// Not exported, so that GetAccountVerificationMode is the only way to get one
// of these values from outside of this package.
type accountVerificationMode string

// Everyone can make an account. Only use for dev purposes.
const ModeAllowAll = accountVerificationMode("AllowAll")

// Verify accounts via email. Good for big open servers.
const ModeEmailVerify = accountVerificationMode("EmailVerify")

// Specific email accounts are automatically verified. Good for small
// self-hosting users.
const ModeWhitelist = accountVerificationMode("Whitelist")

// For test stubs
type EnvInterface interface {
	Getenv(key string) string
}

type Env struct{}

func (e *Env) Getenv(key string) string {
	return os.Getenv(key)
}

func GetAccountVerificationMode(e EnvInterface) (accountVerificationMode, error) {
	return getAccountVerificationMode(e.Getenv(verificationModeKey))
}

func GetAccountWhitelist(e EnvInterface, mode accountVerificationMode) (emails []auth.Email, err error) {
	return getAccountWhitelist(e.Getenv(whitelistKey), mode)
}

// Factor out the guts of the functions so we can test them by just passing in
// the env vars

func getAccountVerificationMode(modeStr string) (accountVerificationMode, error) {
	mode := accountVerificationMode(modeStr)
	switch mode {
	case "":
		// Whitelist is the least dangerous mode. If you forget to set any env
		// vars, it effectively disables all account creation.
		return ModeWhitelist, nil
	case ModeAllowAll:
	case ModeEmailVerify:
	case ModeWhitelist:
	default:
		return "", fmt.Errorf("Invalid account verification mode in %s: %s", verificationModeKey, mode)
	}
	return mode, nil
}

func getAccountWhitelist(whitelist string, mode accountVerificationMode) (emails []auth.Email, err error) {
	if whitelist == "" {
		return []auth.Email{}, nil
	}

	if mode != ModeWhitelist {
		return nil, fmt.Errorf("Do not specify ACCOUNT_WHITELIST in env if ACCOUNT_VERIFICATION_MODE is not Whitelist")
	}

	rawEmails := strings.Split(whitelist, ",")
	for _, rawEmail := range rawEmails {
		// Give them a specific error here to let them know not to add spaces. It
		// could be confusing otherwise to figure out what's invalid.
		if strings.TrimSpace(rawEmail) != rawEmail {
			return nil, fmt.Errorf("Emails in %s should be comma separated with no spaces.", whitelistKey)
		}
		email := auth.Email(rawEmail)
		if !email.Validate() {
			return nil, fmt.Errorf("Invalid email in %s: %s", whitelistKey, email)
		}
		emails = append(emails, email)
	}
	return emails, nil
}
