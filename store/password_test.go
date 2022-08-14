package store

import (
	"strings"
	"testing"
	"time"

	"lbryio/lbry-id/auth"
	"lbryio/lbry-id/wallet"
)

// It involves both wallet and account tables. Should it go in wallet_test.go
// or account_test.go? Decided to just make it its own file.

func TestStoreChangePasswordSuccess(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	userId, email, oldPassword, _ := makeTestUser(t, &s, nil, nil)
	token := auth.AuthTokenString("my-token")

	_, err := s.db.Exec(
		"INSERT INTO auth_tokens (token, user_id, device_id, scope, expiration) VALUES(?,?,?,?,?)",
		token, userId, "my-dev-id", "*", time.Now().UTC().Add(time.Hour*24*14),
	)
	if err != nil {
		t.Fatalf("Error creating token")
	}

	_, err = s.db.Exec(
		"INSERT INTO wallets (user_id, encrypted_wallet, sequence, hmac) VALUES(?,?,?,?)",
		userId, "my-enc-wallet", 1, "my-hmac",
	)
	if err != nil {
		t.Fatalf("Error creating test wallet")
	}

	newPassword := oldPassword + auth.Password("_new")
	newSeed := auth.ClientSaltSeed("edf98765edf98765edf98765edf98765edf98765edf98765edf98765edf98765")
	encryptedWallet := wallet.EncryptedWallet("my-enc-wallet-2")
	sequence := wallet.Sequence(2)
	hmac := wallet.WalletHmac("my-hmac-2")

	lowerEmail := auth.Email(strings.ToLower(string(email)))

	if err := s.ChangePasswordWithWallet(lowerEmail, oldPassword, newPassword, newSeed, encryptedWallet, sequence, hmac); err != nil {
		t.Errorf("ChangePasswordWithWallet (lower case email): unexpected error: %+v", err)
	}

	expectAccountMatch(t, &s, email.Normalize(), email, newPassword, newSeed, nil, nil)
	expectWalletExists(t, &s, userId, encryptedWallet, sequence, hmac)
	expectTokenNotExists(t, &s, token)

	newNewPassword := newPassword + auth.Password("_new")
	newNewSeed := auth.ClientSaltSeed("00008765edf98765edf98765edf98765edf98765edf98765edf98765edf98765")
	newEncryptedWallet := wallet.EncryptedWallet("my-enc-wallet-3")
	newSequence := wallet.Sequence(3)
	newHmac := wallet.WalletHmac("my-hmac-3")

	upperEmail := auth.Email(strings.ToUpper(string(email)))

	if err := s.ChangePasswordWithWallet(upperEmail, newPassword, newNewPassword, newNewSeed, newEncryptedWallet, newSequence, newHmac); err != nil {
		t.Errorf("ChangePasswordWithWallet (upper case email): unexpected error: %+v", err)
	}

	expectAccountMatch(t, &s, email.Normalize(), email, newNewPassword, newNewSeed, nil, nil)
}

func TestStoreChangePasswordErrors(t *testing.T) {
	verifyToken := auth.VerifyTokenString("aoeu1234aoeu1234aoeu1234aoeu1234")
	tt := []struct {
		name              string
		hasWallet         bool
		sequence          wallet.Sequence
		emailSuffix       auth.Email
		oldPasswordSuffix auth.Password
		verifyToken       *auth.VerifyTokenString
		verifyExpiration  *time.Time
		expectedError     error
	}{
		{
			name:              "wrong email",
			hasWallet:         true,                 // we have the requisite wallet
			sequence:          wallet.Sequence(2),   // sequence is correct
			emailSuffix:       auth.Email("_wrong"), // the email is *incorrect*
			oldPasswordSuffix: auth.Password(""),    // the password is correct
			expectedError:     ErrWrongCredentials,
		}, {
			name:              "unverified account",
			hasWallet:         true,               // we have the requisite wallet (even though it should be impossible for an unverified account)
			sequence:          wallet.Sequence(2), // sequence is correct
			emailSuffix:       auth.Email(""),     // the email is correct
			oldPasswordSuffix: auth.Password(""),  // the password is correct
			verifyToken:       &verifyToken,
			verifyExpiration:  &time.Time{},
			expectedError:     ErrNotVerified,
		}, {
			name:              "wrong old password",
			hasWallet:         true,                    // we have the requisite wallet
			sequence:          wallet.Sequence(2),      // sequence is correct
			emailSuffix:       auth.Email(""),          // the email is correct
			oldPasswordSuffix: auth.Password("_wrong"), // the old password is *incorrect*
			expectedError:     ErrWrongCredentials,
		}, {
			name:              "wrong sequence",
			hasWallet:         true,               // we have the requisite wallet
			sequence:          wallet.Sequence(3), // sequence is *incorrect*
			emailSuffix:       auth.Email(""),     // the email is correct
			oldPasswordSuffix: auth.Password(""),  // the password is correct
			expectedError:     ErrWrongSequence,
		}, {
			name:              "no wallet to replace",
			hasWallet:         false,              // we have the requisite wallet
			sequence:          wallet.Sequence(1), // sequence is correct (for there being no wallets)
			emailSuffix:       auth.Email(""),     // the email is correct
			oldPasswordSuffix: auth.Password(""),  // the password is correct

			// Sequence=1 always ends up being wrong for this endpoint since we
			// should never be creating a wallet here.
			expectedError: ErrWrongSequence,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s, sqliteTmpFile := StoreTestInit(t)
			defer StoreTestCleanup(sqliteTmpFile)

			userId, email, oldPassword, oldSeed := makeTestUser(t, &s, tc.verifyToken, tc.verifyExpiration)
			expiration := time.Now().UTC().Add(time.Hour * 24 * 14)
			authToken := auth.AuthToken{
				Token:      auth.AuthTokenString("my-token"),
				DeviceId:   auth.DeviceId("my-dev-id"),
				UserId:     userId,
				Scope:      auth.AuthScope("*"),
				Expiration: &expiration,
			}

			_, err := s.db.Exec(
				"INSERT INTO auth_tokens (token, user_id, device_id, scope, expiration) VALUES(?,?,?,?,?)",
				authToken.Token, authToken.UserId, authToken.DeviceId, authToken.Scope, authToken.Expiration,
			)
			if err != nil {
				t.Fatalf("Error creating token")
			}

			oldEncryptedWallet := wallet.EncryptedWallet("my-enc-wallet-old")
			newEncryptedWallet := wallet.EncryptedWallet("my-enc-wallet-new")
			oldHmac := wallet.WalletHmac("my-hmac-old")
			newHmac := wallet.WalletHmac("my-hmac-new")
			oldSequence := wallet.Sequence(1)

			if tc.hasWallet {
				_, err := s.db.Exec(
					"INSERT INTO wallets (user_id, encrypted_wallet, sequence, hmac) VALUES(?,?,?,?)",
					userId, oldEncryptedWallet, oldSequence, oldHmac,
				)
				if err != nil {
					t.Fatalf("Error creating test wallet")
				}
			}

			submittedEmail := email + tc.emailSuffix                   // Possibly make it the wrong email
			submittedOldPassword := oldPassword + tc.oldPasswordSuffix // Possibly make it the wrong password
			newPassword := oldPassword + auth.Password("_new")         // Make the new password different (as it should be)
			newSeed := auth.ClientSaltSeed("edf98765edf98765edf98765edf98765edf98765edf98765edf98765edf98765")

			if err := s.ChangePasswordWithWallet(submittedEmail, submittedOldPassword, newPassword, newSeed, newEncryptedWallet, tc.sequence, newHmac); err != tc.expectedError {
				t.Errorf("ChangePasswordWithWallet: unexpected value for err. want: %+v, got: %+v", tc.expectedError, err)
			}

			// The password and wallet didn't change, the token didn't get deleted.
			// This tests the transaction rollbacks in particular, given the errors
			// that are at a couple different stages of the txn, triggered by these
			// tests.
			expectAccountMatch(t, &s, email.Normalize(), email, oldPassword, oldSeed, tc.verifyToken, tc.verifyExpiration)
			if tc.hasWallet {
				expectWalletExists(t, &s, userId, oldEncryptedWallet, oldSequence, oldHmac)
			} else {
				expectWalletNotExists(t, &s, userId)
			}
			expectTokenExists(t, &s, authToken)
		})
	}
}

func TestStoreChangePasswordNoWalletSuccess(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	userId, email, oldPassword, _ := makeTestUser(t, &s, nil, nil)
	token := auth.AuthTokenString("my-token")

	_, err := s.db.Exec(
		"INSERT INTO auth_tokens (token, user_id, device_id, scope, expiration) VALUES(?,?,?,?,?)",
		token, userId, "my-dev-id", "*", time.Now().UTC().Add(time.Hour*24*14),
	)
	if err != nil {
		t.Fatalf("Error creating token")
	}

	newPassword := oldPassword + auth.Password("_new")
	newSeed := auth.ClientSaltSeed("edf98765edf98765edf98765edf98765edf98765edf98765edf98765edf98765")

	lowerEmail := auth.Email(strings.ToLower(string(email)))

	if err := s.ChangePasswordNoWallet(lowerEmail, oldPassword, newPassword, newSeed); err != nil {
		t.Errorf("ChangePasswordNoWallet (lower case email): unexpected error: %+v", err)
	}

	expectAccountMatch(t, &s, email.Normalize(), email, newPassword, newSeed, nil, nil)
	expectWalletNotExists(t, &s, userId)
	expectTokenNotExists(t, &s, token)

	newNewPassword := newPassword + auth.Password("_new")
	newNewSeed := auth.ClientSaltSeed("00008765edf98765edf98765edf98765edf98765edf98765edf98765edf98765")

	upperEmail := auth.Email(strings.ToUpper(string(email)))

	if err := s.ChangePasswordNoWallet(upperEmail, newPassword, newNewPassword, newNewSeed); err != nil {
		t.Errorf("ChangePasswordNoWallet (upper case email): unexpected error: %+v", err)
	}

	expectAccountMatch(t, &s, email.Normalize(), email, newNewPassword, newNewSeed, nil, nil)
}

func TestStoreChangePasswordNoWalletErrors(t *testing.T) {
	verifyToken := auth.VerifyTokenString("aoeu1234aoeu1234aoeu1234aoeu1234")

	tt := []struct {
		name              string
		hasWallet         bool
		emailSuffix       auth.Email
		oldPasswordSuffix auth.Password
		verifyToken       *auth.VerifyTokenString
		verifyExpiration  *time.Time
		expectedError     error
	}{
		{
			name:              "wrong email",
			hasWallet:         false,                // we don't have the wallet, as expected for this function
			emailSuffix:       auth.Email("_wrong"), // the email is *incorrect*
			oldPasswordSuffix: auth.Password(""),    // the password is correct
			expectedError:     ErrWrongCredentials,
		}, {
			name:              "wrong old password",
			hasWallet:         false,                   // we don't have the wallet, as expected for this function
			emailSuffix:       auth.Email(""),          // the email is correct
			oldPasswordSuffix: auth.Password("_wrong"), // the old password is *incorrect*
			expectedError:     ErrWrongCredentials,
		}, {
			name:              "unverified account",
			hasWallet:         false,             // we don't have the wallet, as expected for this function
			emailSuffix:       auth.Email(""),    // the email is correct
			oldPasswordSuffix: auth.Password(""), // the password is correct
			verifyToken:       &verifyToken,
			verifyExpiration:  &time.Time{},
			expectedError:     ErrNotVerified,
		}, {
			name:              "unexpected wallet",
			hasWallet:         true,              // we have a wallet which we shouldn't have at this point
			emailSuffix:       auth.Email(""),    // the email is correct
			oldPasswordSuffix: auth.Password(""), // the password is correct
			expectedError:     ErrUnexpectedWallet,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s, sqliteTmpFile := StoreTestInit(t)
			defer StoreTestCleanup(sqliteTmpFile)

			userId, email, oldPassword, oldSeed := makeTestUser(t, &s, tc.verifyToken, tc.verifyExpiration)
			expiration := time.Now().UTC().Add(time.Hour * 24 * 14)
			authToken := auth.AuthToken{
				Token:      auth.AuthTokenString("my-token"),
				DeviceId:   auth.DeviceId("my-dev-id"),
				UserId:     userId,
				Scope:      auth.AuthScope("*"),
				Expiration: &expiration,
			}

			_, err := s.db.Exec(
				"INSERT INTO auth_tokens (token, user_id, device_id, scope, expiration) VALUES(?,?,?,?,?)",
				authToken.Token, authToken.UserId, authToken.DeviceId, authToken.Scope, authToken.Expiration,
			)
			if err != nil {
				t.Fatalf("Error creating token")
			}

			// Only for error case
			encryptedWallet := wallet.EncryptedWallet("my-enc-wallet-old")
			hmac := wallet.WalletHmac("my-hmac-old")
			sequence := wallet.Sequence(1)

			if tc.hasWallet {
				_, err := s.db.Exec(
					"INSERT INTO wallets (user_id, encrypted_wallet, sequence, hmac) VALUES(?,?,?,?)",
					userId, encryptedWallet, sequence, hmac,
				)
				if err != nil {
					t.Fatalf("Error creating test wallet")
				}
			}

			submittedEmail := email + tc.emailSuffix                   // Possibly make it the wrong email
			submittedOldPassword := oldPassword + tc.oldPasswordSuffix // Possibly make it the wrong password
			newPassword := oldPassword + auth.Password("_new")         // Possibly make the new password different (as it should be)
			newSeed := auth.ClientSaltSeed("edf98765edf98765edf98765edf98765edf98765edf98765edf98765edf98765")

			if err := s.ChangePasswordNoWallet(submittedEmail, submittedOldPassword, newPassword, newSeed); err != tc.expectedError {
				t.Errorf("ChangePasswordNoWallet: unexpected value for err. want: %+v, got: %+v", tc.expectedError, err)
			}

			// The password and wallet (if any) didn't change, the token didn't get
			// deleted. This tests the transaction rollbacks in particular, given the
			// errors that are at a couple different stages of the txn, triggered by
			// these tests.
			expectAccountMatch(t, &s, email.Normalize(), email, oldPassword, oldSeed, tc.verifyToken, tc.verifyExpiration)
			if tc.hasWallet {
				expectWalletExists(t, &s, userId, encryptedWallet, sequence, hmac)
			} else {
				expectWalletNotExists(t, &s, userId)
			}
			expectTokenExists(t, &s, authToken)
		})
	}
}
