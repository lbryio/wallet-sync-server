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
	TestToken    auth.TokenString
	FailGenToken bool
}

func (a *TestAuth) NewToken(userId auth.UserId, deviceId auth.DeviceId, scope auth.AuthScope) (*auth.AuthToken, error) {
	if a.FailGenToken {
		return nil, fmt.Errorf("Test error: fail to generate token")
	}
	return &auth.AuthToken{Token: a.TestToken, UserId: userId, DeviceId: deviceId, Scope: scope}, nil
}

type TestStoreFunctions struct {
	SaveToken     bool
	GetToken      bool
	GetUserId     bool
	CreateAccount bool
	SetWallet     bool
	GetWallet     bool
}

type TestStore struct {
	// Fake store functions will set these to `true` as they are called
	Called TestStoreFunctions

	// Fake store functions will fail if these are set to `true` by the test
	// setup
	Failures TestStoreFunctions
}

func (s *TestStore) SaveToken(token *auth.AuthToken) error {
	if s.Failures.SaveToken {
		return fmt.Errorf("TestStore.SaveToken fail")
	}
	s.Called.SaveToken = true
	return nil
}

func (s *TestStore) GetToken(auth.TokenString) (*auth.AuthToken, error) {
	if s.Failures.GetToken {
		return nil, fmt.Errorf("TestStore.GetToken fail")
	}
	s.Called.GetToken = true
	return nil, nil
}

func (s *TestStore) GetUserId(auth.Email, auth.Password) (auth.UserId, error) {
	if s.Failures.GetUserId {
		return 0, store.ErrNoUId
	}
	s.Called.GetUserId = true
	return 0, nil
}

func (s *TestStore) CreateAccount(auth.Email, auth.Password) error {
	if s.Failures.CreateAccount {
		return fmt.Errorf("TestStore.CreateAccount fail")
	}
	s.Called.CreateAccount = true
	return nil
}

func (s *TestStore) SetWallet(
	UserId auth.UserId,
	encryptedWallet wallet.EncryptedWallet,
	sequence wallet.Sequence,
	hmac wallet.WalletHmac,
) (latestEncryptedWallet wallet.EncryptedWallet, latestSequence wallet.Sequence, latestHmac wallet.WalletHmac, sequenceCorrect bool, err error) {
	if s.Failures.SetWallet {
		err = fmt.Errorf("TestStore.SetWallet fail")
		return
	}
	s.Called.SetWallet = true
	return
}

func (s *TestStore) GetWallet(userId auth.UserId) (encryptedWallet wallet.EncryptedWallet, sequence wallet.Sequence, hmac wallet.WalletHmac, err error) {
	if s.Failures.GetWallet {
		err = fmt.Errorf("TestStore.GetWallet fail")
		return
	}
	s.Called.GetWallet = true
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
