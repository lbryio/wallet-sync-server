package store

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"orblivion/lbry-id/auth"
)

func expectTokenExists(t *testing.T, s *Store, expectedToken auth.AuthToken) {
	rows, err := s.db.Query("SELECT * FROM auth_tokens WHERE token=?", expectedToken.Token)
	if err != nil {
		t.Fatalf("Error finding token for: %s - %+v", expectedToken.Token, err)
	}
	defer rows.Close()

	var gotToken auth.AuthToken
	for rows.Next() {

		err := rows.Scan(
			&gotToken.Token,
			&gotToken.UserId,
			&gotToken.DeviceId,
			&gotToken.Scope,
			&gotToken.Expiration,
		)

		if err != nil {
			t.Fatalf("Error finding token for: %s - %+v", expectedToken.Token, err)
		}

		if !reflect.DeepEqual(gotToken, expectedToken) {
			t.Fatalf("token: \n  expected %+v\n  got:     %+v", expectedToken, gotToken)
		}

		return // found a match, we're good
	}
	t.Fatalf("Expected token for: %s", expectedToken.Token)
}

func expectTokenNotExists(t *testing.T, s *Store, token auth.TokenString) {
	rows, err := s.db.Query("SELECT * FROM auth_tokens WHERE token=?", token)
	if err != nil {
		t.Fatalf("Error finding (lack of) token for: %s - %+v", token, err)
	}
	defer rows.Close()

	var gotToken auth.AuthToken
	for rows.Next() {

		err := rows.Scan(
			&gotToken.Token,
			&gotToken.UserId,
			&gotToken.DeviceId,
			&gotToken.Scope,
			&gotToken.Expiration,
		)

		if err != nil {
			t.Fatalf("Error finding (lack of) token for: %s - %+v", token, err)
		}

		t.Fatalf("Expected no token. Got: %+v", gotToken)
	}
	return // found nothing, we're good
}

// Test insertToken, using GetToken as a helper
// Try insertToken twice with the same user and device, error the second time
func TestStoreInsertToken(t *testing.T) {

	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// created for addition to the DB (no expiration attached)
	authToken1 := auth.AuthToken{
		Token:    "seekrit-1",
		DeviceId: "dId",
		Scope:    "*",
		UserId:   123,
	}
	expiration := time.Now().Add(time.Hour * 24 * 14).UTC()

	// Try to get a token, come back empty because we're just starting out
	expectTokenNotExists(t, &s, authToken1.Token)

	// Put in a token
	if err := s.insertToken(&authToken1, expiration); err != nil {
		t.Fatalf("Unexpected error in insertToken: %+v", err)
	}

	// The value expected when we pull it from the database.
	authToken1Expected := authToken1
	authToken1Expected.Expiration = &expiration

	// Get and confirm the token we just put in
	expectTokenExists(t, &s, authToken1Expected)

	// Try to put a different token, fail because we already have one
	authToken2 := authToken1
	authToken2.Token = "seekrit-2"

	if err := s.insertToken(&authToken2, expiration); err != ErrDuplicateToken {
		t.Fatalf(`insertToken err: wanted "%+v", got "%+v"`, ErrDuplicateToken, err)
	}

	// Get the same *first* token we successfully put in
	expectTokenExists(t, &s, authToken1Expected)
}

// Test updateToken, using GetToken and insertToken as helpers
// Try updateToken with no existing token, err for lack of anything to update
// Try updateToken with a preexisting token, succeed
// Try updateToken again with a new token, succeed
func TestStoreUpdateToken(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// created for addition to the DB (no expiration attached)
	authTokenUpdate := auth.AuthToken{
		Token:    "seekrit-update",
		DeviceId: "dId",
		Scope:    "*",
		UserId:   123,
	}
	expiration := time.Now().Add(time.Hour * 24 * 14).UTC()

	// Try to get a token, come back empty because we're just starting out
	expectTokenNotExists(t, &s, authTokenUpdate.Token)

	// Try to update the token - fail because we don't have an entry there in the first place
	if err := s.updateToken(&authTokenUpdate, expiration); err != ErrNoToken {
		t.Fatalf(`updateToken err: wanted "%+v", got "%+v"`, ErrNoToken, err)
	}

	// Try to get a token, come back empty because the update attempt failed to do anything
	expectTokenNotExists(t, &s, authTokenUpdate.Token)

	// Put in a different token, just so we have something to test that
	// updateToken overwrites it
	authTokenInsert := authTokenUpdate
	authTokenInsert.Token = "seekrit-insert"

	if err := s.insertToken(&authTokenInsert, expiration); err != nil {
		t.Fatalf("Unexpected error in insertToken: %+v", err)
	}

	// Now successfully update token
	if err := s.updateToken(&authTokenUpdate, expiration); err != nil {
		t.Fatalf("Unexpected error in updateToken: %+v", err)
	}

	// The value expected when we pull it from the database.
	authTokenUpdateExpected := authTokenUpdate
	authTokenUpdateExpected.Expiration = &expiration

	// Get and confirm the token we just put in
	expectTokenExists(t, &s, authTokenUpdateExpected)

	// Fail to get the token we previously inserted, because it's now been overwritten
	expectTokenNotExists(t, &s, authTokenInsert.Token)
}

// Test that a user can have two different devices.
// Test first and second Save (one for insert, one for update)
// Get fails initially
// Put token1-d1 token1-d2
// Get token1-d1 token1-d2
// Put token2-d1 token2-d2
// Get token2-d1 token2-d2
func TestStoreSaveToken(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// Version 1 of the token for both devices
	// created for addition to the DB (no expiration attached)
	authToken_d1_1 := auth.AuthToken{
		Token:    "seekrit-d1-1",
		DeviceId: "dId-1",
		Scope:    "*",
		UserId:   123,
	}

	authToken_d2_1 := authToken_d1_1
	authToken_d2_1.DeviceId = "dId-2"
	authToken_d2_1.Token = "seekrit-d2-1"

	// Try to get the tokens, come back empty because we're just starting out
	expectTokenNotExists(t, &s, authToken_d1_1.Token)
	expectTokenNotExists(t, &s, authToken_d2_1.Token)

	// Save Version 1 tokens for both devices
	if err := s.SaveToken(&authToken_d1_1); err != nil {
		t.Fatalf("Unexpected error in SaveToken: %+v", err)
	}
	if err := s.SaveToken(&authToken_d2_1); err != nil {
		t.Fatalf("Unexpected error in SaveToken: %+v", err)
	}

	// Check one of the authTokens to make sure expiration was set
	if authToken_d1_1.Expiration == nil {
		t.Fatalf("Expected SaveToken to set an Expiration")
	}
	nowDiff := authToken_d1_1.Expiration.Sub(time.Now().UTC())
	if time.Hour*24*14+time.Minute < nowDiff || nowDiff < time.Hour*24*14-time.Minute {
		t.Fatalf("Expected SaveToken to set a token Expiration 2 weeks in the future.")
	}

	// Get and confirm the tokens we just put in
	expectTokenExists(t, &s, authToken_d1_1)
	expectTokenExists(t, &s, authToken_d2_1)

	// Version 2 of the token for both devices
	authToken_d1_2 := authToken_d1_1
	authToken_d1_2.Token = "seekrit-d1-2"

	authToken_d2_2 := authToken_d2_1
	authToken_d2_2.Token = "seekrit-d2-2"

	// Save Version 2 tokens for both devices
	if err := s.SaveToken(&authToken_d1_2); err != nil {
		t.Fatalf("Unexpected error in SaveToken: %+v", err)
	}
	if err := s.SaveToken(&authToken_d2_2); err != nil {
		t.Fatalf("Unexpected error in SaveToken: %+v", err)
	}

	// Check that the expiration of this new token is marginally later
	if authToken_d1_2.Expiration == nil {
		t.Fatalf("Expected SaveToken to set an Expiration")
	}
	expDiff := authToken_d1_2.Expiration.Sub(*authToken_d1_1.Expiration)
	if time.Second < expDiff || expDiff < 0 {
		t.Fatalf("Expected new expiration to be slightly later than previous expiration. diff: %+v", expDiff)
	}

	// Get and confirm the tokens we just put in
	expectTokenExists(t, &s, authToken_d1_2)
	expectTokenExists(t, &s, authToken_d2_2)

	// Confirm the old ones are gone
	expectTokenNotExists(t, &s, authToken_d1_1.Token)
	expectTokenNotExists(t, &s, authToken_d2_1.Token)
}

// test GetToken using insertToken and updateToken as helpers (so we can set expiration timestamps)
// normal
// token not found
// expired not returned
func TestStoreGetToken(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// created for addition to the DB (no expiration attached)
	authToken := auth.AuthToken{
		Token:    "seekrit-d1",
		DeviceId: "dId",
		Scope:    "*",
		UserId:   123,
	}
	expiration := time.Time(time.Now().UTC().Add(time.Hour * 24 * 14))

	// Not found (nothing saved for this token string)
	gotToken, err := s.GetToken(authToken.Token)
	if gotToken != nil || err != ErrNoToken {
		t.Fatalf("Expected ErrNoToken. token: %+v err: %+v", gotToken, err)
	}

	// Put in a token
	if err := s.insertToken(&authToken, expiration); err != nil {
		t.Fatalf("Unexpected error in insertToken: %+v", err)
	}

	// The value expected when we pull it from the database.
	authTokenExpected := authToken
	authTokenExpected.Expiration = &expiration

	// Confirm it saved
	gotToken, err = s.GetToken(authToken.Token)
	if err != nil {
		t.Fatalf("Unexpected error in GetToken: %+v", err)
	}
	if gotToken == nil || !reflect.DeepEqual(*gotToken, authTokenExpected) {
		t.Fatalf("token: \n  expected %+v\n  got:     %+v", authTokenExpected, gotToken)
	}

	// Update the token to be expired
	expirationOld := time.Now().Add(time.Second * (-1))
	if err := s.updateToken(&authToken, expirationOld); err != nil {
		t.Fatalf("Unexpected error in updateToken: %+v", err)
	}

	// Fail to get the expired token
	gotToken, err = s.GetToken(authToken.Token)
	if gotToken != nil || err != ErrNoToken {
		t.Fatalf("Expected ErrNoToken, for expired token. token: %+v err: %+v", gotToken, err)
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

// TODO - Tests each db method. Check for missing "NOT NULL" fields. Do the loop thing, and always just check for null error.
func TestStoreTokenEmptyFields(t *testing.T) {
	// Make sure expiration doesn't get set if sanitization fails
	t.Fatalf("Test me")
}
