package store

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"lbryio/lbry-id/auth"
)

func StoreTestInit(t *testing.T) (s Store, tmpFile *os.File) {
	s = Store{}

	tmpFile, err := ioutil.TempFile(os.TempDir(), "sqlite-test-")
	if err != nil {
		t.Fatalf("DB setup failure: %+v", err)
		return
	}

	s.Init(tmpFile.Name())

	err = s.Migrate()
	if err != nil {
		t.Fatalf("DB setup failure: %+v", err)
	}

	return
}

func StoreTestCleanup(tmpFile *os.File) {
	if tmpFile != nil {
		os.Remove(tmpFile.Name())
	}
}

func makeTestUser(
	t *testing.T,
	s *Store,
	verifyToken *auth.VerifyTokenString,
	verifyExpiration *time.Time,
) (userId auth.UserId, email auth.Email, password auth.Password, seed auth.ClientSaltSeed) {
	// email with caps to trigger possible problems
	email, password = auth.Email("Abc@Example.Com"), auth.Password("123")
	normEmail := auth.NormalizedEmail("abc@example.com")
	key, salt, err := password.Create()
	if err != nil {
		t.Fatalf("Error creating password")
	}

	seed = auth.ClientSaltSeed("abcd1234abcd1234")

	rows, err := s.db.Query(
		"INSERT INTO accounts (normalized_email, email, key, server_salt, client_salt_seed, verify_token, verify_expiration) values(?,?,?,?,?,?,?) returning user_id",
		normEmail, email, key, salt, seed, verifyToken, verifyExpiration,
	)
	if err != nil {
		t.Fatalf("Error setting up account: %+v", err)
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&userId)
		if err != nil {
			t.Fatalf("Error setting up account: %+v", err)
		}
		return
	}
	t.Fatalf("Error setting up account - no rows found")
	return
}
