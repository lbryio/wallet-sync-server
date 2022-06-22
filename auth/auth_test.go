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
