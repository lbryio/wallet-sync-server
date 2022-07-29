package mail

import (
	"lbryio/lbry-id/auth"
)

type MailInterface interface {
	SendVerificationEmail(email auth.Email, token auth.VerifyTokenString) error
}

type Mail struct{}

func (m *Mail) SendVerificationEmail(auth.Email, auth.VerifyTokenString) (err error) {
	return
}
