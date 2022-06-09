package server

import (
	"testing"
)

func TestServerGetWalletSuccess(t *testing.T) {
	t.Fatalf("Test me: GetWallet succeeds")
}

func TestServerGetWalletErrors(t *testing.T) {
	t.Fatalf("Test me: GetWallet fails for various reasons (malformed, auth, db fail)")
}

func TestServerGetWalletParams(t *testing.T) {
	t.Fatalf("Test me: getWalletParams")
}

func TestServerPostWalletSuccess(t *testing.T) {
	t.Fatalf("Test me: PostWallet succeeds and returns the new wallet, PostWallet succeeds but is preempted")
}

func TestServerPostWalletTooLate(t *testing.T) {
	t.Fatalf("Test me: PostWallet fails for sequence being too low, returns the latest wallet")
}

func TestServerPostWalletErrors(t *testing.T) {
	// (malformed json, db fail, auth token not found, wallet metadata invalid (via stub, make sure the validation function is even called), sequence too high, device id doesn't match token device id)
	// Client sends sequence != 1 for first entry
	// Client sends sequence == x + 10 for xth entry or whatever
	t.Fatalf("Test me: PostWallet fails for various reasons")
}

func TestServerValidateWalletRequest(t *testing.T) {
	// also add a basic test case for this in TestServerAuthHandlerSuccess to make sure it's called at all
	t.Fatalf("Test me: Implement and test WalletRequest.validate()")
}

func TestServerHandleWallet(t *testing.T) {
	t.Fatalf("Test me: Call the get or post function as appropriate. Alternately: call handleWallet for the existing tests.")
}
