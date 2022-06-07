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

	ErrDuplicateWalletState = fmt.Errorf("WalletState already exists for this user")
	ErrNoWalletState        = fmt.Errorf("WalletState does not exist for this user at this sequence")

	ErrDuplicateEmail   = fmt.Errorf("Email already exists for this user")
	ErrDuplicateAccount = fmt.Errorf("User already has an account")

	ErrNoUId = fmt.Errorf("User Id not found with these credentials")
)

// For test stubs
type StoreInterface interface {
	SaveToken(*auth.AuthToken) error
	GetToken(auth.AuthTokenString) (*auth.AuthToken, error)
	SetWalletState(auth.UserId, string, int, wallet.WalletStateHmac) (string, wallet.WalletStateHmac, bool, error)
	GetWalletState(auth.UserId) (string, wallet.WalletStateHmac, error)
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
	// We store `sequence` as a seprate field in the `wallet_state` table, even
	// though it's also saved as part of the `walle_state_blob` column. We do
	// this for transaction safety. For instance, let's say two different clients
	// are trying to update the sequence from 5 to 6. The update command will
	// specify "WHERE sequence=5". Only one of these commands will succeed, and
	// the other will get back an error.

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

	// TODO does it actually fail with empty "NOT NULL" fields?
	query := `
		CREATE TABLE IF NOT EXISTS auth_tokens(
			token TEXT NOT NULL UNIQUE,
			user_id INTEGER NOT NULL,
			device_id TEXT NOT NULL,
			scope TEXT NOT NULL,
			expiration DATETIME NOT NULL,
			PRIMARY KEY (user_id, device_id)
		);
		CREATE TABLE IF NOT EXISTS wallet_states(
			user_id INTEGER NOT NULL,
			wallet_state_blob TEXT NOT NULL,
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
func (s *Store) GetToken(token auth.AuthTokenString) (*auth.AuthToken, error) {
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

/////////////////////////////////
// Wallet State / Download Key //
/////////////////////////////////

func (s *Store) GetWalletState(userId auth.UserId) (walletStateJson string, hmac wallet.WalletStateHmac, err error) {
	rows, err := s.db.Query(
		"SELECT wallet_state_blob, hmac FROM wallet_states WHERE user_id=?",
		userId,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(
			&walletStateJson,
			&hmac,
		)
		return
	}
	err = ErrNoWalletState
	return
}

func (s *Store) insertFirstWalletState(
	userId auth.UserId,
	walletStateJson string,
	hmac wallet.WalletStateHmac,
) (err error) {
	// This will only be used to attempt to insert the first wallet state
	//   (sequence=1). The database will enforce that this will not be set
	//   if this user already has a walletState.
	_, err = s.db.Exec(
		"INSERT INTO wallet_states (user_id, wallet_state_blob, sequence, hmac) values(?,?,?,?)",
		userId, walletStateJson, 1, hmac,
	)

	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		// I initially expected to need to check for ErrConstraintUnique.
		// Maybe for psql it will be?
		if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintPrimaryKey) {
			err = ErrDuplicateWalletState
		}
	}

	return
}

func (s *Store) updateWalletStateToSequence(
	userId auth.UserId,
	walletStateJson string,
	sequence int,
	hmac wallet.WalletStateHmac,
) (err error) {
	// This will be used for wallet states with sequence > 1.
	// Use the database to enforce that we only update if we are incrementing the sequence.
	// This way, if two clients attempt to update at the same time, it will return
	// ErrNoWalletState for the second one.
	res, err := s.db.Exec(
		"UPDATE wallet_states SET wallet_state_blob=?, sequence=?, hmac=? WHERE user_id=? AND sequence=?",
		walletStateJson, sequence, hmac, userId, sequence-1,
	)
	if err != nil {
		return
	}

	numRows, err := res.RowsAffected()
	if err != nil {
		return
	}
	if numRows == 0 {
		err = ErrNoWalletState
	}
	return
}

// Assumption: walletState has been validated (sequence >=1, etc)
// Assumption: Sequence matches walletState.Sequence()
// Sequence is only passed in here to avoid deserializing walletStateJson again
// WalletState *struct* is not passed in because the clients need the exact string to match the hmac
func (s *Store) SetWalletState(
	userId auth.UserId,
	walletStateJson string,
	sequence int,
	hmac wallet.WalletStateHmac,
) (latestWalletStateJson string, latestHmac wallet.WalletStateHmac, updated bool, err error) {
	if sequence == 1 {
		// If sequence == 1, the client assumed that this is our first
		// walletState. Try to insert. If we get a conflict, the client
		// assumed incorrectly and we proceed below to return the latest
		// walletState from the db.
		err = s.insertFirstWalletState(userId, walletStateJson, hmac)
		if err == nil {
			// Successful update
			latestWalletStateJson = walletStateJson
			latestHmac = hmac
			updated = true
			return
		} else if err != ErrDuplicateWalletState {
			// Unsuccessful update for reasons other than sequence conflict
			return
		}
	} else {
		// If sequence > 1, the client assumed that it is replacing walletState
		// with sequence - 1. Explicitly try to update the walletState with
		// sequence - 1. If we updated no rows, the client assumed incorrectly
		// and we proceed below to return the latest walletState from the db.
		err = s.updateWalletStateToSequence(userId, walletStateJson, sequence, hmac)
		if err == nil {
			latestWalletStateJson = walletStateJson
			latestHmac = hmac
			updated = true
			return
		} else if err != ErrNoWalletState {
			return
		}
	}

	// We failed to update above due to a sequence conflict. Perhaps the client
	// was unaware of an update done by another client. Let's send back the latest
	// version right away so the requesting client can take care of it.
	//
	// Note that this means that `err` will not be `nil` at this point, but we
	// already accounted for it with `updated=false`. Instead, we'll pass on any
	// errors from calling `GetWalletState`.
	latestWalletStateJson, latestHmac, err = s.GetWalletState(userId)
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
		// I initially expected to need to check for ErrConstraintUnique.
		// Maybe for psql it will be?
		// TODO - is this right? Does the above comment explain that it's backwards
		// from what I would have expected? Or did I do this backwards?
		if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintPrimaryKey) {
			err = ErrDuplicateEmail
		}
		if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
			err = ErrDuplicateAccount
		}
	}

	return
}
