package store

import (
	"errors"
	"testing"

	"github.com/mattn/go-sqlite3"

	"lbryio/lbry-id/auth"
)

func expectAccountMatch(t *testing.T, s *Store, email auth.Email, password auth.Password) {
	var key auth.KDFKey
	var salt auth.Salt

	err := s.db.QueryRow(
		`SELECT key, salt from accounts WHERE email=?`,
		email,
	).Scan(&key, &salt)
	if err != nil {
		t.Fatalf("Error finding account for: %s %s - %+v", email, password, err)
	}

	match, err := password.Check(key, salt)
	if err != nil {
		t.Fatalf("Error checking password for: %s %s - %+v", email, password, err)
	}
	if !match {
		t.Fatalf("Expected account for: %s %s", email, password)
	}
}

func expectAccountNotExists(t *testing.T, s *Store, email auth.Email) {
	rows, err := s.db.Query(
		`SELECT 1 from accounts WHERE email=?`,
		email,
	)
	if err != nil {
		t.Fatalf("Error finding account for: %s - %+v", email, err)
	}
	defer rows.Close()

	for rows.Next() {
		t.Fatalf("Expected no account for: %s", email)
	}

	// found nothing, we're good
}

// Test CreateAccount
// Try CreateAccount twice with the same email and different password, error the second time
func TestStoreCreateAccount(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	email, password := auth.Email("abc@example.com"), auth.Password("123")

	// Get an account, come back empty
	expectAccountNotExists(t, &s, email)

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

	// Get the email and same *first* password we successfully put in
	expectAccountMatch(t, &s, email, password)
}

// Test GetUserId for nonexisting email
func TestStoreGetUserIdAccountNotExists(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	email, password := auth.Email("abc@example.com"), auth.Password("123")

	// Check that there's no user id for email and password first
	if userId, err := s.GetUserId(email, password); err != ErrWrongCredentials || userId != 0 {
		t.Fatalf(`GetUserId error for nonexistant account: wanted "%+v", got "%+v. userId: %v"`, ErrWrongCredentials, err, userId)
	}
}

// Test GetUserId for existing account, with the correct and incorrect password
func TestStoreGetUserIdAccountExists(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	createdUserId, email, password := makeTestUser(t, &s)

	// Check that there's now a user id for the email and password
	if userId, err := s.GetUserId(email, password); err != nil || userId != createdUserId {
		t.Fatalf("Unexpected error in GetUserId: err: %+v userId: %v", err, userId)
	}

	// Check that it won't return if the wrong password is given
	if userId, err := s.GetUserId(email, password+auth.Password("_wrong")); err != ErrWrongCredentials || userId != 0 {
		t.Fatalf(`GetUserId error for wrong password: wanted "%+v", got "%+v. userId: %v"`, ErrWrongCredentials, err, userId)
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
		// Not testing empty key and salt because they get generated to something
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
