package wallet

import (
	"testing"
)

// Test stubs for now

func TestWalletSequence(t *testing.T) {
	t.Fatalf("Test me: test that walletState.Sequence() == walletState.lastSynced[wallet.DeviceId]")
}

func TestWalletValidateWalletState(t *testing.T) {
	// walletState.DeviceId in walletState.lastSynced
	// Sequence for lastSynced all > 1
	t.Fatalf("Test me: Implement and test validateWalletState.")
}

// TODO - other wallet integrity stuff? particularly related to sequence?
