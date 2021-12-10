package main // TODO - make it its own `store` package later

type StoreInterface interface {
	SaveToken(*AuthToken) error
}

type Store struct{}

func (s *Store) SaveToken(token *AuthToken) error {
	// params: pubKey PublicKey, DeviceID string?
	//   or is PubKey part of AuthToken struct?
	//   Anyway, (pubkey, deviceID) is primary key. we should have one token for each device.
	return nil
}

/* TODO:
authToken table contains:

...?

downloadKey table:

publicKey, email, KDF(downloadKey)

walletState table contains:

email, publicKey, walletState, sequence

(sequence is redundant since it's already in walletState but needed for transaction safety):

insert where publicKey=publicKey, sequence=sequence-1

if success, return success
if fail, select where publicKey=publicKey, return walletState

downloadKey:

select email=email (They won't have their public key if they need their downloadKey). check KDF(downloadKey)
*/
