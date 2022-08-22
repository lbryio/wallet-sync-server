package store

import (
	"errors"
	"testing"

	"github.com/mattn/go-sqlite3"

	"lbryio/wallet-sync-server/auth"
	"lbryio/wallet-sync-server/wallet"
)

func expectWalletExists(
	t *testing.T,
	s *Store,
	userId auth.UserId,
	expectedEncryptedWallet wallet.EncryptedWallet,
	expectedSequence wallet.Sequence,
	expectedHmac wallet.WalletHmac,
) {
	rows, err := s.db.Query(
		"SELECT encrypted_wallet, sequence, hmac FROM wallets WHERE user_id=?", userId)
	if err != nil {
		t.Fatalf("Error finding wallet for user_id=%d: %+v", userId, err)
	}
	defer rows.Close()

	var encryptedWallet wallet.EncryptedWallet
	var sequence wallet.Sequence
	var hmac wallet.WalletHmac

	for rows.Next() {

		err := rows.Scan(
			&encryptedWallet,
			&sequence,
			&hmac,
		)

		if err != nil {
			t.Fatalf("Error finding wallet for user_id=%d: %+v", userId, err)
		}

		if encryptedWallet != expectedEncryptedWallet || sequence != expectedSequence || hmac != expectedHmac || err != nil {
			t.Fatalf("Unexpected values for wallet: encrypted wallet: %+v sequence: %+v hmac: %+v err: %+v", encryptedWallet, sequence, hmac, err)
		}

		return // found a match, we're good
	}
	t.Fatalf("Expected wallet for user_id=%d: %+v", userId, err)
}

func expectWalletNotExists(t *testing.T, s *Store, userId auth.UserId) {
	rows, err := s.db.Query(
		"SELECT encrypted_wallet, sequence, hmac FROM wallets WHERE user_id=?", userId)
	if err != nil {
		t.Fatalf("Error finding (lack of) wallet for user_id=%d: %+v", userId, err)
	}
	defer rows.Close()

	var encryptedWallet wallet.EncryptedWallet
	var sequence wallet.Sequence
	var hmac wallet.WalletHmac

	for rows.Next() {

		err := rows.Scan(
			&encryptedWallet,
			&sequence,
			&hmac,
		)

		if err != nil {
			t.Fatalf("Error finding (lack of) wallet for user_id=%d: %+v", userId, err)
		}

		t.Fatalf("Expected no wallet. Got: encrypted wallet: %+v sequence: %+v hmac: %+v", encryptedWallet, sequence, hmac)
	}
	return // found nothing, we're good
}

// Test insertFirstWallet
// Try insertFirstWallet twice with the same user id, error the second time
func TestStoreInsertWallet(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// Get a valid userId
	userId, _, _, _ := makeTestUser(t, &s, nil, nil)

	// Get a wallet, come back empty
	expectWalletNotExists(t, &s, userId)

	// Put in a first wallet
	if err := s.insertFirstWallet(userId, wallet.EncryptedWallet("my-enc-wallet"), wallet.WalletHmac("my-hmac")); err != nil {
		t.Fatalf("Unexpected error in insertFirstWallet: %+v", err)
	}

	// Get a wallet, have the values we put in with a sequence of 1
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet"), wallet.Sequence(1), wallet.WalletHmac("my-hmac"))

	// Put in a first wallet for a second time, have an error for trying
	if err := s.insertFirstWallet(userId, wallet.EncryptedWallet("my-enc-wallet-2"), wallet.WalletHmac("my-hmac-2")); err != ErrDuplicateWallet {
		t.Fatalf(`insertFirstWallet err: wanted "%+v", got "%+v"`, ErrDuplicateToken, err)
	}

	// Get the same *first* wallet we successfully put in
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet"), wallet.Sequence(1), wallet.WalletHmac("my-hmac"))
}

// Test updateWalletToSequence, using insertFirstWallet as a helper
// Try updateWalletToSequence with no existing wallet, err for lack of anything to update
// Try updateWalletToSequence with a preexisting wallet but the wrong sequence, fail
// Try updateWalletToSequence with a preexisting wallet and the correct sequence, succeed
// Try updateWalletToSequence again, succeed
func TestStoreUpdateWallet(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// Get a valid userId
	userId, _, _, _ := makeTestUser(t, &s, nil, nil)

	// Try to update a wallet, fail for nothing to update
	if err := s.updateWalletToSequence(userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.Sequence(1), wallet.WalletHmac("my-hmac-a")); err != ErrNoWallet {
		t.Fatalf(`updateWalletToSequence err: wanted "%+v", got "%+v"`, ErrNoWallet, err)
	}

	// Get a wallet, come back empty since it failed
	expectWalletNotExists(t, &s, userId)

	// Put in a first wallet
	if err := s.insertFirstWallet(userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.WalletHmac("my-hmac-a")); err != nil {
		t.Fatalf("Unexpected error in insertFirstWallet: %+v", err)
	}

	// Try to update the wallet, fail for having the wrong sequence
	if err := s.updateWalletToSequence(userId, wallet.EncryptedWallet("my-enc-wallet-b"), wallet.Sequence(3), wallet.WalletHmac("my-hmac-b")); err != ErrNoWallet {
		t.Fatalf(`updateWalletToSequence err: wanted "%+v", got "%+v"`, ErrNoWallet, err)
	}

	// Get the same wallet we initially *inserted*, since it didn't update
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.Sequence(1), wallet.WalletHmac("my-hmac-a"))

	// Update the wallet successfully, with the right sequence
	if err := s.updateWalletToSequence(userId, wallet.EncryptedWallet("my-enc-wallet-b"), wallet.Sequence(2), wallet.WalletHmac("my-hmac-b")); err != nil {
		t.Fatalf("Unexpected error in updateWalletToSequence: %+v", err)
	}

	// Get a wallet, have the values we put in
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-b"), wallet.Sequence(2), wallet.WalletHmac("my-hmac-b"))

	// Update the wallet again successfully
	if err := s.updateWalletToSequence(userId, wallet.EncryptedWallet("my-enc-wallet-c"), wallet.Sequence(3), wallet.WalletHmac("my-hmac-c")); err != nil {
		t.Fatalf("Unexpected error in updateWalletToSequence: %+v", err)
	}

	// Get a wallet, have the values we put in
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-c"), wallet.Sequence(3), wallet.WalletHmac("my-hmac-c"))
}

// NOTE - the "behind the scenes" comments give a view of what we're expecting
// to happen, and why we're testing what we are. Sometimes it should insert,
// sometimes it should update. It depends on whether it's the first wallet
// submitted, and that's easily determined by sequence=1. However, if we switch
// to a database with "upserts" and take advantage of it, what happens behind
// the scenes will change a little, so the comments should be updated. Though,
// we'd probably best test the same cases.
//
// TODO when we have lastSynced again: test fail via update for having
// non-matching device sequence history. Though, maybe this goes into wallet
// util
func TestStoreSetWallet(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// Get a valid userId
	userId, _, _, _ := makeTestUser(t, &s, nil, nil)

	// Sequence 2 - fails - out of sequence (behind the scenes, tries to update but there's nothing there yet)
	if err := s.SetWallet(userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.Sequence(2), wallet.WalletHmac("my-hmac-a")); err != ErrWrongSequence {
		t.Fatalf(`SetWallet err: wanted "%+v", got "%+v"`, ErrWrongSequence, err)
	}
	expectWalletNotExists(t, &s, userId)

	// Sequence 1 - succeeds - out of sequence (behind the scenes, does an insert)
	if err := s.SetWallet(userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.Sequence(1), wallet.WalletHmac("my-hmac-a")); err != nil {
		t.Fatalf("Unexpected error in SetWallet: %+v", err)
	}
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.Sequence(1), wallet.WalletHmac("my-hmac-a"))

	// Sequence 1 - fails - out of sequence (behind the scenes, tries to insert but there's something there already)
	if err := s.SetWallet(userId, wallet.EncryptedWallet("my-enc-wallet-b"), wallet.Sequence(1), wallet.WalletHmac("my-hmac-b")); err != ErrWrongSequence {
		t.Fatalf(`SetWallet err: wanted "%+v", got "%+v"`, ErrWrongSequence, err)
	}
	// Expect the *first* wallet to still be there
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.Sequence(1), wallet.WalletHmac("my-hmac-a"))

	// Sequence 3 - fails - out of sequence (behind the scenes: tries via update, which is appropriate here)
	if err := s.SetWallet(userId, wallet.EncryptedWallet("my-enc-wallet-b"), wallet.Sequence(3), wallet.WalletHmac("my-hmac-b")); err != ErrWrongSequence {
		t.Fatalf(`SetWallet err: wanted "%+v", got "%+v"`, ErrWrongSequence, err)
	}
	// Expect the *first* wallet to still be there
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.Sequence(1), wallet.WalletHmac("my-hmac-a"))

	// Sequence 2 - succeeds - (behind the scenes, does an update. Tests successful update-after-insert)
	if err := s.SetWallet(userId, wallet.EncryptedWallet("my-enc-wallet-b"), wallet.Sequence(2), wallet.WalletHmac("my-hmac-b")); err != nil {
		t.Fatalf("Unexpected error in SetWallet: %+v", err)
	}
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-b"), wallet.Sequence(2), wallet.WalletHmac("my-hmac-b"))

	// Sequence 3 - succeeds - (behind the scenes, does an update. Tests successful update-after-update. Maybe gratuitous?)
	if err := s.SetWallet(userId, wallet.EncryptedWallet("my-enc-wallet-c"), wallet.Sequence(3), wallet.WalletHmac("my-hmac-c")); err != nil {
		t.Fatalf("Unexpected error in SetWallet: %+v", err)
	}
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-c"), wallet.Sequence(3), wallet.WalletHmac("my-hmac-c"))
}

// Pretty simple, only two cases: wallet is there or it's not.
func TestStoreGetWallet(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// Get a valid userId
	userId, _, _, _ := makeTestUser(t, &s, nil, nil)

	// GetWallet fails when there's no wallet
	encryptedWallet, sequence, hmac, err := s.GetWallet(userId)
	if len(encryptedWallet) != 0 || sequence != 0 || len(hmac) != 0 || err != ErrNoWallet {
		t.Fatalf("Expected ErrNoWallet, and no wallet values. Instead got: encrypted wallet: %+v sequence: %+v hmac: %+v err: %+v", encryptedWallet, sequence, hmac, err)
	}

	if err := s.SetWallet(userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.Sequence(1), wallet.WalletHmac("my-hmac-a")); err != nil {
		t.Fatalf("Unexpected error in SetWallet: %+v", err)
	}

	// GetWallet succeeds when there's a wallet
	encryptedWallet, sequence, hmac, err = s.GetWallet(userId)
	if encryptedWallet != wallet.EncryptedWallet("my-enc-wallet-a") || sequence != wallet.Sequence(1) || hmac != wallet.WalletHmac("my-hmac-a") || err != nil {
		t.Fatalf("Unexpected values for wallet: encrypted wallet: %+v sequence: %+v hmac: %+v err: %+v", encryptedWallet, sequence, hmac, err)
	}
}

func TestStoreWalletEmptyFields(t *testing.T) {
	// Make sure expiration doesn't get set if sanitization fails
	tt := []struct {
		name            string
		encryptedWallet wallet.EncryptedWallet
		hmac            wallet.WalletHmac
	}{
		{
			name:            "missing encrypted wallet",
			encryptedWallet: wallet.EncryptedWallet(""),
			hmac:            wallet.WalletHmac("my-hmac"),
		}, {
			name:            "missing hmac",
			encryptedWallet: wallet.EncryptedWallet("my-enc-wallet"),
			hmac:            wallet.WalletHmac(""),
		},
		// Not testing 0 sequence because the method basically doesn't allow for it.
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s, sqliteTmpFile := StoreTestInit(t)
			defer StoreTestCleanup(sqliteTmpFile)

			userId, _, _, _ := makeTestUser(t, &s, nil, nil)

			var sqliteErr sqlite3.Error

			err := s.insertFirstWallet(userId, tc.encryptedWallet, tc.hmac)
			if errors.As(err, &sqliteErr) {
				if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintCheck) {
					return // We got the error we expected
				}
			}
			t.Errorf("Expected check constraint error for empty field. Got %+v", err)
		})
	}
}
