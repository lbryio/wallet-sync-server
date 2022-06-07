package auth

import (
	"testing"
)

// Test stubs for now

func TestAuthNewTokenSuccess(t *testing.T) {
	t.Fatalf("Test me: New token passes. Different scopes etc.")
}

func TestAuthNewTokenFail(t *testing.T) {
	t.Fatalf("Test me: New token fails (error generating random string? others?)")
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
}
