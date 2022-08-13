package main

import (
	"log"

	"lbryio/lbry-id/auth"
	"lbryio/lbry-id/env"
	"lbryio/lbry-id/mail"
	"lbryio/lbry-id/server"
	"lbryio/lbry-id/store"
)

func storeInit() (s store.Store) {
	s = store.Store{}

	s.Init("sql.db")

	err := s.Migrate()
	if err != nil {
		log.Fatalf("DB setup failure: %+v", err)
	}

	return
}

// Output information about the email verification mode so the user can confirm
// what they set. Also trigger an error on startup if there's a configuration
// problem.
func logEmailVerificationConfigs(e *env.Env) (err error) {
	verificationMode, err := env.GetAccountVerificationMode(e)
	if err != nil {
		return
	}
	accountWhitelist, err := env.GetAccountWhitelist(e, verificationMode)
	if err != nil {
		return
	}

	// just to report config errors to the user on startup
	sendingDomain, serverDomain, _, _, err := env.GetMailgunConfigs(e, verificationMode)
	if err != nil {
		return
	}

	if verificationMode == env.AccountVerificationModeWhitelist {
		log.Printf("Account verification mode: %s - Whitelist has %d email(s).\n", verificationMode, len(accountWhitelist))
	} else {
		log.Printf("Account verification mode: %s", verificationMode)
	}
	if verificationMode == env.AccountVerificationModeEmailVerify {
		log.Printf("Mailgun domains: %s for sending addresses, %s for links in the email", sendingDomain, serverDomain)
	}
	return
}

func main() {
	e := env.Env{}

	if err := logEmailVerificationConfigs(&e); err != nil {
		log.Fatal(err.Error())
	}

	store := storeInit()

	// The port that the sync server serves from.
	internalPort := 8090

	srv := server.Init(&auth.Auth{}, &store, &e, &mail.Mail{&e}, internalPort)
	srv.Serve()
}
