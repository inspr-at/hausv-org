package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/markus-barta/hausv-org/internal/textutil"
)

// ProfileOverlayStorage is the behaviour both the JSON ProfileOverlayStore and
// the SQLite SQLProfileOverlayStore satisfy (HAUSV-168).
type ProfileOverlayStorage interface {
	Get(email string) (ProfileOverlay, bool)
	Set(email string, overlay ProfileOverlay) error
}

var (
	_ ProfileOverlayStorage = (*ProfileOverlayStore)(nil)
	_ ProfileOverlayStorage = (*SQLProfileOverlayStore)(nil)
)

// SQLProfileOverlayStore is the SQLite-backed self-service profile overlay
// store. Table created by migration 0003_profile_overlays (internal/db).
type SQLProfileOverlayStore struct {
	db *sql.DB
}

func NewSQLProfileOverlayStore(db *sql.DB) *SQLProfileOverlayStore {
	return &SQLProfileOverlayStore{db: db}
}

func (s *SQLProfileOverlayStore) Get(email string) (ProfileOverlay, bool) {
	if s == nil {
		return ProfileOverlay{}, false
	}
	email = textutil.Email(email)
	if email == "" {
		return ProfileOverlay{}, false
	}
	var (
		o         ProfileOverlay
		optIn     int
		updatedAt string
	)
	if err := s.db.QueryRow(
		`SELECT title, first_name, last_name, phone, directory_opt_in, updated_at
		 FROM profile_overlays WHERE email = ?`, email,
	).Scan(&o.Title, &o.FirstName, &o.LastName, &o.Phone, &optIn, &updatedAt); err != nil {
		return ProfileOverlay{}, false
	}
	o.DirectoryOptIn = optIn != 0
	if t, err := time.Parse(time.RFC3339Nano, updatedAt); err == nil {
		o.UpdatedAt = t.UTC()
	}
	return o, true
}

func (s *SQLProfileOverlayStore) Set(email string, overlay ProfileOverlay) error {
	if s == nil {
		return nil
	}
	email = textutil.Email(email)
	if email == "" {
		return fmt.Errorf("invalid profile email")
	}
	overlay = NormalizeProfileOverlay(overlay)
	overlay.UpdatedAt = time.Now().UTC()
	return s.upsert(email, overlay)
}

func (s *SQLProfileOverlayStore) upsert(email string, o ProfileOverlay) error {
	optIn := 0
	if o.DirectoryOptIn {
		optIn = 1
	}
	_, err := s.db.Exec(
		`INSERT INTO profile_overlays(email, title, first_name, last_name, phone, directory_opt_in, updated_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(email) DO UPDATE SET
		   title=excluded.title, first_name=excluded.first_name, last_name=excluded.last_name,
		   phone=excluded.phone, directory_opt_in=excluded.directory_opt_in, updated_at=excluded.updated_at`,
		email, o.Title, o.FirstName, o.LastName, o.Phone, optIn, o.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	return err
}

// ImportOverlays copies records from a JSON ProfileOverlayStore, each only if
// not already present (clobber-safe on every boot), preserving the original
// UpdatedAt (HAUSV-170).
func (s *SQLProfileOverlayStore) ImportOverlays(src *ProfileOverlayStore) error {
	if src == nil {
		return nil
	}
	src.mu.Lock()
	snapshot := make(map[string]ProfileOverlay, len(src.data.Profiles))
	for email, o := range src.data.Profiles {
		snapshot[email] = o
	}
	src.mu.Unlock()
	for rawEmail, o := range snapshot {
		email := textutil.Email(rawEmail)
		if email == "" {
			continue
		}
		optIn := 0
		if o.DirectoryOptIn {
			optIn = 1
		}
		updatedAt := o.UpdatedAt.UTC().Format(time.RFC3339Nano)
		if o.UpdatedAt.IsZero() {
			updatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		}
		if _, err := s.db.Exec(
			`INSERT INTO profile_overlays(email, title, first_name, last_name, phone, directory_opt_in, updated_at)
			 VALUES(?, ?, ?, ?, ?, ?, ?) ON CONFLICT(email) DO NOTHING`,
			email, o.Title, o.FirstName, o.LastName, o.Phone, optIn, updatedAt,
		); err != nil {
			return err
		}
	}
	return nil
}
