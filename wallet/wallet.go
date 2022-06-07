package wallet

import "orblivion/lbry-id/auth"

// Currently a small package but given other packages it makes imports easier.
// Also this might grow substantially over time

// For test stubs
type WalletUtilInterface interface {
	ValidateWalletStateMetadata(walletState *WalletStateMetadata) bool
}

type WalletUtil struct{}

// This is a subset of the WalletState structure, only the metadata fields. We
// don't need access to the encrypted wallet.
type WalletStateMetadata struct {
	DeviceId   auth.DeviceId         `json:"deviceId"`
	LastSynced map[auth.DeviceId]int `json:"lastSynced"`
}

type WalletStateHmac string

// TODO - These "validate" functions could/should be methods. Though I think
// we'd lose mockability for testing, since the method isn't the
// WalletUtilInterface.
// Mainly the job of the clients but we may as well short-circuit problems
// here before saving them.
func (wu *WalletUtil) ValidateWalletStateMetadata(walletState *WalletStateMetadata) bool {

	// TODO - nonempty fields, up to date, etc
	return true
}

// Assumptions: `ws` has been validated
// Avoid having to check for error
func (ws *WalletStateMetadata) Sequence() int {
	return ws.LastSynced[ws.DeviceId]
}
