package store

import (
	"database/sql"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// AnnouncementReadRepository is an announcement-read store already bound to
// one tenant. Tenant selection deliberately sits below this API: callers can
// ask only about an email, so accidentally querying another tenant is not a
// call that can be written.
type AnnouncementReadRepository interface {
	LastSeen(email string) time.Time
	MarkSeen(email string, seenAt time.Time) error
}

// AnnouncementReadStorage is the unbound backend implemented by both the JSON
// and SQLite stores. Its marker method keeps the raw, tenant-aware operations
// inside this package; HTTP code receives only AnnouncementReadRepository.
type AnnouncementReadStorage interface {
	announcementReadStorage()
}

var (
	_ AnnouncementReadStorage = (*AnnouncementReadStore)(nil)
	_ AnnouncementReadStorage = (*SQLAnnouncementReadStore)(nil)
)

type boundAnnouncementReadRepository struct {
	storage    announcementReadBackend
	tenantSlug string
}

type announcementReadBackend interface {
	lastSeen(tenantSlug string, email string) time.Time
	markSeen(tenantSlug string, email string, seenAt time.Time) error
}

// BindAnnouncementReadRepository is the boundary used by tenant middleware.
// An empty tenant is rejected, so even incorrect middleware wiring fails
// closed instead of producing an unscoped repository.
func BindAnnouncementReadRepository(storage AnnouncementReadStorage, tenantSlug string) (AnnouncementReadRepository, bool) {
	tenantSlug = textutil.Slug(tenantSlug)
	backend, ok := storage.(announcementReadBackend)
	if !ok || tenantSlug == "" {
		return nil, false
	}
	return &boundAnnouncementReadRepository{storage: backend, tenantSlug: tenantSlug}, true
}

func (r *boundAnnouncementReadRepository) LastSeen(email string) time.Time {
	if r == nil || r.storage == nil {
		return time.Time{}
	}
	return r.storage.lastSeen(r.tenantSlug, email)
}

func (r *boundAnnouncementReadRepository) MarkSeen(email string, seenAt time.Time) error {
	if r == nil || r.storage == nil {
		return nil
	}
	return r.storage.markSeen(r.tenantSlug, email, seenAt)
}

// SQLAnnouncementReadStore records the per-user last-seen announcement time.
// Table from migration 0007.
type SQLAnnouncementReadStore struct {
	db *sql.DB
}

func NewSQLAnnouncementReadStore(db *sql.DB) *SQLAnnouncementReadStore {
	return &SQLAnnouncementReadStore{db: db}
}

func (*SQLAnnouncementReadStore) announcementReadStorage() {}

func (s *SQLAnnouncementReadStore) lastSeen(tenantSlug string, email string) time.Time {
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
		`SELECT seen_at FROM announcement_reads WHERE tenant_slug = $1 AND email = $2`, tenantSlug, email,
	).Scan(&raw); err != nil {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func (s *SQLAnnouncementReadStore) markSeen(tenantSlug string, email string, seenAt time.Time) error {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	email = textutil.Email(email)
	if tenantSlug == "" || email == "" {
		return nil
	}
	_, err := s.db.Exec(
		`INSERT INTO announcement_reads(tenant_slug, email, seen_at) VALUES($1, $2, $3)
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
			`INSERT INTO announcement_reads(tenant_slug, email, seen_at) VALUES($1, $2, $3)
			 ON CONFLICT(tenant_slug, email) DO NOTHING`,
			tenant, email, p.at.UTC().Format(time.RFC3339Nano),
		); err != nil {
			return err
		}
	}
	return nil
}
