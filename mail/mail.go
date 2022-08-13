package mail

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mailgun/mailgun-go/v4"

	"lbryio/lbry-id/auth"
	"lbryio/lbry-id/env"
	"lbryio/lbry-id/server/paths"
)

const MAILGUN_DEBUG = false

// useful with MAILGUN_DEBUG to see what gets called with what
const MAILGUN_DRY_RUN = false

type MailInterface interface {
	SendVerificationEmail(auth.Email, auth.VerifyTokenString) error
}

type Mail struct {
	Env env.EnvInterface
}

// Split out everything I can to make it testable. Right now
// mailgun.MailgunImpl is inspectable enough to test but
// mailgun.Message is not.
func (m *Mail) prepareMessage(token auth.VerifyTokenString) (
	mg *mailgun.MailgunImpl,
	sender string,
	subject string,
	text string,
	html string,
	err error,
) {
	verificationMode, err := env.GetAccountVerificationMode(m.Env)
	if err != nil {
		return
	}

	sendingDomain, serverDomain, isDomainEU, privateAPIKey, err := env.GetMailgunConfigs(m.Env, verificationMode)
	if err != nil {
		return
	}

	// Create an instance of the Mailgun Client
	mg = mailgun.NewMailgun(sendingDomain, privateAPIKey)

	// see https://help.mailgun.com/hc/en-us/articles/360007512013-Can-I-migrate-my-domain-to-EU-
	if isDomainEU {
		mg.SetAPIBase("https://api.eu.mailgun.net/v3")
	}

	sender = fmt.Sprintf("wallet-sync@%s", sendingDomain)
	subject = fmt.Sprintf("Verify your wallet sync account on %s", serverDomain)
	url := fmt.Sprintf("https://%s%s?verifyToken=%s", serverDomain, paths.PathVerify, token)

	text = fmt.Sprintf("Click here to verify your account:\n\n%s", url)
	html = fmt.Sprintf("Click here to verify your account:\n\n<a href=\"%s\">%s</a>", url, url)

	if MAILGUN_DEBUG {
		log.Printf(
			"NewMessage\n\n%s\n\n%s\n\n%s\n\n%s",
			sender, subject, text, html,
		)
	}

	return
}

func (m *Mail) SendVerificationEmail(recipient auth.Email, token auth.VerifyTokenString) (err error) {
	mg, sender, subject, text, html, err := m.prepareMessage(token)

	if err != nil {
		return err
	}

	message := mg.NewMessage(sender, subject, text, string(recipient))
	message.SetHtml(html)

	if MAILGUN_DEBUG {
		log.Printf("Send\n\n%+v\n", message)
	}

	if !MAILGUN_DRY_RUN {
		// Send the message with a 10 second timeout
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		resp, id, err := mg.Send(ctx, message)

		if err != nil {
			log.Fatal(err)
		}

		if MAILGUN_DEBUG {
			log.Printf("Sent Mailgun message. ID: %s Resp: %s\n", id, resp)
		}
	}

	return
}
