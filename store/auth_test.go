package store

import (
	"strings"
	"testing"

	"orblivion/lbry-id/auth"
)

func expectAccountExists(t *testing.T, s *Store, email auth.Email, password auth.Password) {
	_, err := s.GetUserId(email, password)
	if err != nil {
		t.Fatalf("Unexpected error in GetUserId: %+v", err)
	}
}

func expectAccountNotExists(t *testing.T, s *Store, email auth.Email, password auth.Password) {
	_, err := s.GetUserId(email, password)
	if err != ErrNoUId {
		t.Fatalf("Expected ErrNoUId. err: %+v", err)
	}
}

// Test CreateAccount, using GetUserId as a helper
// Try CreateAccount twice with the same email and different password, error the second time
func TestStoreCreateAccount(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	email, password := auth.Email("abc@example.com"), auth.Password("123")

	// Get an account, come back empty
	expectAccountNotExists(t, &s, email, password)

	// Create an account
	if err := s.CreateAccount(email, password); err != nil {
		t.Fatalf("Unexpected error in CreateAccount: %+v", err)
	}

	// Get and confirm the account we just put in
	expectAccountExists(t, &s, email, password)

	newPassword := auth.Password("xyz")

	// Try to create a new account with the same email and different password,
	// fail because email already exists
	if err := s.CreateAccount(email, newPassword); err != ErrDuplicateAccount {
		t.Fatalf(`CreateAccount err: wanted "%+v", got "%+v"`, ErrDuplicateAccount, err)
	}

	// Get the email and same *first* password we successfully put in, but not the second
	expectAccountExists(t, &s, email, password)
	expectAccountNotExists(t, &s, email, newPassword)
}

// Test GetUserId, using CreateAccount as a helper
// Try GetUserId before creating an account (fail), and after (succeed)
func TestStoreGetUserId(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	email, password := auth.Email("abc@example.com"), auth.Password("123")

	// Check that there's no user id for email and password first
	if userId, err := s.GetUserId(email, password); err != ErrNoUId || userId != 0 {
		t.Fatalf(`CreateAccount err: wanted "%+v", got "%+v. userId: %v"`, ErrNoUId, err, userId)
	}

	// Create the account
	_ = s.CreateAccount(email, password)

	// Check that there's now a user id for the email and password
	if userId, err := s.GetUserId(email, password); err != nil || userId == 0 {
		t.Fatalf("Unexpected error in GetUserId: err: %+v userId: %v", err, userId)
	}
}

// Make sure we're saving in UTC. Make sure we have no weird timezone issues.
func TestStoreTokenUTC(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	authToken := auth.AuthToken{
		Token:    "seekrit-1",
		DeviceId: "dId",
		Scope:    "*",
		UserId:   123,
	}

	if err := s.SaveToken(&authToken); err != nil {
		t.Fatalf("Unexpected error in SaveToken: %+v", err)
	}

	rows, err := s.db.Query("SELECT expiration FROM auth_tokens LIMIT 1")
	defer rows.Close()

	if err != nil {
		t.Fatalf("Unexpected error getting expiration from db: %+v", err)
	}

	var expirationString string
	for rows.Next() {

		err := rows.Scan(
			&expirationString,
		)

		if err != nil {
			t.Fatalf("Unexpected error parsing expiration from db: %+v", err)
		}
	}

	if !strings.HasSuffix(expirationString, "Z") {
		t.Fatalf("Expected expiration timezone to be UTC (+00:00). Got %s", expirationString)
	}
}
