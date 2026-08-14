package store

import (
	"database/sql"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// AnnouncementReadStorage is the behaviour both the JSON AnnouncementReadStore
// and the SQLite SQLAnnouncementReadStore satisfy (HAUSV-168). Key is
// (tenant, email).
type AnnouncementReadStorage interface {
	LastSeen(tenantSlug string, email string) time.Time
	MarkSeen(tenantSlug string, email string, seenAt time.Time) error
}

var (
	_ AnnouncementReadStorage = (*AnnouncementReadStore)(nil)
	_ AnnouncementReadStorage = (*SQLAnnouncementReadStore)(nil)
)

// SQLAnnouncementReadStore records the per-user last-seen announcement time.
// Table from migration 0007.
type SQLAnnouncementReadStore struct {
	db *sql.DB
}

func NewSQLAnnouncementReadStore(db *sql.DB) *SQLAnnouncementReadStore {
	return &SQLAnnouncementReadStore{db: db}
}

func (s *SQLAnnouncementReadStore) LastSeen(tenantSlug string, email string) time.Time {
	if s == nil {
		return time.Time{}
	}
	tenantSlug = textutil.Slug(tenantSlug)
	email = textutil.Email(email)
	if tenantSlug == "" || email == "" {
		return time.Time{}
	}
	var raw string
	if err := s.db.QueryRow(
		`SELECT seen_at FROM announcement_reads WHERE tenant_slug = ? AND email = ?`, tenantSlug, email,
	).Scan(&raw); err != nil {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func (s *SQLAnnouncementReadStore) MarkSeen(tenantSlug string, email string, seenAt time.Time) error {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	email = textutil.Email(email)
	if tenantSlug == "" || email == "" {
		return nil
	}
	_, err := s.db.Exec(
		`INSERT INTO announcement_reads(tenant_slug, email, seen_at) VALUES(?, ?, ?)
		 ON CONFLICT(tenant_slug, email) DO UPDATE SET seen_at = excluded.seen_at`,
		tenantSlug, email, seenAt.UTC().Format(time.RFC3339Nano),
	)
	return err
}

// ImportReads copies the nested (tenant -> email -> time) map, each pair only if
// absent (clobber-safe) (HAUSV-170).
func (s *SQLAnnouncementReadStore) ImportReads(src *AnnouncementReadStore) error {
	if src == nil {
		return nil
	}
	src.mu.Lock()
	type pair struct {
		tenant, email string
		at            time.Time
	}
	pairs := []pair{}
	for tenant, byEmail := range src.data.Seen {
		for email, at := range byEmail {
			pairs = append(pairs, pair{tenant, email, at})
		}
	}
	src.mu.Unlock()
	for _, p := range pairs {
		tenant := textutil.Slug(p.tenant)
		email := textutil.Email(p.email)
		if tenant == "" || email == "" {
			continue
		}
		if _, err := s.db.Exec(
			`INSERT INTO announcement_reads(tenant_slug, email, seen_at) VALUES(?, ?, ?)
			 ON CONFLICT(tenant_slug, email) DO NOTHING`,
			tenant, email, p.at.UTC().Format(time.RFC3339Nano),
		); err != nil {
			return err
		}
	}
	return nil
}
