package server

import (
	"fmt"
	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/store"
	"orblivion/lbry-id/wallet"
	"testing"
)

// Implementing interfaces for stubbed out packages

type TestAuth struct {
	TestToken    auth.AuthTokenString
	FailGenToken bool
}

func (a *TestAuth) NewToken(userId auth.UserId, deviceId auth.DeviceId, scope auth.AuthScope) (*auth.AuthToken, error) {
	if a.FailGenToken {
		return nil, fmt.Errorf("Test error: fail to generate token")
	}
	return &auth.AuthToken{Token: a.TestToken, UserId: userId, DeviceId: deviceId, Scope: scope}, nil
}

type TestStore struct {
	FailSave  bool
	FailLogin bool

	SaveTokenCalled bool
}

func (s *TestStore) SaveToken(token *auth.AuthToken) error {
	if s.FailSave {
		return fmt.Errorf("TestStore.SaveToken fail")
	}
	s.SaveTokenCalled = true
	return nil
}

func (s *TestStore) GetToken(auth.AuthTokenString) (*auth.AuthToken, error) {
	return nil, nil
}

func (s *TestStore) GetUserId(auth.Email, auth.Password) (auth.UserId, error) {
	if s.FailLogin {
		return 0, store.ErrNoUId
	}
	return 0, nil
}

func (s *TestStore) CreateAccount(auth.Email, auth.Password) error {
	return nil
}

func (s *TestStore) SetWalletState(
	UserId auth.UserId,
	walletStateJson string,
	sequence int,
	hmac wallet.WalletStateHmac,
) (latestWalletStateJson string, latestHmac wallet.WalletStateHmac, updated bool, err error) {
	return
}

func (s *TestStore) GetWalletState(UserId auth.UserId) (walletStateJson string, hmac wallet.WalletStateHmac, err error) {
	return
}

func TestServerHelperCheckAuthSuccess(t *testing.T) {
	t.Fatalf("Test me: checkAuth success")
}

func TestServerHelperCheckAuthErrors(t *testing.T) {
	t.Fatalf("Test me: checkAuth failure")
}

func TestServerHelperGetGetDataSuccess(t *testing.T) {
	t.Fatalf("Test me: getGetData success")
}
func TestServerHelperGetGetDataErrors(t *testing.T) {
	t.Fatalf("Test me: getGetData failure")
}

func TestServerHelperGetPostDataSuccess(t *testing.T) {
	t.Fatalf("Test me: getPostData success")
}
func TestServerHelperGetPostDataErrors(t *testing.T) {
	t.Fatalf("Test me: getPostData failure")
}

func TestServerHelperRequestOverheadSuccess(t *testing.T) {
	t.Fatalf("Test me: requestOverhead success")
}
func TestServerHelperRequestOverheadErrors(t *testing.T) {
	t.Fatalf("Test me: requestOverhead failures")
}
