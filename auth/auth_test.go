package auth

import (
	"testing"
)

// Test stubs for now

func TestAuthNewToken(t *testing.T) {
	auth := Auth{}
	authToken, err := auth.NewToken(234, "dId", "my-scope")

	if err != nil {
		t.Fatalf("Error creating new token")
	}

	if authToken.UserId != 234 ||
		authToken.DeviceId != "dId" ||
		authToken.Scope != "my-scope" {
		t.Fatalf("authToken fields don't match expected values")
	}

	// result.Token is in hex, AuthTokenLength is bytes in the original
	expectedTokenLength := AuthTokenLength * 2
	if len(authToken.Token) != expectedTokenLength {
		t.Fatalf("authToken token string length isn't the expected length")
	}
}

func TestAuthScopeValid(t *testing.T) {
	fullAuthToken := AuthToken{Scope: "*"}
	if !fullAuthToken.ScopeValid("*") {
		t.Fatalf("Expected * to be a valid scope for *")
	}
	if !fullAuthToken.ScopeValid("banana") {
		t.Fatalf("Expected * to be a valid scope for banana")
	}

	bananaAuthToken := AuthToken{Scope: "banana"}
	if !bananaAuthToken.ScopeValid("banana") {
		t.Fatalf("Expected banana to be a valid scope for banana")
	}
}

func TestAuthScopeInvalid(t *testing.T) {
	bananaAuthToken := AuthToken{Scope: "banana"}

	if bananaAuthToken.ScopeValid("*") {
		t.Fatalf("Expected banana to be an invalid scope for *")
	}

	if bananaAuthToken.ScopeValid("carrot") {
		t.Fatalf("Expected banana to be an invalid scope for carrot")
	}
}

func TestCreatePassword(t *testing.T) {
	// Since the salt is randomized, there's really not much we can do to test
	// the create function other than to check the length of the outputs and that
	// they're different each time.

	const password = Password("password")

	key1, salt1, err := password.Create()
	if err != nil {
		t.Error("Error creating password")
	}
	if len(key1) != 64 {
		t.Error("Key has wrong length", key1)
	}
	if len(salt1) != 16 {
		t.Error("Salt has wrong length", salt1)
	}

	key2, salt2, err := password.Create()
	if err != nil {
		t.Error("Error creating password")
	}
	if key1 == key2 {
		t.Error("Key is not random", key1)
	}
	if salt1 == salt2 {
		t.Error("Salt is not random", key1)
	}
}

func TestCheckPassword(t *testing.T) {
	const password = Password("password 1")
	const key = KDFKey("b9a3669973fcd2da3625e84da9d9a2da87bd280bcb02586851e1cb5bee1efa10")
	const salt = Salt("080cbdf6d247c665")

	match, err := password.Check(key, salt)
	if err != nil {
		t.Error("Error checking password")
	}
	if !match {
		t.Error("Expected password to match correct key and salt")
	}

	const wrongKey = KDFKey("0000000073fcd2da3625e84da9d9a2da87bd280bcb02586851e1cb5bee1efa10")
	match, err = password.Check(wrongKey, salt)
	if err != nil {
		t.Error("Error checking password")
	}
	if match {
		t.Error("Expected password to not match incorrect key")
	}

	const wrongSalt = Salt("00000000d247c665")
	match, err = password.Check(key, wrongSalt)
	if err != nil {
		t.Error("Error checking password")
	}
	if match {
		t.Error("Expected password to not match incorrect salt")
	}

	const invalidSalt = Salt("Whoops")
	match, err = password.Check(key, invalidSalt)
	if err == nil {
		// It does a decode of salt inside the function but not the key so we won't
		// test invalid hex string with that
		t.Error("Expected password check to fail with invalid salt")
	}
}
