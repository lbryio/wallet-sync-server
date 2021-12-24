package main

import (
	"testing"
)

// Test stubs for now

func TestAuthSignaturePass(t *testing.T) {
	t.Fatalf("Valid siganture passes")
}

func TestAuthSignatureFail(t *testing.T) {
	t.Fatalf("Valid siganture fails")
}

func TestAuthNewFullTokenSuccess(t *testing.T) {
	t.Fatalf("New token passes")
}

func TestAuthNewFullTokenFail(t *testing.T) {
	t.Fatalf("New token fails (error generating random string? others?)")
}
