package main // TODO - make it its own `auth` package later

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// TODO - Learn how to use https://github.com/golang/oauth2 instead
// TODO - Look into jwt, etc.
// For now I just want a process that's shaped like what I'm looking for (pubkey signatures, downloadKey, etc)

type AuthTokenString string
type PublicKey string

type AuthInterface interface {
	NewFullToken(pubKey PublicKey, tokenRequest *TokenRequest) (*AuthToken, error)
	IsValidSignature(pubKey PublicKey, payload string, signature string) bool

	// for future request:
	//   IsValidToken(AuthTokenString) bool
}

type Auth struct{}

func (a *Auth) IsValidSignature(pubKey PublicKey, payload string, signature string) bool {
	// TODO - a real check
	return signature == "Good Signature"
}

type AuthToken struct {
	Token      AuthTokenString `json:"token"`
	DeviceID   string          `json:"deviceId"`
	Scope      string          `json:"scope"`
	PubKey     PublicKey       `json:"publicKey"`
	Expiration *time.Time      `json:"expiration"`
}

type TokenRequest struct {
	DeviceID string `json:"deviceId"`
}

// TODO - probably shouldn't be (s *Server) in this file
func (s *Server) validateTokenRequest(tokenRequest *TokenRequest) bool {
	// TODO
	return true
}

const tokenLength = 32

func (a *Auth) NewFullToken(pubKey PublicKey, tokenRequest *TokenRequest) (*AuthToken, error) {
	b := make([]byte, tokenLength)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("Error generating token: %+v", err)
	}

	return &AuthToken{
		Token:    AuthTokenString(hex.EncodeToString(b)),
		DeviceID: tokenRequest.DeviceID,
		Scope:    "*",
		PubKey:   pubKey,
		// TODO add Expiration here instead of putting it in store.go. and thus redo store.go. d'oh.
	}, nil
}
