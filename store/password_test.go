package store

import (
	"testing"
	"time"

	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/wallet"
)

func TestStoreChangePasswordSuccess(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	userId, email, oldPassword := makeTestUser(t, &s)
	token := auth.TokenString("my-token")

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
	encryptedWallet := wallet.EncryptedWallet("my-enc-wallet-2")
	sequence := wallet.Sequence(2)
	hmac := wallet.WalletHmac("my-hmac-2")

	if err := s.ChangePassword(email, oldPassword, newPassword, encryptedWallet, sequence, hmac); err != nil {
		t.Errorf("ChangePassword: unexpected error: %+v", err)
	}

	expectAccountMatch(t, &s, email, newPassword)
	expectWalletExists(t, &s, userId, encryptedWallet, sequence, hmac)
	expectTokenNotExists(t, &s, token)
}

func TestStoreChangePasswordErrors(t *testing.T) {
	tt := []struct {
		name              string
		hasWallet         bool
		sequence          wallet.Sequence
		emailSuffix       auth.Email
		oldPasswordSuffix auth.Password
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

			// Maybe ErrNoWallet once we sort it out. See TODO item by its definition.
			expectedError: ErrWrongSequence,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s, sqliteTmpFile := StoreTestInit(t)
			defer StoreTestCleanup(sqliteTmpFile)

			userId, email, oldPassword := makeTestUser(t, &s)
			expiration := time.Now().UTC().Add(time.Hour * 24 * 14)
			authToken := auth.AuthToken{
				Token:      auth.TokenString("my-token"),
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
			newPassword := oldPassword + auth.Password("_new")         // Possibly make the new password different (as it should be)

			if err := s.ChangePassword(submittedEmail, submittedOldPassword, newPassword, newEncryptedWallet, tc.sequence, newHmac); err != tc.expectedError {
				t.Errorf("ChangePassword: unexpected value for err. want: %+v, got: %+v", tc.expectedError, err)
			}

			// The password and wallet didn't change, the token didn't get deleted.
			// This tests the transaction rollbacks in particular, given the errors
			// that are at a couple different stages of the txn, triggered by these
			// tests.
			expectAccountMatch(t, &s, email, oldPassword)
			if tc.hasWallet {
				expectWalletExists(t, &s, userId, oldEncryptedWallet, oldSequence, oldHmac)
			} else {
				expectWalletNotExists(t, &s, userId)
			}
			expectTokenExists(t, &s, authToken)
		})
	}
}
