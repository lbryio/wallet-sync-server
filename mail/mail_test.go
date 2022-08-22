package mail

import (
	"strings"
	"testing"

	"lbryio/wallet-sync-server/auth"
)

type TestEnv struct {
	env map[string]string
}

func (e *TestEnv) Getenv(key string) string {
	return e.env[key]
}

func TestPrepareEmailNotEU(t *testing.T) {

	const apiKey = "mg-api-key"
	const sendingDomain = "sending.example.com"
	const serverDomain = "server.example.com"

	const recipient = auth.Email("recipient@example.com")
	const token = auth.VerifyTokenString("abcd1234abcd1234abcd1234abcd1234")

	env := map[string]string{
		"ACCOUNT_VERIFICATION_MODE": "EmailVerify",
		"MAILGUN_PRIVATE_API_KEY":   apiKey,
		"MAILGUN_SENDING_DOMAIN":    sendingDomain,
		"MAILGUN_SERVER_DOMAIN":     serverDomain,
	}

	m := Mail{&TestEnv{env}}

	mg, sender, subject, text, html, err := m.prepareMessage(token)

	if err != nil || mg == nil {
		t.Errorf("Unexpected values from prepareMessage: %+v %s", mg, err.Error())
	}

	if got, want := mg.APIKey(), apiKey; want != got {
		t.Errorf("Unexpected mg.APIKey(). Got: %s Want: %s", want, got)
	}

	if got, want := mg.APIBase(), "https://api.mailgun.net/v3"; want != got {
		t.Errorf("Unexpected mg.APIBase(). Got: %s Want: %s", want, got)
	}

	if got, want := sender, "wallet-sync@sending.example.com"; want != got {
		t.Errorf("Unexpected sender. Got: %s Want: %s", want, got)
	}

	if !strings.Contains(subject, serverDomain) {
		t.Errorf("Expected subject to contain %s. Got: %s", serverDomain, subject)
	}

	if !strings.Contains(text, serverDomain) {
		t.Errorf("Expected text to contain %s. Got: %s", serverDomain, text)
	}

	if !strings.Contains(html, serverDomain) {
		t.Errorf("Expected html to contain %s. Got: %s", serverDomain, html)
	}
}

func TestPrepareEmailEU(t *testing.T) {
	const apiKey = "mg-api-key"
	const sendingDomain = "sending.example.com"
	const serverDomain = "server.example.com"

	const recipient = auth.Email("recipient@example.com")
	const token = auth.VerifyTokenString("abcd1234abcd1234abcd1234abcd1234")

	env := map[string]string{
		"MAILGUN_SENDING_DOMAIN_IS_EU": "true",
		"ACCOUNT_VERIFICATION_MODE":    "EmailVerify",
		"MAILGUN_PRIVATE_API_KEY":      apiKey,
		"MAILGUN_SENDING_DOMAIN":       sendingDomain,
		"MAILGUN_SERVER_DOMAIN":        serverDomain,
	}

	m := Mail{&TestEnv{env}}

	mg, _, _, _, _, err := m.prepareMessage(token)

	if err != nil || mg == nil {
		t.Errorf("Unexpected values from prepareMessage: %+v %s", mg, err.Error())
	}

	if got, want := mg.APIBase(), "https://api.eu.mailgun.net/v3"; want != got {
		t.Errorf("Unexpected mg.APIBase(). Got: %s Want: %s", want, got)
	}
}
