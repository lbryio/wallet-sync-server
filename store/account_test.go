package store

import (
	"errors"
	"testing"

	"github.com/mattn/go-sqlite3"

	"orblivion/lbry-id/auth"
)

func expectAccountMatch(t *testing.T, s *Store, email auth.Email, password auth.Password) {
	rows, err := s.db.Query(
		`SELECT 1 from accounts WHERE email=? AND password=?`,
		email, password.Obfuscate(),
	)
	if err != nil {
		t.Fatalf("Error finding account for: %s %s - %+v", email, password, err)
	}
	defer rows.Close()

	for rows.Next() {
		return // found something, we're good
	}

	t.Fatalf("Expected account for: %s %s", email, password)
}

func expectAccountNotMatch(t *testing.T, s *Store, email auth.Email, password auth.Password) {
	rows, err := s.db.Query(
		`SELECT 1 from accounts WHERE email=? AND password=?`,
		email, password.Obfuscate(),
	)
	if err != nil {
		t.Fatalf("Error finding account for: %s %s - %+v", email, password, err)
	}
	defer rows.Close()

	for rows.Next() {
		t.Fatalf("Expected no account for: %s %s", email, password)
	}

	// found nothing, we're good
}

// Test CreateAccount, using GetUserId as a helper
// Try CreateAccount twice with the same email and different password, error the second time
func TestStoreCreateAccount(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	email, password := auth.Email("abc@example.com"), auth.Password("123")

	// Get an account, come back empty
	expectAccountNotMatch(t, &s, email, password)

	// Create an account
	if err := s.CreateAccount(email, password); err != nil {
		t.Fatalf("Unexpected error in CreateAccount: %+v", err)
	}

	// Get and confirm the account we just put in
	expectAccountMatch(t, &s, email, password)

	newPassword := auth.Password("xyz")

	// Try to create a new account with the same email and different password,
	// fail because email already exists
	if err := s.CreateAccount(email, newPassword); err != ErrDuplicateAccount {
		t.Fatalf(`CreateAccount err: wanted "%+v", got "%+v"`, ErrDuplicateAccount, err)
	}

	// Get the email and same *first* password we successfully put in, but not the second
	expectAccountMatch(t, &s, email, password)
	expectAccountNotMatch(t, &s, email, newPassword)
}

// Test GetUserId, using CreateAccount as a helper
// Try GetUserId before creating an account (fail), and after (succeed)
func TestStoreGetUserId(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	email, password := auth.Email("abc@example.com"), auth.Password("123")

	// Check that there's no user id for email and password first
	if userId, err := s.GetUserId(email, password); err != ErrWrongCredentials || userId != 0 {
		t.Fatalf(`CreateAccount err: wanted "%+v", got "%+v. userId: %v"`, ErrWrongCredentials, err, userId)
	}

	// Create the account
	_ = s.CreateAccount(email, password)

	// Check that there's now a user id for the email and password
	if userId, err := s.GetUserId(email, password); err != nil || userId == 0 {
		t.Fatalf("Unexpected error in GetUserId: err: %+v userId: %v", err, userId)
	}
}

func TestStoreAccountEmptyFields(t *testing.T) {
	// Make sure expiration doesn't get set if sanitization fails
	tt := []struct {
		name     string
		email    auth.Email
		password auth.Password
	}{
		{
			name:     "missing email",
			email:    "",
			password: "xyz",
		},
		// Not testing empty password because it gets obfuscated to something
		// non-empty in the method
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s, sqliteTmpFile := StoreTestInit(t)
			defer StoreTestCleanup(sqliteTmpFile)

			var sqliteErr sqlite3.Error

			err := s.CreateAccount(tc.email, tc.password)
			if errors.As(err, &sqliteErr) {
				if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintCheck) {
					return // We got the error we expected
				}
			}
			t.Errorf("Expected check constraint error for empty field. Got %+v", err)
		})
	}
}
