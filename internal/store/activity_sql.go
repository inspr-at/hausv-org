package store

import (
	"database/sql"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// ActivityStorage is the behaviour both the JSON ActivityStore and the SQLite
// SQLActivityStore satisfy. The app depends on this interface so the backend can
// be swapped store-by-store during the SQLite migration (HAUSV-166 / HAUSV-168).
type ActivityStorage interface {
	Touch(email string, at time.Time, authMethod string) error
	Get(email string) (ActivityRecord, bool)
}

// Compile-time proof that both backends satisfy the contract.
var (
	_ ActivityStorage = (*ActivityStore)(nil)
	_ ActivityStorage = (*SQLActivityStore)(nil)
)

// SQLActivityStore is the SQLite-backed login-activity store. Its table is
// created by migration 0002_login_activity (internal/db). Times are stored as
// RFC3339Nano UTC text, matching the JSON store's UTC semantics on round-trip.
type SQLActivityStore struct {
	db *sql.DB
}

func NewSQLActivityStore(db *sql.DB) *SQLActivityStore {
	return &SQLActivityStore{db: db}
}

func (s *SQLActivityStore) Touch(email string, at time.Time, authMethod string) error {
	email = textutil.Email(email)
	if email == "" {
		return nil
	}
	_, err := s.db.Exec(
		`INSERT INTO login_activity(email, last_login, auth_method) VALUES($1, $2, $3)
		 ON CONFLICT(email) DO UPDATE SET last_login = excluded.last_login, auth_method = excluded.auth_method`,
		email, at.UTC().Format(time.RFC3339Nano), authMethod,
	)
	return err
}

func (s *SQLActivityStore) Get(email string) (ActivityRecord, bool) {
	email = textutil.Email(email)
	if email == "" {
		return ActivityRecord{}, false
	}
	var lastLogin, authMethod string
	if err := s.db.QueryRow(
		`SELECT last_login, auth_method FROM login_activity WHERE email = $1`, email,
	).Scan(&lastLogin, &authMethod); err != nil {
		return ActivityRecord{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, lastLogin)
	if err != nil {
		return ActivityRecord{}, false
	}
	return ActivityRecord{LastLogin: t.UTC(), AuthMethod: authMethod}, true
}

// ImportActivity copies records from a JSON ActivityStore into SQLite,
// importing each email only if it is not already present (ON CONFLICT DO
// NOTHING). This makes it safe to run on every boot: a login written to SQLite
// after the swap is never clobbered by the older JSON snapshot (HAUSV-170).
func (s *SQLActivityStore) ImportActivity(src *ActivityStore) error {
	if src == nil {
		return nil
	}
	src.mu.Lock()
	snapshot := make(map[string]ActivityRecord, len(src.data))
	for email, rec := range src.data {
		snapshot[email] = rec
	}
	src.mu.Unlock()
	for rawEmail, rec := range snapshot {
		email := textutil.Email(rawEmail)
		if email == "" {
			continue
		}
		if _, err := s.db.Exec(
			`INSERT INTO login_activity(email, last_login, auth_method) VALUES($1, $2, $3)
			 ON CONFLICT(email) DO NOTHING`,
			email, rec.LastLogin.UTC().Format(time.RFC3339Nano), rec.AuthMethod,
		); err != nil {
			return err
		}
	}
	return nil
}
