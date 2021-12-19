package main // TODO - make it its own `store` package later

import (
	"database/sql"

	"errors"
	"fmt"
	"github.com/mattn/go-sqlite3"
	"log"
	"time"
)

var (
	ErrDuplicateToken = fmt.Errorf("Token already exists for this user and device")
	ErrNoToken        = fmt.Errorf("Token does not exist for this user and device")
)

type StoreInterface interface {
	SaveToken(*AuthToken) error
}

type Store struct {
	db *sql.DB
}

func (s *Store) Migrate() error {
	query := `
		CREATE TABLE IF NOT EXISTS auth_tokens(
			token TEXT NOT NULL,
			public_key TEXT NOT NULL,
			device_id TEXT NOT NULL,
			expiration DATETIME NOT NULL,
			PRIMARY KEY (public_key, device_id)
		);
	`

	_, err := s.db.Exec(query)
	return err
}

func (s *Store) GetToken(pubKey PublicKey, deviceID string) (*AuthToken, error) {
	expirationCutoff := time.Now().UTC()

	rows, err := s.db.Query("SELECT * FROM auth_tokens WHERE public_key=? AND device_id=? AND expiration>?",
		pubKey, deviceID, expirationCutoff,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var authToken AuthToken
	for rows.Next() {

		err := rows.Scan(
			&authToken.Token,
			&authToken.PubKey,
			&authToken.DeviceID,
			&authToken.Expiration,
		)

		if err != nil {
			return nil, err
		}
		return &authToken, nil
	}
	return nil, nil
}

func (s *Store) insertToken(authToken *AuthToken, expiration time.Time) (err error) {
	_, err = s.db.Exec(
		"INSERT INTO auth_tokens (token, public_key, device_id, expiration) values(?,?,?,?)",
		authToken.Token, authToken.PubKey, authToken.DeviceID, expiration,
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

func (s *Store) updateToken(authToken *AuthToken, experation time.Time) (err error) {
	res, err := s.db.Exec(
		"UPDATE auth_tokens SET token=?, expiration=? WHERE public_key=? AND device_id=?",
		authToken.Token, experation, authToken.PubKey, authToken.DeviceID,
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

func (s *Store) SaveToken(token *AuthToken) (err error) {
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
			err = s.updateToken(token, expiration)
		}
	}
	if err == nil {
		token.Expiration = &expiration
	}
	return
}

func (s *Store) Init(fileName string) {
	db, err := sql.Open("sqlite3", fileName)
	if err != nil {
		log.Fatal(err)
	}
	s.db = db
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
