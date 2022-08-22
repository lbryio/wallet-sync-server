package env

import (
	"fmt"
	"os"
	"strings"

	"lbryio/wallet-sync-server/auth"
)

// NOTE for users: If you have weird characters in your email address, please
// remember to properly escape it as necessary when putting it in an
// environmental variable, lest you run commands you didn't mean to run.
//
// We'll replace this with a config file later.
const whitelistKey = "ACCOUNT_WHITELIST"
const verificationModeKey = "ACCOUNT_VERIFICATION_MODE"
const mailgunIsDomainEUKey = "MAILGUN_SENDING_DOMAIN_IS_EU"
const mailgunPrivateAPIKeyKey = "MAILGUN_PRIVATE_API_KEY"

// for the "from" address
const mailgunSendingDomainKey = "MAILGUN_SENDING_DOMAIN"

// for links in the emails
const mailgunServerDomainKey = "MAILGUN_SERVER_DOMAIN"

type AccountVerificationMode string

// Everyone can make an account. Only use for dev purposes.
const AccountVerificationModeAllowAll = AccountVerificationMode("AllowAll")

// Verify accounts via email. Good for big open servers.
const AccountVerificationModeEmailVerify = AccountVerificationMode("EmailVerify")

// Specific email accounts are automatically verified. Good for small
// self-hosting users.
const AccountVerificationModeWhitelist = AccountVerificationMode("Whitelist")

// For test stubs
type EnvInterface interface {
	Getenv(key string) string
}

type Env struct{}

func (e *Env) Getenv(key string) string {
	return os.Getenv(key)
}

func GetAccountVerificationMode(e EnvInterface) (AccountVerificationMode, error) {
	return getAccountVerificationMode(e.Getenv(verificationModeKey))
}

func GetAccountWhitelist(e EnvInterface, mode AccountVerificationMode) (emails []auth.Email, err error) {
	return getAccountWhitelist(e.Getenv(whitelistKey), mode)
}

func GetMailgunConfigs(e EnvInterface, mode AccountVerificationMode) (sendingDomain string, serverDomain string, isDomainEU bool, privateAPIKey string, err error) {
	return getMailgunConfigs(e.Getenv(mailgunSendingDomainKey), e.Getenv(mailgunServerDomainKey), e.Getenv(mailgunIsDomainEUKey), e.Getenv(mailgunPrivateAPIKeyKey), mode)
}

// Factor out the guts of the functions so we can test them by just passing in
// the env vars

func getAccountVerificationMode(modeStr string) (AccountVerificationMode, error) {
	mode := AccountVerificationMode(modeStr)
	switch mode {
	case "":
		// Whitelist is the least dangerous mode. If you forget to set any env
		// vars, it effectively disables all account creation.
		return AccountVerificationModeWhitelist, nil
	case AccountVerificationModeAllowAll:
	case AccountVerificationModeEmailVerify:
	case AccountVerificationModeWhitelist:
	default:
		return "", fmt.Errorf("Invalid account verification mode in %s: %s", verificationModeKey, mode)
	}
	return mode, nil
}

func getAccountWhitelist(whitelist string, mode AccountVerificationMode) (emails []auth.Email, err error) {
	if whitelist == "" {
		return []auth.Email{}, nil
	}

	if mode != AccountVerificationModeWhitelist {
		return nil, fmt.Errorf("Do not specify %s in env if %s is not %s",
			whitelistKey,
			verificationModeKey,
			AccountVerificationModeWhitelist,
		)
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

func getMailgunConfigs(sendingDomain string, serverDomain string, isDomainEUStr string, privateAPIKey string, mode AccountVerificationMode) (string, string, bool, string, error) {
	if mode != AccountVerificationModeEmailVerify && (sendingDomain != "" || serverDomain != "" || isDomainEUStr != "" || privateAPIKey != "") {
		return "", "", false, "", fmt.Errorf("Do not specify %s, %s, %s or %s in env if %s is not %s",
			mailgunSendingDomainKey,
			mailgunServerDomainKey,
			mailgunIsDomainEUKey,
			mailgunPrivateAPIKeyKey,
			verificationModeKey,
			AccountVerificationModeEmailVerify,
		)
	}
	if mode == AccountVerificationModeEmailVerify && (sendingDomain == "" || serverDomain == "" || privateAPIKey == "") {
		return "", "", false, "", fmt.Errorf("Specify %s, %s and %s in env if %s is %s",
			mailgunSendingDomainKey,
			mailgunServerDomainKey,
			mailgunPrivateAPIKeyKey,
			verificationModeKey,
			AccountVerificationModeEmailVerify,
		)
	}

	if isDomainEUStr != "true" && isDomainEUStr != "false" && isDomainEUStr != "" {
		return "", "", false, "", fmt.Errorf("%s must be 'true' or 'false'", mailgunIsDomainEUKey)
	}

	return sendingDomain, serverDomain, isDomainEUStr == "true", privateAPIKey, nil
}
