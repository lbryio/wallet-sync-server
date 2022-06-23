package store

import (
	"orblivion/lbry-id/auth"
	"reflect"
	"testing"
	"time"
)

// Test insertToken, using GetToken as a helper
// Try insertToken twice with the same public key, error the second time
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

	// Get a token, come back empty
	gotToken, err := s.GetToken(authToken1.Token)
	if gotToken != nil || err != ErrNoToken {
		t.Fatalf("Expected ErrNoToken. token: %+v err: %+v", gotToken, err)
	}

	// Put in a token
	if err := s.insertToken(&authToken1, expiration); err != nil {
		t.Fatalf("Unexpected error in insertToken: %+v", err)
	}

	// The value expected when we pull it from the database.
	authToken1Expected := authToken1
	authToken1Expected.Expiration = &expiration

	// Get and confirm the token we just put in
	gotToken, err = s.GetToken(authToken1.Token)
	if err != nil {
		t.Fatalf("Unexpected error in GetToken: %+v", err)
	}
	if gotToken == nil || !reflect.DeepEqual(*gotToken, authToken1Expected) {
		t.Fatalf("token: \n  expected %+v\n  got:     %+v", authToken1Expected, *gotToken)
	}

	// Try to put a different token, fail because we already have one
	authToken2 := authToken1
	authToken2.Token = "seekrit-2"

	if err := s.insertToken(&authToken2, expiration); err != ErrDuplicateToken {
		t.Fatalf(`insertToken err: wanted "%+v", got "%+v"`, ErrDuplicateToken, err)
	}

	// Get the same *first* token we successfully put in
	gotToken, err = s.GetToken(authToken1.Token)
	if err != nil {
		t.Fatalf("Unexpected error in GetToken: %+v", err)
	}
	if gotToken == nil || !reflect.DeepEqual(*gotToken, authToken1Expected) {
		t.Fatalf("token: expected %+v, got: %+v", authToken1Expected, gotToken)
	}
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
	gotToken, err := s.GetToken(authTokenUpdate.Token)
	if gotToken != nil || err != ErrNoToken {
		t.Fatalf("Expected ErrNoToken. token: %+v err: %+v", gotToken, err)
	}

	// Try to update the token - fail because we don't have an entry there in the first place
	if err := s.updateToken(&authTokenUpdate, expiration); err != ErrNoToken {
		t.Fatalf(`updateToken err: wanted "%+v", got "%+v"`, ErrNoToken, err)
	}

	// Try to get a token, come back empty because the update attempt failed to do anything
	gotToken, err = s.GetToken(authTokenUpdate.Token)
	if gotToken != nil || err != ErrNoToken {
		t.Fatalf("Expected ErrNoToken. token: %+v err: %+v", gotToken, err)
	}

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
	gotToken, err = s.GetToken(authTokenUpdate.Token)
	if err != nil {
		t.Fatalf("Unexpected error in GetToken: %+v", err)
	}
	if gotToken == nil || !reflect.DeepEqual(*gotToken, authTokenUpdateExpected) {
		t.Fatalf("token: \n  expected %+v\n  got:     %+v", authTokenUpdateExpected, *gotToken)
	}

	// Fail to get the token we previously inserted, because it's now been overwritten
	gotToken, err = s.GetToken(authTokenInsert.Token)
	if gotToken != nil || err != ErrNoToken {
		t.Fatalf("Expected ErrNoToken. token: %+v err: %+v", gotToken, err)
	}
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
	gotToken, err := s.GetToken(authToken_d1_1.Token)
	if gotToken != nil || err != ErrNoToken {
		t.Fatalf("Expected ErrNoToken. token: %+v err: %+v", gotToken, err)
	}
	gotToken, err = s.GetToken(authToken_d2_1.Token)
	if gotToken != nil || err != ErrNoToken {
		t.Fatalf("Expected ErrNoToken. token: %+v err: %+v", gotToken, err)
	}

	// Save Version 1 tokens for both devices
	if err = s.SaveToken(&authToken_d1_1); err != nil {
		t.Fatalf("Unexpected error in SaveToken: %+v", err)
	}
	if err = s.SaveToken(&authToken_d2_1); err != nil {
		t.Fatalf("Unexpected error in SaveToken: %+v", err)
	}

	// Check one of the authTokens to make sure expiration was set
	if authToken_d1_1.Expiration == nil {
		t.Fatalf("Expected SaveToken to set an Expiration")
	}
	nowDiff := authToken_d1_1.Expiration.Sub(time.Now())
	if time.Hour*24*14+time.Minute < nowDiff || nowDiff < time.Hour*24*14-time.Minute {
		t.Fatalf("Expected SaveToken to set a token Expiration 2 weeks in the future.")
	}

	// Get and confirm the tokens we just put in
	gotToken, err = s.GetToken(authToken_d1_1.Token)
	if err != nil {
		t.Fatalf("Unexpected error in GetToken: %+v", err)
	}
	if gotToken == nil || !reflect.DeepEqual(*gotToken, authToken_d1_1) {
		t.Fatalf("token: \n  expected %+v\n  got:    %+v", authToken_d1_1, gotToken)
	}
	gotToken, err = s.GetToken(authToken_d2_1.Token)
	if err != nil {
		t.Fatalf("Unexpected error in GetToken: %+v", err)
	}
	if gotToken == nil || !reflect.DeepEqual(*gotToken, authToken_d2_1) {
		t.Fatalf("token: expected %+v, got: %+v", authToken_d2_1, gotToken)
	}

	// Version 2 of the token for both devices
	authToken_d1_2 := authToken_d1_1
	authToken_d1_2.Token = "seekrit-d1-2"

	authToken_d2_2 := authToken_d2_1
	authToken_d2_2.Token = "seekrit-d2-2"

	// Save Version 2 tokens for both devices
	if err = s.SaveToken(&authToken_d1_2); err != nil {
		t.Fatalf("Unexpected error in SaveToken: %+v", err)
	}
	if err = s.SaveToken(&authToken_d2_2); err != nil {
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
	gotToken, err = s.GetToken(authToken_d1_2.Token)
	if err != nil {
		t.Fatalf("Unexpected error in GetToken: %+v", err)
	}
	if gotToken == nil || !reflect.DeepEqual(*gotToken, authToken_d1_2) {
		t.Fatalf("token: \n  expected %+v\n  got:    %+v", authToken_d1_2, gotToken)
	}
	gotToken, err = s.GetToken(authToken_d2_2.Token)
	if err != nil {
		t.Fatalf("Unexpected error in GetToken: %+v", err)
	}
	if gotToken == nil || !reflect.DeepEqual(*gotToken, authToken_d2_2) {
		t.Fatalf("token: expected %+v, got: %+v", authToken_d2_2, gotToken)
	}
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

func TestStoreSanitizeEmptyFields(t *testing.T) {
	// Make sure expiration doesn't get set if sanitization fails
	t.Fatalf("Test me")
}

func TestStoreTimeZones(t *testing.T) {
	// Make sure the tz situation is as we prefer in the DB unless we just do UTC.
	t.Fatalf("Test me")
}

func TestStoreSetWalletSuccess(t *testing.T) {
	/*
	  Sequence 1 - works via insert
	  Sequence 2 - works via update
	  Sequence 3 - works via update
	*/
	t.Fatalf("Test me: Wallet Set successes")
}

func TestStoreSetWalletFail(t *testing.T) {
	/*
	  Sequence 1 - fails via insert - fail by having something there already
	  Sequence 2 - fails via update - fail by not having something there already
	  Sequence 3 - fails via update - fail by having something with wrong sequence number
	  Sequence 4 - fails via update - fail by having something with non-matching device sequence history

	  Maybe some of the above gets put off to wallet util
	*/
	t.Fatalf("Test me: Wallet Set failures")
}

func TestStoreInsertWalletSuccess(t *testing.T) {
	t.Fatalf("Test me: Wallet insert successes")
}

func TestStoreInsertWalletFail(t *testing.T) {
	t.Fatalf("Test me: Wallet insert failures")
}

func TestStoreUpdateWalletSuccess(t *testing.T) {
	t.Fatalf("Test me: Wallet update successes")
}

func TestStoreUpdateWalletFail(t *testing.T) {
	t.Fatalf("Test me: Wallet update failures")
}

func TestStoreGetWalletSuccess(t *testing.T) {
	t.Fatalf("Test me: Wallet get success")
}

func TestStoreGetWalletFail(t *testing.T) {
	t.Fatalf("Test me: Wallet get failures")
}

func TestStoreCreateAccount(t *testing.T) {
	t.Fatalf("Test me: Account create success and failures")
}

func TestStoreGetUserId(t *testing.T) {
	t.Fatalf("Test me: User ID get success and failures")
}
