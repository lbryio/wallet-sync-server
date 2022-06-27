package store

// TODO - DeviceId - What about clients that lie about deviceId? Maybe require a certain format to make sure it gives a real value? Something it wouldn't come up with by accident.

import (
	"database/sql"

	"errors"
	"fmt"
	"github.com/mattn/go-sqlite3"
	"log"
	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/wallet"
	"time"
)

var (
	ErrDuplicateToken = fmt.Errorf("Token already exists for this user and device")
	ErrNoToken        = fmt.Errorf("Token does not exist for this user and device")

	ErrDuplicateWallet = fmt.Errorf("Wallet already exists for this user")
	ErrNoWallet        = fmt.Errorf("Wallet does not exist for this user at this sequence")

	ErrWrongSequence = fmt.Errorf("Wallet could not be updated to this sequence")

	ErrDuplicateEmail   = fmt.Errorf("Email already exists for this user")
	ErrDuplicateAccount = fmt.Errorf("User already has an account")

	ErrNoUId = fmt.Errorf("User Id not found with these credentials")
)

// For test stubs
type StoreInterface interface {
	SaveToken(*auth.AuthToken) error
	GetToken(auth.TokenString) (*auth.AuthToken, error)
	SetWallet(auth.UserId, wallet.EncryptedWallet, wallet.Sequence, wallet.WalletHmac) error
	GetWallet(auth.UserId) (wallet.EncryptedWallet, wallet.Sequence, wallet.WalletHmac, error)
	GetUserId(auth.Email, auth.Password) (auth.UserId, error)
	CreateAccount(auth.Email, auth.Password) (err error)
}

type Store struct {
	db *sql.DB
}

func (s *Store) Init(fileName string) {
	db, err := sql.Open("sqlite3", fileName)
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
			PRIMARY KEY (user_id, device_id)
		);
		CREATE TABLE IF NOT EXISTS wallets(
			user_id INTEGER NOT NULL,
			encrypted_wallet TEXT NOT NULL,
			sequence INTEGER NOT NULL,
			hmac TEXT NOT NULL,
			PRIMARY KEY (user_id)
			FOREIGN KEY (user_id) REFERENCES accounts(user_id)
		);
		CREATE TABLE IF NOT EXISTS accounts(
			email TEXT NOT NULL UNIQUE,
			user_id INTEGER PRIMARY KEY AUTOINCREMENT,
			password TEXT NOT NULL
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
func (s *Store) GetToken(token auth.TokenString) (*auth.AuthToken, error) {
	expirationCutoff := time.Now().UTC()

	rows, err := s.db.Query(
		"SELECT * FROM auth_tokens WHERE token=? AND expiration>?", token, expirationCutoff,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var authToken auth.AuthToken
	for rows.Next() {

		err := rows.Scan(
			&authToken.Token,
			&authToken.UserId,
			&authToken.DeviceId,
			&authToken.Scope,
			&authToken.Expiration,
		)

		if err != nil {
			return nil, err
		}
		return &authToken, nil
	}
	return nil, ErrNoToken // TODO - will need to test
}

func (s *Store) insertToken(authToken *auth.AuthToken, expiration time.Time) (err error) {
	_, err = s.db.Exec(
		"INSERT INTO auth_tokens (token, user_id, device_id, scope, expiration) values(?,?,?,?,?)",
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
		err = ErrNoToken
	}
	return
}

func (s *Store) SaveToken(token *auth.AuthToken) (err error) {
	// TODO: For psql, do upsert here instead of separate insertToken and updateToken functions

	// TODO - Should we auto-delete expired tokens?

	expiration := time.Now().UTC().Add(time.Hour * 24 * 14)

	// This is most likely not the first time calling this function for this
	// device, so there's probably already a token in there.
	err = s.updateToken(token, expiration)

	if err == ErrNoToken {
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

func (s *Store) GetWallet(userId auth.UserId) (encryptedWallet wallet.EncryptedWallet, sequence wallet.Sequence, hmac wallet.WalletHmac, err error) {
	rows, err := s.db.Query(
		"SELECT encrypted_wallet, sequence, hmac FROM wallets WHERE user_id=?",
		userId,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(
			&encryptedWallet,
			&sequence,
			&hmac,
		)
		return
	}
	err = ErrNoWallet
	return
}

func (s *Store) insertFirstWallet(
	userId auth.UserId,
	encryptedWallet wallet.EncryptedWallet,
	hmac wallet.WalletHmac,
) (err error) {
	// This will only be used to attempt to insert the first wallet (sequence=1).
	//   The database will enforce that this will not be set if this user already
	//   has a wallet.
	_, err = s.db.Exec(
		"INSERT INTO wallets (user_id, encrypted_wallet, sequence, hmac) values(?,?,?,?)",
		userId, encryptedWallet, 1, hmac,
	)

	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		// I initially expected to need to check for ErrConstraintUnique.
		// Maybe for psql it will be?
		if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintPrimaryKey) {
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
	// This will be used for wallets with sequence > 1.
	// Use the database to enforce that we only update if we are incrementing the sequence.
	// This way, if two clients attempt to update at the same time, it will return
	// ErrNoWallet for the second one.
	res, err := s.db.Exec(
		"UPDATE wallets SET encrypted_wallet=?, sequence=?, hmac=? WHERE user_id=? AND sequence=?",
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
		err = ErrNoWallet
	}
	return
}

// Assumption: Sequence has been validated (>=1)
func (s *Store) SetWallet(userId auth.UserId, encryptedWallet wallet.EncryptedWallet, sequence wallet.Sequence, hmac wallet.WalletHmac) (err error) {
	if sequence == 1 {
		// If sequence == 1, the client assumed that this is our first
		// wallet. Try to insert. If we get a conflict, the client
		// assumed incorrectly and we proceed below to return the latest
		// wallet from the db.
		err = s.insertFirstWallet(userId, encryptedWallet, hmac)
		if err == ErrDuplicateWallet {
			// A wallet already exists. That means the input sequence should not be 1.
			// To the caller, this means the sequence was wrong.
			err = ErrWrongSequence
		}
	} else {
		// If sequence > 1, the client assumed that it is replacing wallet
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
	rows, err := s.db.Query(
		`SELECT user_id from accounts WHERE email=? AND password=?`,
		email, password.Obfuscate(),
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&userId)
		return
	}
	err = ErrNoUId
	return
}

/////////////
// Account //
/////////////

func (s *Store) CreateAccount(email auth.Email, password auth.Password) (err error) {
	// userId auto-increments
	_, err = s.db.Exec(
		"INSERT INTO accounts (email, password) values(?,?)",
		email, password.Obfuscate(),
	)

	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
			err = ErrDuplicateAccount
		}
	}

	return
}
