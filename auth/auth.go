package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/scrypt"
)

type UserId int32
type NormalizedEmail string // Should always contain a normalized value
type Email string
type DeviceId string
type Password string
type KDFKey string         // KDF output
type ClientSaltSeed string // part of client-side KDF input along with root password
type ServerSalt string     // server-side KDF input for accounts
type AuthTokenString string
type VerifyTokenString string
type AuthScope string

const ScopeFull = AuthScope("*")

// For test stubs
type AuthInterface interface {
	// TODO maybe have a "refresh token" thing if the client won't have email available all the time?
	NewAuthToken(UserId, DeviceId, AuthScope) (*AuthToken, error)
	NewVerifyTokenString() (VerifyTokenString, error)
}

type Auth struct{}

type AuthToken struct {
	Token      AuthTokenString `json:"token"`
	DeviceId   DeviceId        `json:"deviceId"`
	Scope      AuthScope       `json:"scope"`
	UserId     UserId          `json:"userId"`
	Expiration *time.Time      `json:"expiration"`
}

const TokenLength = 32

func (a *Auth) NewAuthToken(userId UserId, deviceId DeviceId, scope AuthScope) (*AuthToken, error) {
	b := make([]byte, TokenLength)
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

func (a *Auth) NewVerifyTokenString() (VerifyTokenString, error) {
	b := make([]byte, TokenLength)
	// TODO - Is this is a secure random function? (Maybe audit)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("Error generating token: %+v", err)
	}

	return VerifyTokenString(hex.EncodeToString(b)), nil
}

// NOTE - not stubbing methods of structs like this. more convoluted than it's worth right now
func (at *AuthToken) ScopeValid(required AuthScope) bool {
	// So far * is the only scope issued. Used to have more, didn't want to
	// delete this feature yet in case we add more again. We'll delete it if it's
	// of no use and ends up complicating anything.
	return at.Scope == ScopeFull || at.Scope == required
}

const ServerSaltLength = 16
const ClientSaltSeedLength = 32

const passwordScryptN = 1 << 15
const passwordScryptR = 8
const passwordScryptP = 1
const passwordScryptKeyLen = 32

// https://words.filippo.io/the-scrypt-parameters/
func passwordScrypt(p Password, saltBytes []byte) ([]byte, error) {
	return scrypt.Key(
		[]byte(p),
		saltBytes,
		passwordScryptN,
		passwordScryptR,
		passwordScryptP,
		passwordScryptKeyLen,
	)
}

// Given a password (in the same format submitted via request), generate a
// random salt, run the password and salt thorugh the KDF, and return the salt
// and kdf output. The result generally goes into a database.
func (p Password) Create() (key KDFKey, salt ServerSalt, err error) {
	saltBytes := make([]byte, ServerSaltLength)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", "", fmt.Errorf("Error generating salt: %+v", err)
	}
	keyBytes, err := passwordScrypt(p, saltBytes)
	if err == nil {
		key = KDFKey(hex.EncodeToString(keyBytes[:]))
		salt = ServerSalt(hex.EncodeToString(saltBytes[:]))
	}
	return
}

// Given a password (in the same format submitted via request), a salt, and an
// expected kdf output, run the password and salt thorugh the KDF, and return
// whether the result kdf output matches the kdf test output.
// The salt and test kdf output generally come out of the database, and is used
// to check a submitted password.
func (p Password) Check(checkKey KDFKey, salt ServerSalt) (match bool, err error) {
	saltBytes, err := hex.DecodeString(string(salt))
	if err != nil {
		return false, fmt.Errorf("Error decoding salt from hex: %+v", err)
	}
	keyBytes, err := passwordScrypt(p, saltBytes)
	if err == nil {
		match = KDFKey(hex.EncodeToString(keyBytes[:])) == checkKey
	}
	return
}

func (e Email) Validate() bool {
	email, err := mail.ParseAddress(string(e))
	if err != nil {
		return false
	}
	// "Joe <joe@example.com>" is valid according to ParseAddress. Likewise
	// " joe@example.com". Etc. We only want the exact address, "joe@example.com"
	// to be valid. ParseAddress will extract the exact address as
	// parsed.Address. So we'll take the input email, put it through
	// ParseAddress, see if it parses successfully, and then compare the input
	// email to parsed.Address to make sure that it was an exact address to begin
	// with.
	return string(e) == email.Address
}

func (c ClientSaltSeed) Validate() bool {
	_, err := hex.DecodeString(string(c))
	const seedHexLength = ClientSaltSeedLength * 2
	return len(c) == seedHexLength && err == nil
}

func (p Password) Validate() bool {
	return len(p) >= 8 // Should be much longer but it's a sanity check.
}

// TODO consider unicode. Also some providers might be case sensitive, and/or
// may have other ways of having email addresses be equivalent (which we may
// not care about though)
func (e Email) Normalize() NormalizedEmail {
	return NormalizedEmail(strings.ToLower(string(e)))
}
