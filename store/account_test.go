package store

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mattn/go-sqlite3"

	"lbryio/wallet-sync-server/auth"
)

func expectAccountMatch(
	t *testing.T,
	s *Store,
	normEmail auth.NormalizedEmail,
	expectedEmail auth.Email,
	password auth.Password,
	seed auth.ClientSaltSeed,
	expectedVerifyTokenString *auth.VerifyTokenString,
	approxVerifyExpiration *time.Time,
	approxCreated time.Time,
	approxUpdated time.Time,
) {
	var key auth.KDFKey
	var salt auth.ServerSalt
	var email auth.Email
	var verifyExpiration *time.Time
	var verifyTokenString *auth.VerifyTokenString
	var created time.Time
	var updated time.Time

	err := s.db.QueryRow(
		`SELECT key, server_salt, email, verify_token, verify_expiration, created, updated from accounts WHERE normalized_email=? AND client_salt_seed=?`,
		normEmail, seed,
	).Scan(&key, &salt, &email, &verifyTokenString, &verifyExpiration, &created, &updated)
	if err != nil {
		t.Fatalf("Error finding account for: %s %s - %+v", normEmail, password, err)
	}

	match, err := password.Check(key, salt)
	if err != nil {
		t.Fatalf("Error checking password for: %s %s - %+v", email, password, err)
	}
	if !match {
		t.Fatalf("Password incorrect for: %s %s", email, password)
	}

	if email != expectedEmail {
		t.Fatalf("Email case not as expected. Want: %s Got: %s", email, expectedEmail)
	}

	if (verifyTokenString == nil) != (expectedVerifyTokenString == nil) {
		t.Fatalf(
			"Verify token string nil-ness not as expected. Want: %v Got: %v",
			expectedVerifyTokenString == nil,
			verifyTokenString == nil,
		)
	}

	if expectedVerifyTokenString != nil {
		if *verifyTokenString != *expectedVerifyTokenString {
			t.Fatalf(
				"Verify token string not as expected. Want: %s Got: %s",
				*expectedVerifyTokenString,
				*verifyTokenString,
			)
		}
	}

	if approxVerifyExpiration != nil {
		if verifyExpiration == nil {
			t.Fatalf("Expected verify expiration to not be nil")
		}
		expDiff := approxVerifyExpiration.Sub(*verifyExpiration)
		if time.Second < expDiff || expDiff < -time.Second {
			t.Fatalf(
				"Verify expiration not as expected. Want approximately: %s Got: %s",
				approxVerifyExpiration,
				verifyExpiration,
			)
		}
	}
	if approxVerifyExpiration == nil && verifyExpiration != nil {
		t.Fatalf("Expected verify expiration to be nil. Got: %+v", verifyExpiration)
	}

	expDiff := approxCreated.Sub(created)
	if time.Second*2 < expDiff || expDiff < -time.Second*2 {
		t.Fatalf(
			"Created timestamp not as expected. Want approximately: %s Got: %s",
			approxCreated,
			created,
		)
	}

	expDiff = approxUpdated.Sub(updated)
	if time.Second*2 < expDiff || expDiff < -time.Second*2 {
		t.Fatalf(
			"Updated timestamp not as expected. Want approximately: %s Got: %s",
			approxUpdated,
			updated,
		)
	}
}

func expectAccountNotExists(t *testing.T, s *Store, normEmail auth.NormalizedEmail) {
	rows, err := s.db.Query(
		`SELECT 1 from accounts WHERE normalized_email=?`,
		normEmail,
	)
	if err != nil {
		t.Fatalf("Error finding account for: %s - %+v", normEmail, err)
	}
	defer rows.Close()

	for rows.Next() {
		t.Fatalf("Expected no account for: %s", normEmail)
	}

	// found nothing, we're good
}

// Test CreateAccount
// Try CreateAccount twice with the same email and different password, error the second time
func TestStoreCreateAccount(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	email, normEmail := auth.Email("Abc@Example.Com"), auth.NormalizedEmail("abc@example.com")
	password, seed := auth.Password("123"), auth.ClientSaltSeed("abcd1234abcd1234")

	// Get an account, come back empty
	expectAccountNotExists(t, &s, normEmail)

	// Create an account. Make it verified (i.e. no token) for the usual
	// case. We'll test unverified (with token) separately.
	if err := s.CreateAccount(email, password, seed, nil); err != nil {
		t.Fatalf("Unexpected error in CreateAccount: %+v", err)
	}

	// Get and confirm the account we just put in
	expectAccountMatch(t, &s, normEmail, email, password, seed, nil, nil, time.Now().UTC(), time.Now().UTC())

	newPassword := auth.Password("xyz")

	// Try to create a new account with the same email and different password,
	// fail because email already exists
	if err := s.CreateAccount(email, newPassword, seed, nil); err != ErrDuplicateAccount {
		t.Fatalf(`CreateAccount err: wanted "%+v", got "%+v"`, ErrDuplicateAccount, err)
	}

	differentCaseEmail := auth.Email("aBC@examplE.CoM")

	// Try to create a new account with the same email different capitalization.
	// fail because email already exists
	if err := s.CreateAccount(differentCaseEmail, password, seed, nil); err != ErrDuplicateAccount {
		t.Fatalf(`CreateAccount err (for case insensitivity check): wanted "%+v", got "%+v"`, ErrDuplicateAccount, err)
	}

	// Get the email and same *first* password we successfully put in
	expectAccountMatch(t, &s, normEmail, email, password, seed, nil, nil, time.Now().UTC(), time.Now().UTC())
}

// Test that I can use CreateAccount twice for different emails with no veriy token
// This is related to https://github.com/lbryio/wallet-sync-server/issues/13
func TestStoreCreateAccountTwoVerifiedSucceed(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	email1, normEmail1 := auth.Email("Abc@Example.Com"), auth.NormalizedEmail("abc@example.com")
	password1, seed1 := auth.Password("123"), auth.ClientSaltSeed("abcd1234abcd1234")

	email2, normEmail2 := auth.Email("Abc2@Example.Com"), auth.NormalizedEmail("abc2@example.com")
	password2, seed2 := auth.Password("123"), auth.ClientSaltSeed("abcd1234abcd1234")

	// Create a couple accounts. Don't care if they have the same password.
	// Make them verified (i.e. no token) for the usual
	// case. We'll test unverified (with token) separately.
	if err := s.CreateAccount(email1, password1, seed1, nil); err != nil {
		t.Fatalf("Unexpected error in CreateAccount: %+v", err)
	}

	if err := s.CreateAccount(email2, password2, seed2, nil); err != nil {
		t.Fatalf("Unexpected error in CreateAccount: %+v", err)
	}

	// Get and confirm the accounts we just put in
	expectAccountMatch(t, &s, normEmail1, email1, password1, seed1, nil, nil, time.Now().UTC(), time.Now().UTC())
	expectAccountMatch(t, &s, normEmail2, email2, password2, seed2, nil, nil, time.Now().UTC(), time.Now().UTC())
}

// Test that I cannot use CreateAccount twice with the same verify token, but
//   I can with different verify tokens
// This is related to https://github.com/lbryio/wallet-sync-server/issues/13
func TestStoreCreateAccountTwoSameVerfiyTokenFail(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	email1, normEmail1 := auth.Email("Abc@Example.Com"), auth.NormalizedEmail("abc@example.com")
	password1, seed1 := auth.Password("123"), auth.ClientSaltSeed("abcd1234abcd1234")
	verifyToken1 := auth.VerifyTokenString("abcd1234abcd1234abcd1234abcd1234")

	email2, normEmail2 := auth.Email("Abc2@Example.Com"), auth.NormalizedEmail("abc2@example.com")
	password2, seed2 := auth.Password("xyz"), auth.ClientSaltSeed("abcd1234abcd1234")
	verifyToken2 := auth.VerifyTokenString("00001234abcd1234abcd123400000000")

	// Create the first account
	if err := s.CreateAccount(email1, password1, seed1, &verifyToken1); err != nil {
		t.Fatalf("Unexpected error in CreateAccount: %+v", err)
	}

	// Try to create the second account with the same verify token, fail
	if err := s.CreateAccount(email2, password2, seed2, &verifyToken1); err != ErrDuplicateAccount {
		t.Fatalf(`CreateAccount err: wanted "%+v", got "%+v"`, ErrDuplicateAccount, err)
	}

	// Confirm that it didn't save
	expectAccountNotExists(t, &s, normEmail2)

	// Create the second account with a different verify token
	if err := s.CreateAccount(email2, password2, seed2, &verifyToken2); err != nil {
		t.Fatalf("Unexpected error in CreateAccount: %+v", err)
	}

	// Get and confirm the accounts we just put in
	approxVerifyExpiration := time.Now().Add(time.Hour * 24 * 2).UTC()
	expectAccountMatch(t, &s, normEmail1, email1, password1, seed1, &verifyToken1, &approxVerifyExpiration, time.Now().UTC(), time.Now().UTC())
	expectAccountMatch(t, &s, normEmail2, email2, password2, seed2, &verifyToken2, &approxVerifyExpiration, time.Now().UTC(), time.Now().UTC())
}

// Try CreateAccount with a verification string, thus unverified
func TestStoreCreateAccountUnverified(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	email, normEmail := auth.Email("Abc@Example.Com"), auth.NormalizedEmail("abc@example.com")
	password, seed := auth.Password("123"), auth.ClientSaltSeed("abcd1234abcd1234")

	// Create an account
	verifyToken := auth.VerifyTokenString("abcd1234abcd1234abcd1234abcd1234")
	if err := s.CreateAccount(email, password, seed, &verifyToken); err != nil {
		t.Fatalf("Unexpected error in CreateAccount: %+v", err)
	}

	// Get and confirm the account we just put in
	approxVerifyExpiration := time.Now().Add(time.Hour * 24 * 2).UTC()
	expectAccountMatch(t, &s, normEmail, email, password, seed, &verifyToken, &approxVerifyExpiration, time.Now().UTC(), time.Now().UTC())
}

// Test GetUserId for nonexisting email
func TestStoreGetUserIdAccountNotExists(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	email, password := auth.Email("abc@example.com"), auth.Password("123")

	if userId, err := s.GetUserId(email, password); err != ErrWrongCredentials || userId != 0 {
		t.Fatalf(`GetUserId error for nonexistant account: wanted "%+v", got "%+v. userId: %v"`, ErrWrongCredentials, err, userId)
	}
}

// Test GetUserId for existing account, with the correct and incorrect password
func TestStoreGetUserIdAccountExists(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	createdUserId, email, password, _ := makeTestUser(t, &s, nil, nil)

	// Check that the userId is correct for the email, irrespective of the case of
	// the characters in the email.
	lowerEmail := auth.Email(strings.ToLower(string(email)))
	upperEmail := auth.Email(strings.ToUpper(string(email)))

	// Check that there's now a user id for the email and password
	if userId, err := s.GetUserId(lowerEmail, password); err != nil || userId != createdUserId {
		t.Fatalf("Unexpected error in GetUserId: err: %+v userId: %v", err, userId)
	}

	// Check that there's now a user id for the email and password
	if userId, err := s.GetUserId(upperEmail, password); err != nil || userId != createdUserId {
		t.Fatalf("Unexpected error in GetUserId: err: %+v userId: %v", err, userId)
	}

	// Check that it won't return if the wrong password is given
	if userId, err := s.GetUserId(email, password+auth.Password("_wrong")); err != ErrWrongCredentials || userId != 0 {
		t.Fatalf(`GetUserId error for wrong password: wanted "%+v", got "%+v. userId: %v"`, ErrWrongCredentials, err, userId)
	}
}

// Test GetUserId for existing but unverified account
func TestStoreGetUserIdAccountUnverified(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	verifyToken := auth.VerifyTokenString("abcd1234abcd1234abcd1234abcd1234")
	_, email, password, _ := makeTestUser(t, &s, &verifyToken, &time.Time{})

	// Check that it won't return if the account is unverified
	if userId, err := s.GetUserId(email, password); err != ErrNotVerified || userId != 0 {
		t.Fatalf(`GetUserId error for unverified account: wanted "%+v", got "%+v. userId: %v"`, ErrNotVerified, err, userId)
	}
}

func TestStoreAccountEmptyFields(t *testing.T) {
	// Make sure expiration doesn't get set if sanitization fails
	tt := []struct {
		name           string
		email          auth.Email
		clientSaltSeed auth.ClientSaltSeed
		password       auth.Password
	}{
		{
			name:           "missing email",
			email:          "",
			clientSaltSeed: "abcd1234abcd1234",
			password:       "xyz",
		},
		{
			name:           "missing client salt seed",
			email:          "a@example.com",
			clientSaltSeed: "",
			password:       "xyz",
		},
		// Not testing empty key and salt because they get generated to something
		// non-empty in the method, even if email is empty
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s, sqliteTmpFile := StoreTestInit(t)
			defer StoreTestCleanup(sqliteTmpFile)

			var sqliteErr sqlite3.Error

			err := s.CreateAccount(tc.email, tc.password, tc.clientSaltSeed, nil)
			if errors.As(err, &sqliteErr) {
				if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintCheck) {
					return // We got the error we expected
				}
			}
			t.Errorf("Expected check constraint error for empty field. Got %+v", err)
		})
	}
}

// Test GetClientSaltSeed for existing account
func TestStoreGetClientSaltSeedAccountExists(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	_, email, _, createdSeed := makeTestUser(t, &s, nil, nil)

	// Check that the seed is correct for the email, irrespective of the case of
	// the characters in the email.
	lowerEmail := auth.Email(strings.ToLower(string(email)))
	upperEmail := auth.Email(strings.ToUpper(string(email)))

	if seed, err := s.GetClientSaltSeed(lowerEmail); err != nil || seed != createdSeed {
		t.Fatalf("Unexpected error in GetClientSaltSeed: err: %+v seed: %v", err, seed)
	}
	if seed, err := s.GetClientSaltSeed(upperEmail); err != nil || seed != createdSeed {
		t.Fatalf("Unexpected error in GetClientSaltSeed: err: %+v seed: %v", err, seed)
	}
}

// Test GetClientSaltSeed for nonexisting email
func TestStoreGetClientSaltSeedAccountNotExists(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	email := auth.Email("abc@example.com")

	if seed, err := s.GetClientSaltSeed(email); err != ErrWrongCredentials || seed != "" {
		t.Fatalf(`GetClientSaltSeed error for nonexistant account: wanted "%+v", got "%+v. seed: %v"`, ErrWrongCredentials, err, seed)
	}
}

// Test UpdateVerifyTokenString for existing account
func TestUpdateVerifyTokenStringSuccess(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	verifyTokenString1 := auth.VerifyTokenString("00000000000000000000000000000000")
	time1 := time.Time{}

	_, email, password, createdSeed := makeTestUser(t, &s, &verifyTokenString1, &time1)

	// we're not testing normalization features so we'll just use this here
	normEmail := email.Normalize()

	// Check that the token updates for the email, irrespective of the case of
	// the characters in the email.
	lowerEmail := auth.Email(strings.ToLower(string(email)))
	upperEmail := auth.Email(strings.ToUpper(string(email)))

	verifyTokenString2 := auth.VerifyTokenString("abcd1234abcd1234abcd1234abcd1234")
	verifyTokenString3 := auth.VerifyTokenString("ef095678ef095678ef095678ef095678")
	approxVerifyExpiration := time.Now().Add(time.Hour * 24 * 2).UTC()

	if err := s.UpdateVerifyTokenString(lowerEmail, verifyTokenString2); err != nil {
		t.Fatalf("Unexpected error in UpdateVerifyTokenString: err: %+v", err)
	}
	expectAccountMatch(t, &s, normEmail, email, password, createdSeed, &verifyTokenString2, &approxVerifyExpiration, time.Now().UTC(), time.Now().UTC())

	if err := s.UpdateVerifyTokenString(upperEmail, verifyTokenString3); err != nil {
		t.Fatalf("Unexpected error in UpdateVerifyTokenString: err: %+v", err)
	}
	expectAccountMatch(t, &s, normEmail, email, password, createdSeed, &verifyTokenString3, &approxVerifyExpiration, time.Now().UTC(), time.Now().UTC())
}

// Test UpdateVerifyTokenString for nonexisting email
func TestStoreUpdateVerifyTokenStringAccountNotExists(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	email := auth.Email("abc@example.com")

	if err := s.UpdateVerifyTokenString(email, "abcd1234abcd1234abcd1234abcd1234"); err != ErrWrongCredentials {
		t.Fatalf(`UpdateVerifyTokenString error for nonexistant account: wanted "%+v", got "%+v."`, ErrWrongCredentials, err)
	}
}

// Test UpdateVerifyTokenString for already verified account
func TestStoreUpdateVerifyTokenStringAccountVerified(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	_, email, _, _ := makeTestUser(t, &s, nil, nil)

	if err := s.UpdateVerifyTokenString(email, "abcd1234abcd1234abcd1234abcd1234"); err != ErrNoTokenForUser {
		t.Fatalf(`UpdateVerifyTokenString error for already verified account: wanted "%+v", got "%+v."`, ErrNoTokenForUser, err)
	}
}

// Test VerifyAccount for existing account
func TestUpdateVerifyAccountSuccess(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	verifyTokenString := auth.VerifyTokenString("abcd1234abcd1234abcd1234abcd1234")
	verifyExpiration := time.Now().Add(time.Second * 10).UTC() // expires in one second

	_, email, password, createdSeed := makeTestUser(t, &s, &verifyTokenString, &verifyExpiration)

	// we're not testing normalization features so we'll just use this here
	normEmail := email.Normalize()

	if err := s.VerifyAccount(verifyTokenString); err != nil {
		t.Fatalf("Unexpected error in VerifyAccount: err: %+v", err)
	}
	expectAccountMatch(t, &s, normEmail, email, password, createdSeed, nil, nil, time.Now().UTC(), time.Now().UTC())
}

// Test VerifyAccount for nonexisting token
func TestStoreVerifyAccountTokenNotExists(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	if err := s.VerifyAccount("abcd1234abcd1234abcd1234abcd1234"); err != ErrNoTokenForUser {
		t.Fatalf(`VerifyAccount error for nonexistant token: wanted "%+v", got "%+v."`, ErrNoTokenForUser, err)
	}
}

// Test VerifyAccount for expired token
func TestUpdateVerifyAccountTokenExpired(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	verifyTokenString := auth.VerifyTokenString("abcd1234abcd1234abcd1234abcd1234")
	verifyExpiration := time.Now().Add(time.Second * (-1)).UTC() // expired one second ago

	_, email, password, createdSeed := makeTestUser(t, &s, &verifyTokenString, &verifyExpiration)

	// we're not testing normalization features so we'll just use this here
	normEmail := email.Normalize()

	if err := s.VerifyAccount(verifyTokenString); err != ErrNoTokenForUser {
		t.Fatalf(`VerifyAccount error for expired token: wanted "%+v", got "%+v."`, ErrNoTokenForUser, err)
	}

	expectAccountMatch(t, &s, normEmail, email, password, createdSeed, &verifyTokenString, &verifyExpiration, time.Now().UTC(), time.Now().UTC())
}
