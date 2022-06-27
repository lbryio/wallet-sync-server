package store

import (
	"testing"

	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/wallet"
)

func expectWalletExists(
	t *testing.T,
	s *Store,
	userId auth.UserId,
	expectedEncryptedWallet wallet.EncryptedWallet,
	expectedSequence wallet.Sequence,
	expectedHmac wallet.WalletHmac,
) {
	encryptedWallet, sequence, hmac, err := s.GetWallet(userId)
	if encryptedWallet != expectedEncryptedWallet || sequence != expectedSequence || hmac != expectedHmac || err != nil {
		t.Fatalf("Unexpected values for wallet: encrypted wallet: %+v sequence: %+v hmac: %+v err: %+v", encryptedWallet, sequence, hmac, err)
	}
}

func expectWalletNotExists(t *testing.T, s *Store, userId auth.UserId) {
	encryptedWallet, sequence, hmac, err := s.GetWallet(userId)
	if len(encryptedWallet) != 0 || sequence != 0 || len(hmac) != 0 || err != ErrNoWallet {
		t.Fatalf("Expected ErrNoWallet, and no wallet values. Instead got: encrypted wallet: %+v sequence: %+v hmac: %+v err: %+v", encryptedWallet, sequence, hmac, err)
	}
}

func setupWalletTest(s *Store) auth.UserId {
	email, password := auth.Email("abc@example.com"), auth.Password("123")
	_ = s.CreateAccount(email, password)
	userId, _ := s.GetUserId(email, password)
	return userId
}

// Test insertFirstWallet, using GetWallet, CreateAccount and GetUserID as a helpers
// Try insertFirstWallet twice with the same user id, error the second time
func TestStoreInsertWallet(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// Get a valid userId
	userId := setupWalletTest(&s)

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

// Test updateWalletToSequence, using GetWallet, CreateAccount, GetUserID, and insertFirstWallet as helpers
// Try updateWalletToSequence with no existing wallet, err for lack of anything to update
// Try updateWalletToSequence with a preexisting wallet but the wrong sequence, fail
// Try updateWalletToSequence with a preexisting wallet and the correct sequence, succeed
// Try updateWalletToSequence again, succeed
func TestStoreUpdateWallet(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// Get a valid userId
	userId := setupWalletTest(&s)

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
	userId := setupWalletTest(&s)

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
	userId := setupWalletTest(&s)

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


