package store

// TODO - DeviceId - What about clients that lie about deviceId? Maybe require a certain format to make sure it gives a real value? Something it wouldn't come up with by accident.

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/mattn/go-sqlite3"

	"lbryio/wallet-sync-server/auth"
	"lbryio/wallet-sync-server/wallet"
)

var (
	ErrDuplicateToken       = fmt.Errorf("Token already exists for this user and device")
	ErrNoTokenForUserDevice = fmt.Errorf("Token does not exist for this user and device")
	ErrNoTokenForUser       = fmt.Errorf("Token does not exist for this user")

	ErrDuplicateWallet = fmt.Errorf("Wallet already exists for this user")

	ErrNoWallet = fmt.Errorf("Wallet does not exist for this user")

	ErrUnexpectedWallet = fmt.Errorf("Wallet unexpectedly exist for this user")
	ErrWrongSequence    = fmt.Errorf("Wallet could not be updated to this sequence")

	ErrDuplicateEmail   = fmt.Errorf("Email already exists for this user")
	ErrDuplicateAccount = fmt.Errorf("User already has an account")

	ErrWrongCredentials = fmt.Errorf("No match for email and/or password")
	ErrNotVerified      = fmt.Errorf("User account is not verified")
)

const (
	AuthTokenLifespan   = time.Hour * 24 * 14
	VerifyTokenLifespan = time.Hour * 24 * 2

	// Eventually it could become variable when we introduce server switching. A user
	// might be on a later sequence when they switch from another server.
	InitialWalletSequence = 1
)

// For test stubs
type StoreInterface interface {
	SaveToken(*auth.AuthToken) error
	GetToken(auth.AuthTokenString) (*auth.AuthToken, error)
	SetWallet(auth.UserId, wallet.EncryptedWallet, wallet.Sequence, wallet.WalletHmac) error
	GetWallet(auth.UserId) (wallet.EncryptedWallet, wallet.Sequence, wallet.WalletHmac, error)
	GetUserId(auth.Email, auth.Password) (auth.UserId, error)
	CreateAccount(auth.Email, auth.Password, auth.ClientSaltSeed, *auth.VerifyTokenString) error
	UpdateVerifyTokenString(auth.Email, auth.VerifyTokenString) error
	VerifyAccount(auth.VerifyTokenString) error
	ChangePasswordWithWallet(auth.Email, auth.Password, auth.Password, auth.ClientSaltSeed, wallet.EncryptedWallet, wallet.Sequence, wallet.WalletHmac) (auth.UserId, error)
	ChangePasswordNoWallet(auth.Email, auth.Password, auth.Password, auth.ClientSaltSeed) (auth.UserId, error)
	GetClientSaltSeed(auth.Email) (auth.ClientSaltSeed, error)
}

type Store struct {
	db *sql.DB
}

func (s *Store) Init(fileName string) {
	db, err := sql.Open("sqlite3", "file:"+fileName+"?_foreign_keys=on")
	if err != nil {
		log.Fatal(err)
	}
	s.db = db
}

func (s *Store) Migrate() error {
	// We use the `sequence` field for transaction safety. For instance, let's
	// say two different clients are trying to update the sequence from 5 to 6.
	// The update command will specify "WHERE sequence=5". Only one of these
	// commands will succeed, and the other will get back an error.

	// We use AUTOINCREMENT against the protestations of people on the Internet
	// who claim that INTEGER PRIMARY KEY automatically has autoincrment, and
	// that using it when it's not "strictly needed" uses extra resources. But
	// without AUTOINCREMENT, it might reuse primary keys if a row is deleted and
	// re-added. Who wants that risk? Besides, we'll switch to Postgres when it's
	// time to scale anyway.

	// We use UNIQUE on auth_tokens.token so that we can retrieve it easily and
	// identify the user (and I suppose the uniqueness provides a little extra
	// security in case we screw up the random generator). However the primary
	// key should still be (user_id, device_id) so that a device's row can be
	// updated with a new token.
	query := `
		CREATE TABLE IF NOT EXISTS auth_tokens(
			token TEXT NOT NULL UNIQUE,
			user_id INTEGER NOT NULL,
			device_id TEXT NOT NULL,
			scope TEXT NOT NULL,
			expiration DATETIME NOT NULL,
			CHECK (
			  -- should eventually fail for foreign key constraint instead
			  device_id <> '' AND

			  token <> '' AND
			  scope <> '' AND

			  -- Don't know when it uses either format to denote UTC
			  expiration <> "0001-01-01 00:00:00+00:00" AND
			  expiration <> "0001-01-01 00:00:00Z"

			),
			PRIMARY KEY (user_id, device_id)
			FOREIGN KEY (user_id) REFERENCES accounts(user_id)
		);
		CREATE TABLE IF NOT EXISTS wallets(
			user_id INTEGER NOT NULL,
			encrypted_wallet TEXT NOT NULL,
			sequence INTEGER NOT NULL,
			hmac TEXT NOT NULL,
			updated DATETIME NOT NULL,

			PRIMARY KEY (user_id)
			FOREIGN KEY (user_id) REFERENCES accounts(user_id)
			CHECK (
			  encrypted_wallet <> '' AND
			  hmac <> '' AND
			  sequence <> 0
			)
		);
		CREATE TABLE IF NOT EXISTS accounts(
			normalized_email TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL,
			key TEXT NOT NULL,
			client_salt_seed TEXT NOT NULL,
			server_salt TEXT NOT NULL,

			-- UNIQUE because we will query by token when verifying
			--
			-- Nullable because we want to use null to represent verified users. We can't use empty string
			-- because multiple accounts with empty string will trigger the unique constraint, unlike null.
			verify_token TEXT UNIQUE,

			verify_expiration DATETIME,
			user_id INTEGER PRIMARY KEY AUTOINCREMENT,
			created DATETIME DEFAULT (DATETIME('now')),
			updated DATETIME NOT NULL,
			CHECK (
			  email <> '' AND
			  normalized_email <> '' AND
			  key <> '' AND
			  client_salt_seed <> '' AND
			  server_salt <> ''
			)
		);
	`

	_, err := s.db.Exec(query)
	return err
}

////////////////
// Auth Token //
////////////////

// TODO - Is it safe to assume that the owner of the token is legit, and is
// coming from the legit device id? No need to query by userId and deviceId
// (which I did previously)?
//
// TODO Put the timestamp in the token to avoid duplicates over time. And/or just use a library! Someone solved this already.
// Assumption: User is verified (as it was necessary to call SaveToken to begin
// with)
func (s *Store) GetToken(token auth.AuthTokenString) (authToken *auth.AuthToken, err error) {
	expirationCutoff := time.Now().UTC()

	authToken = &(auth.AuthToken{})

	err = s.db.QueryRow(
		"SELECT token, user_id, device_id, scope, expiration FROM auth_tokens WHERE token=? AND expiration>?", token, expirationCutoff,
	).Scan(
		&authToken.Token,
		&authToken.UserId,
		&authToken.DeviceId,
		&authToken.Scope,
		&authToken.Expiration,
	)
	if err == sql.ErrNoRows {
		err = ErrNoTokenForUserDevice
	}
	if err != nil {
		authToken = nil
	}
	return
}

func (s *Store) insertToken(authToken *auth.AuthToken, expiration time.Time) (err error) {
	_, err = s.db.Exec(
		"INSERT INTO auth_tokens (token, user_id, device_id, scope, expiration) VALUES(?,?,?,?,?)",
		authToken.Token, authToken.UserId, authToken.DeviceId, authToken.Scope, expiration,
	)

	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		// I initially expected to need to check for ErrConstraintUnique.
		// Maybe for psql it will be?
		if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintPrimaryKey) {
			err = ErrDuplicateToken
		}
	}

	return
}

func (s *Store) updateToken(authToken *auth.AuthToken, experation time.Time) (err error) {
	res, err := s.db.Exec(
		"UPDATE auth_tokens SET token=?, expiration=?, scope=? WHERE user_id=? AND device_id=?",
		authToken.Token, experation, authToken.Scope, authToken.UserId, authToken.DeviceId,
	)
	if err != nil {
		return
	}

	numRows, err := res.RowsAffected()
	if err != nil {
		return
	}
	if numRows == 0 {
		err = ErrNoTokenForUserDevice
	}
	return
}

// Assumption: User is verified (as they have been identified with GetUserId
// which requires users be verified)
func (s *Store) SaveToken(token *auth.AuthToken) (err error) {
	// TODO: For psql, do upsert here instead of separate insertToken and updateToken functions
	//       Actually it may even be available for SQLite?
	//       But not for wallet, it probably makes sense to keep that separate because of the sequence variable

	// TODO - Should we auto-delete expired tokens?

	expiration := time.Now().UTC().Add(AuthTokenLifespan)

	// This is most likely not the first time calling this function for this
	// device, so there's probably already a token in there.
	err = s.updateToken(token, expiration)

	if err == ErrNoTokenForUserDevice {
		// If we don't have a token already saved, insert a new one:
		err = s.insertToken(token, expiration)

		if err == ErrDuplicateToken {
			// By unlikely coincidence, a token was created between trying `updateToken`
			// and trying `insertToken`. At this point we can safely `updateToken`.
			// TODO - reconsider this - if one client has two concurrent requests
			// that create this situation, maybe the second one should just fail?
			err = s.updateToken(token, expiration)
		}
	}
	if err == nil {
		token.Expiration = &expiration
	}
	return
}

////////////
// Wallet //
////////////

// Assumption: Auth token has been checked (thus account is verified)
func (s *Store) GetWallet(userId auth.UserId) (encryptedWallet wallet.EncryptedWallet, sequence wallet.Sequence, hmac wallet.WalletHmac, err error) {
	err = s.db.QueryRow(
		"SELECT encrypted_wallet, sequence, hmac FROM wallets WHERE user_id=?",
		userId,
	).Scan(
		&encryptedWallet,
		&sequence,
		&hmac,
	)
	if err == sql.ErrNoRows {
		err = ErrNoWallet
	}
	return
}

func (s *Store) insertFirstWallet(
	userId auth.UserId,
	encryptedWallet wallet.EncryptedWallet,
	hmac wallet.WalletHmac,
) (err error) {
	// This will only be used to attempt to insert the first wallet (sequence=InitialWalletSequence).
	//   The database will enforce that this will not be set if this user already
	//   has a wallet.
	_, err = s.db.Exec(
		"INSERT INTO wallets (user_id, encrypted_wallet, sequence, hmac, updated) VALUES(?,?,?,?, datetime('now'))",
		userId, encryptedWallet, InitialWalletSequence, hmac,
	)

	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		// I initially expected to need to check for ErrConstraintUnique.
		// Maybe for psql it will be?
		if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintPrimaryKey) {
			// NOTE While ErrDuplicateWallet makes sense in the context of trying to insert,
			// SetWallet, which also handles update, translates this to ErrWrongSequence
			err = ErrDuplicateWallet
		}
	}

	return
}

func (s *Store) updateWalletToSequence(
	userId auth.UserId,
	encryptedWallet wallet.EncryptedWallet,
	sequence wallet.Sequence,
	hmac wallet.WalletHmac,
) (err error) {
	// This will be used for wallets with sequence > InitialWalletSequence.
	// Use the database to enforce that we only update if we are incrementing the sequence.
	// This way, if two clients attempt to update at the same time, it will return
	// an error for the second one.
	res, err := s.db.Exec(
		"UPDATE wallets SET encrypted_wallet=?, sequence=?, hmac=?, updated=datetime('now') WHERE user_id=? AND sequence=?",
		encryptedWallet, sequence, hmac, userId, sequence-1,
	)
	if err != nil {
		return
	}

	numRows, err := res.RowsAffected()
	if err != nil {
		return
	}
	if numRows == 0 {
		// NOTE While ErrNoWallet makes sense in the context of trying to update,
		// SetWallet, which also handles insert, translates this to ErrWrongSequence
		err = ErrNoWallet
	}
	return
}

// Assumption: Sequence has been validated (>=InitialWalletSequence)
// Assumption: Auth token has been checked (thus account is verified)
func (s *Store) SetWallet(userId auth.UserId, encryptedWallet wallet.EncryptedWallet, sequence wallet.Sequence, hmac wallet.WalletHmac) (err error) {
	if sequence == InitialWalletSequence {
		// If sequence == InitialWalletSequence, the client assumed that this is our first
		// wallet. Try to insert. If we get a conflict, the client
		// assumed incorrectly and we proceed below to return the latest
		// wallet from the db.
		err = s.insertFirstWallet(userId, encryptedWallet, hmac)
		if err == ErrDuplicateWallet {
			// A wallet already exists. That means the input sequence should not be InitialWalletSequence.
			// To the caller, this means the sequence was wrong.
			err = ErrWrongSequence
		}
	} else {
		// If sequence > InitialWalletSequence, the client assumed that it is replacing wallet
		// with sequence - 1. Explicitly try to update the wallet with
		// sequence - 1. If we updated no rows, the client assumed incorrectly
		// and we proceed below to return the latest wallet from the db.
		err = s.updateWalletToSequence(userId, encryptedWallet, sequence, hmac)
		if err == ErrNoWallet {
			// No wallet found to replace at the `sequence - 1`. To the caller, this
			// means the sequence they put in was wrong.
			err = ErrWrongSequence
		}
	}
	return
}

func (s *Store) GetUserId(email auth.Email, password auth.Password) (userId auth.UserId, err error) {
	var key auth.KDFKey
	var salt auth.ServerSalt
	var verified bool

	err = s.db.QueryRow(
		`SELECT user_id, key, server_salt, verify_token is null from accounts WHERE normalized_email=?`,
		email.Normalize(),
	).Scan(&userId, &key, &salt, &verified)
	if err == sql.ErrNoRows {
		err = ErrWrongCredentials
	}
	if err != nil {
		return
	}
	match, err := password.Check(key, salt)
	if err == nil && !match {
		err = ErrWrongCredentials
		userId = auth.UserId(0)
	}
	if err == nil && !verified {
		err = ErrNotVerified
		userId = auth.UserId(0)
	}
	return
}

/////////////
// Account //
/////////////

func (s *Store) CreateAccount(email auth.Email, password auth.Password, seed auth.ClientSaltSeed, verifyToken *auth.VerifyTokenString) (err error) {
	key, salt, err := password.Create()
	if err != nil {
		return
	}

	var verifyExpiration *time.Time
	if verifyToken != nil {
		verifyExpiration = new(time.Time)
		*verifyExpiration = time.Now().UTC().Add(VerifyTokenLifespan)
	}

	// userId auto-increments
	_, err = s.db.Exec(
		"INSERT INTO accounts (normalized_email, email, key, server_salt, client_salt_seed, verify_token, verify_expiration, updated) VALUES(?,?,?,?,?,?,?, datetime('now'))",
		email.Normalize(), email, key, salt, seed, verifyToken, verifyExpiration,
	)
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
			err = ErrDuplicateAccount
		}
	}

	return
}

// In case the user needs a new verification email, generate a new verify token
// with a new deadline 2 days away.
//
// This function should only work if the account is not already verified.
// Otherwise we risk de-verifying accounts which would be confusing and
// annoying if it were to ever get triggered.
func (s *Store) UpdateVerifyTokenString(email auth.Email, verifyTokenString auth.VerifyTokenString) (err error) {
	expiration := time.Now().UTC().Add(VerifyTokenLifespan)

	res, err := s.db.Exec(
		`UPDATE accounts SET verify_token=?, verify_expiration=?, updated=datetime('now') WHERE normalized_email=? and verify_token is not null`,
		verifyTokenString, expiration, email.Normalize(),
	)
	if err != nil {
		return
	}

	numRows, err := res.RowsAffected()
	if err != nil {
		return
	}
	if numRows == 0 {
		// Since we got a miss (presumably not very common), let's do another check
		// to see which error to return: invalid email or invalid token
		var dummy int
		err = s.db.QueryRow(
			`SELECT 1 from accounts WHERE normalized_email=?`,
			email.Normalize(),
		).Scan(&dummy)
		if err == sql.ErrNoRows {
			err = ErrWrongCredentials
		}
		if err == nil {
			err = ErrNoTokenForUser
		}
	}
	return
}

func (s *Store) VerifyAccount(verifyTokenString auth.VerifyTokenString) (err error) {
	expirationCutoff := time.Now().UTC()

	res, err := s.db.Exec(
		"UPDATE accounts SET verify_token=null, verify_expiration=null, updated=datetime('now') WHERE verify_token=? AND verify_expiration>?",
		verifyTokenString, expirationCutoff,
	)
	if err != nil {
		return
	}

	numRows, err := res.RowsAffected()
	if err != nil {
		return
	}
	if numRows == 0 {
		err = ErrNoTokenForUser
	}
	return
}

// Change password. For the user, this requires changing their root password,
// which changes the encryption key for the wallet as well. Thus, we should
// update the wallet at the same time to avoid ever having a situation where
// these two don't match.
//
// Also delete all auth tokens to force clients to update their root password
// to get a new token. This prevents other clients from posting a wallet
// encrypted with the old key.
//
// Return userId as a pure convenience for the calling request handler.
//
// TODO - A wallet encrypted with the old key could still save successfully in
//   a race condition:
// 1) get auth token request passes old password check
// 2) password change transaction begins and ends
// 3) get auth token request saves and returns a new token
// 4) post wallet using the auth token that snuck by
// One obvious solution would be to integrate everything into one database
//   transaction. This problem could apply to other requests as well. Not just
//   database ones: there's a similar potential race condition trying to boot
//   users from all of their websockets on password change. We should think
//   about it. Maybe we could have a counter for password changes, similar to
//   Sequence? And the tokens have that number attached to it. We can check it
//   as an extra validation of the token.
func (s *Store) ChangePasswordWithWallet(
	email auth.Email,
	oldPassword auth.Password,
	newPassword auth.Password,
	clientSaltSeed auth.ClientSaltSeed,
	encryptedWallet wallet.EncryptedWallet,
	sequence wallet.Sequence,
	hmac wallet.WalletHmac,
) (userId auth.UserId, err error) {
	return s.changePassword(
		email,
		oldPassword,
		newPassword,
		clientSaltSeed,
		encryptedWallet,
		sequence,
		hmac,
	)
}

// Change password, but with no wallet currently saved. Since there's no
// wallet saved, there's no wallet to update. The encryption key is moot.
//
// Also delete all auth tokens to force clients to update their root password
// to get a new token. This prevents other clients from posting a wallet
// encrypted with the old key.
//
// Return userId as a pure convenience for the calling request handler.
func (s *Store) ChangePasswordNoWallet(
	email auth.Email,
	oldPassword auth.Password,
	newPassword auth.Password,
	clientSaltSeed auth.ClientSaltSeed,
) (userId auth.UserId, err error) {
	return s.changePassword(
		email,
		oldPassword,
		newPassword,
		clientSaltSeed,
		wallet.EncryptedWallet(""),
		wallet.Sequence(0),
		wallet.WalletHmac(""),
	)
}

// Common code for for WithWallet and WithNoWallet password change functions
func (s *Store) changePassword(
	email auth.Email,
	oldPassword auth.Password,
	newPassword auth.Password,
	clientSaltSeed auth.ClientSaltSeed,
	encryptedWallet wallet.EncryptedWallet,
	sequence wallet.Sequence,
	hmac wallet.WalletHmac,
) (userId auth.UserId, err error) {

	tx, err := s.db.Begin()
	if err != nil {
		return
	}

	// Lots of error conditions. Just defer this. However, we need to make sure to
	// make sure the variable `err` is set to the error before we return, instead
	// of doing `return <error>`.
	endTxn := func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}
	defer endTxn()

	var oldKey auth.KDFKey
	var oldSalt auth.ServerSalt
	var verified bool

	err = tx.QueryRow(
		`SELECT user_id, key, server_salt, verify_token is null from accounts WHERE normalized_email=?`,
		email.Normalize(),
	).Scan(&userId, &oldKey, &oldSalt, &verified)
	if err == sql.ErrNoRows {
		err = ErrWrongCredentials
	}
	if err != nil {
		return
	}
	match, err := oldPassword.Check(oldKey, oldSalt)
	if err == nil && !match {
		err = ErrWrongCredentials
	}
	if err == nil && !verified {
		err = ErrNotVerified
	}
	if err != nil {
		return
	}

	newKey, newSalt, err := newPassword.Create()
	if err != nil {
		return
	}

	res, err := tx.Exec(
		"UPDATE accounts SET key=?, server_salt=?, client_salt_seed=?, updated=datetime('now') WHERE user_id=?",
		newKey, newSalt, clientSaltSeed, userId,
	)
	if err != nil {
		return
	}
	numRows, err := res.RowsAffected()
	if err != nil {
		return
	}
	if numRows == 0 {
		// Very unexpected error!
		err = fmt.Errorf("Password failed to update")
		return
	}

	if encryptedWallet != "" {
		// With a wallet expected: update it.

		res, err = tx.Exec(
			`UPDATE wallets SET encrypted_wallet=?, sequence=?, hmac=?, updated=datetime('now')
			 WHERE user_id=? AND sequence=?`,
			encryptedWallet, sequence, hmac, userId, sequence-1,
		)
		if err != nil {
			return
		}
		numRows, err = res.RowsAffected()
		if err != nil {
			return
		}
		if numRows == 0 {
			err = ErrWrongSequence
			return
		}
	} else {
		// With no wallet expected: assert we have no wallet.

		var dummy string
		err = tx.QueryRow("SELECT 1 FROM wallets WHERE user_id=?", userId).Scan(&dummy)
		if err != sql.ErrNoRows {
			if err == nil {
				// We expected no rows
				err = ErrUnexpectedWallet
				return
			}
			// Some other error
			return
		}
	}

	// Don't care how many I delete here. Might even be zero (no login token
	// while changing password seems plausible). The main reason for this is
	// that we want to prevent any client from saving a subsequent wallet
	// without changing its password first.
	_, err = tx.Exec("DELETE FROM auth_tokens WHERE user_id=?", userId)
	return
}

// It's a public endpoint, we don't really care if the user is verified
func (s *Store) GetClientSaltSeed(email auth.Email) (seed auth.ClientSaltSeed, err error) {
	err = s.db.QueryRow(
		`SELECT client_salt_seed from accounts WHERE normalized_email=?`,
		email.Normalize(),
	).Scan(&seed)
	if err == sql.ErrNoRows {
		err = ErrWrongCredentials
	}
	return
}
