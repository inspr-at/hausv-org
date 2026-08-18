package store

import (
	"fmt"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
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
	db *TenantDB
}

func NewSQLProfileOverlayStore(db *TenantDB) *SQLProfileOverlayStore {
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
		optIn     bool
		updatedAt string
	)
	unscoped := s.db.Unscoped("profile_overlays has no tenant_id: the self-service name and phone follow the person across every house")
	if err := unscoped.QueryRow(
		`SELECT title, first_name, last_name, phone, directory_opt_in, updated_at
		 FROM profile_overlays WHERE email = $1`, email,
	).Scan(&o.Title, &o.FirstName, &o.LastName, &o.Phone, &optIn, &updatedAt); err != nil {
		return ProfileOverlay{}, false
	}
	o.DirectoryOptIn = optIn
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
	_, err := s.db.Unscoped("profile_overlays has no tenant_id: the self-service name and phone follow the person across every house").Exec(
		`INSERT INTO profile_overlays(email, title, first_name, last_name, phone, directory_opt_in, updated_at)
		 VALUES($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT(email) DO UPDATE SET
		   title=excluded.title, first_name=excluded.first_name, last_name=excluded.last_name,
		   phone=excluded.phone, directory_opt_in=excluded.directory_opt_in, updated_at=excluded.updated_at`,
		email, o.Title, o.FirstName, o.LastName, o.Phone, o.DirectoryOptIn, o.UpdatedAt.UTC().Format(time.RFC3339Nano),
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
	imports := s.db.Unscoped("boot import replay of the JSON overlay snapshot into a table with no tenant_id, before the first request")
	for rawEmail, o := range snapshot {
		email := textutil.Email(rawEmail)
		if email == "" {
			continue
		}
		updatedAt := o.UpdatedAt.UTC().Format(time.RFC3339Nano)
		if o.UpdatedAt.IsZero() {
			updatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		}
		if _, err := imports.Exec(
			`INSERT INTO profile_overlays(email, title, first_name, last_name, phone, directory_opt_in, updated_at)
			 VALUES($1, $2, $3, $4, $5, $6, $7) ON CONFLICT(email) DO NOTHING`,
			email, o.Title, o.FirstName, o.LastName, o.Phone, o.DirectoryOptIn, updatedAt,
		); err != nil {
			return err
		}
	}
	return nil
}
