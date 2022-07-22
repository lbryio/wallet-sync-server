package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
type TokenString string
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
	Token      TokenString `json:"token"`
	DeviceId   DeviceId    `json:"deviceId"`
	Scope      AuthScope   `json:"scope"`
	UserId     UserId      `json:"userId"`
	Expiration *time.Time  `json:"expiration"`
}

const AuthTokenLength = 32

func (a *Auth) NewToken(userId UserId, deviceId DeviceId, scope AuthScope) (*AuthToken, error) {
	b := make([]byte, AuthTokenLength)
	// TODO - Is this is a secure random function? (Maybe audit)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("Error generating token: %+v", err)
	}

	return &AuthToken{
		Token:    TokenString(hex.EncodeToString(b)),
		DeviceId: deviceId,
		Scope:    scope,
		UserId:   userId,
		// TODO add Expiration here instead of putting it in store.go. and thus redo store.go. d'oh.
	}, nil
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

// https://words.filippo.io/the-scrypt-parameters/
func passwordScrypt(p Password, saltBytes []byte) ([]byte, error) {
	scryptN := 32768
	scryptR := 8
	scryptP := 1
	keyLen := 32
	return scrypt.Key([]byte(p), saltBytes, scryptN, scryptR, scryptP, keyLen)
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

// TODO consider unicode. Also some providers might be case sensitive, and/or
// may have other ways of having email addresses be equivalent (which we may
// not care about though)
func (e Email) Normalize() NormalizedEmail {
	return NormalizedEmail(strings.ToLower(string(e)))
}
