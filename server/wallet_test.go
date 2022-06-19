package server

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/wallet"
)

func TestServerGetWalletSuccess(t *testing.T) {
	testAuth := TestAuth{
		TestToken: auth.TokenString("seekrit"),
	}
	testStore := TestStore{
		TestAuthToken: auth.AuthToken{
			Token: auth.TokenString("seekrit"),
			Scope: auth.ScopeFull,
		},

		TestEncryptedWallet: wallet.EncryptedWallet("my-encrypted-wallet"),
		TestSequence:        wallet.Sequence(2),
		TestHmac:            wallet.WalletHmac("my-hmac"),
	}

	s := Server{&testAuth, &testStore}

	req := httptest.NewRequest(http.MethodGet, PathWallet+"/?token=seekrit", bytes.NewBuffer([]byte{}))
	w := httptest.NewRecorder()

	// test handleWallet while we're at it, which is a dispatch for get and post
	// wallet
	s.handleWallet(w, req)
	body, _ := ioutil.ReadAll(w.Body)

	if want, got := http.StatusOK, w.Result().StatusCode; want != got {
		t.Errorf("StatusCode: expected %s (%d), got %s (%d)", http.StatusText(want), want, http.StatusText(got), got)
	}

	var result WalletResponse
	err := json.Unmarshal(body, &result)

	if err != nil || result.EncryptedWallet != testStore.TestEncryptedWallet || result.Hmac != testStore.TestHmac || result.Sequence != testStore.TestSequence {
		t.Errorf("Expected wallet response to have the test wallet values: result: %+v err: %+v", string(body), err)
	}

	if !testStore.Called.GetWallet {
		t.Errorf("Expected Store.GetWallet to be called")
	}
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
