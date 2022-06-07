package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// TODO - Learn how to use https://github.com/golang/oauth2 instead
// TODO - Look into jwt, etc.
// For now I just want a process that's shaped like what I'm looking for.
//   (email/password, encrypted wallets, hmac, lastSynced, etc)

type UserId int32
type Email string
type DeviceId string
type Password string
type AuthTokenString string
type AuthScope string

const ScopeFull = AuthScope("*")

// For test stubs
type AuthInterface interface {
	// TODO maybe have a "refresh token" thing if the client won't have email available all the time?
	NewToken(UserId, DeviceId, AuthScope) (*AuthToken, error)
}

type Auth struct{}

// Note that everything here is given to anybody who presents a valid
// downloadKey and associated email. Currently these fields are safe to give
// at that low security level, but keep this in mind as we change this struct.
type AuthToken struct {
	Token      AuthTokenString `json:"token"`
	DeviceId   DeviceId        `json:"deviceId"`
	Scope      AuthScope       `json:"scope"`
	UserId     UserId          `json:"userId"`
	Expiration *time.Time      `json:"expiration"`
}

const AuthTokenLength = 32

func (a *Auth) NewToken(userId UserId, deviceId DeviceId, scope AuthScope) (*AuthToken, error) {
	b := make([]byte, AuthTokenLength)
	// TODO - Is this is a secure random function? (Maybe audit)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("Error generating token: %+v", err)
	}

	return &AuthToken{
		Token:    AuthTokenString(hex.EncodeToString(b)),
		DeviceId: deviceId,
		Scope:    scope,
		UserId:   userId,
		// TODO add Expiration here instead of putting it in store.go. and thus redo store.go. d'oh.
	}, nil
}

// NOTE - not stubbing methods of structs like this. more convoluted than it's worth right now
func (at *AuthToken) ScopeValid(required AuthScope) bool {
	// So far the only scope issued. Used to have more, didn't want to delete
	// this feature yet in case we add more again. We'll delete it if it's of
	// no use and ends up complicating anything.
	return at.Scope == ScopeFull
}

func (p Password) Obfuscate() string {
	// TODO KDF instead
	hash := sha256.Sum256([]byte(p))
	return hex.EncodeToString(hash[:])
}
