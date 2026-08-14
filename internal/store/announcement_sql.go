package store

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// AnnouncementStorage is the behaviour both the JSON AnnouncementStore and the
// SQLite SQLAnnouncementStore satisfy (HAUSV-168).
type AnnouncementStorage interface {
	Create(item Announcement) (Announcement, error)
	Update(id string, updated Announcement) (bool, error)
	Delete(tenantSlug string, id string) (bool, error)
	Visible(tenantSlug string, now time.Time) []Announcement
	Archive(tenantSlug string, now time.Time) []Announcement
	ListTenant(tenantSlug string) []Announcement
}

var (
	_ AnnouncementStorage = (*AnnouncementStore)(nil)
	_ AnnouncementStorage = (*SQLAnnouncementStore)(nil)
)

// SQLAnnouncementStore keeps each announcement as a JSON document keyed by
// (tenant, id). Table from migration 0008.
type SQLAnnouncementStore struct {
	db *sql.DB
}

func NewSQLAnnouncementStore(db *sql.DB) *SQLAnnouncementStore {
	return &SQLAnnouncementStore{db: db}
}

func (s *SQLAnnouncementStore) writeTx(tx *sql.Tx, item Announcement) error {
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO announcements(tenant_slug, id, data) VALUES(?, ?, ?)
		 ON CONFLICT(tenant_slug, id) DO UPDATE SET data=excluded.data`,
		textutil.Slug(item.TenantSlug), item.ID, string(blob),
	)
	return err
}

func (s *SQLAnnouncementStore) Create(item Announcement) (Announcement, error) {
	now := time.Now().UTC()
	item.ID = ""
	item.CreatedAt = now
	item.UpdatedAt = now
	if item.PublishedAt.IsZero() {
		item.PublishedAt = now
	}
	if item.Category == "" {
		item.Category = "Info"
	}
	id, err := randomToken(12)
	if err != nil {
		return Announcement{}, err
	}
	item.ID = id
	tx, err := s.db.Begin()
	if err != nil {
		return Announcement{}, err
	}
	defer tx.Rollback()
	if err := s.writeTx(tx, item); err != nil {
		return Announcement{}, err
	}
	if err := tx.Commit(); err != nil {
		return Announcement{}, err
	}
	return item, nil
}

func (s *SQLAnnouncementStore) Update(id string, updated Announcement) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	tenant := textutil.Slug(updated.TenantSlug)
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var data string
	if err := tx.QueryRow(`SELECT data FROM announcements WHERE tenant_slug=? AND id=?`, tenant, id).Scan(&data); err != nil {
		return false, nil
	}
	var existing Announcement
	if err := json.Unmarshal([]byte(data), &existing); err != nil {
		return false, nil
	}
	updated.ID = existing.ID
	updated.CreatedAt = existing.CreatedAt
	updated.UpdatedAt = time.Now().UTC()
	if updated.PublishedAt.IsZero() {
		updated.PublishedAt = existing.PublishedAt
	}
	if updated.AuthorEmail == "" {
		updated.AuthorEmail = existing.AuthorEmail
	}
	if updated.AuthorName == "" {
		updated.AuthorName = existing.AuthorName
	}
	if err := s.writeTx(tx, updated); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *SQLAnnouncementStore) Delete(tenantSlug string, id string) (bool, error) {
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	res, err := s.db.Exec(`DELETE FROM announcements WHERE tenant_slug=? AND id=?`, tenantSlug, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SQLAnnouncementStore) allForTenant(tenantSlug string) []Announcement {
	tenantSlug = textutil.Slug(tenantSlug)
	rows, err := s.db.Query(`SELECT data FROM announcements WHERE tenant_slug=?`, tenantSlug)
	if err != nil {
		return []Announcement{}
	}
	defer rows.Close()
	out := []Announcement{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var item Announcement
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}
		out = append(out, item)
	}
	return out
}

func (s *SQLAnnouncementStore) Visible(tenantSlug string, now time.Time) []Announcement {
	out := []Announcement{}
	for _, item := range s.allForTenant(tenantSlug) {
		if item.PublishedAt.After(now) {
			continue
		}
		if item.ExpiresAt != nil && !item.ExpiresAt.After(now) {
			continue
		}
		out = append(out, item)
	}
	SortAnnouncements(out)
	return out
}

func (s *SQLAnnouncementStore) Archive(tenantSlug string, now time.Time) []Announcement {
	out := []Announcement{}
	for _, item := range s.allForTenant(tenantSlug) {
		if item.PublishedAt.After(now) {
			continue
		}
		out = append(out, item)
	}
	SortAnnouncements(out)
	return out
}

func (s *SQLAnnouncementStore) ListTenant(tenantSlug string) []Announcement {
	out := s.allForTenant(tenantSlug)
	SortAnnouncements(out)
	return out
}

// ImportAnnouncements copies records from a JSON store, each only if absent
// (clobber-safe) (HAUSV-170).
func (s *SQLAnnouncementStore) ImportAnnouncements(src *AnnouncementStore) error {
	if src == nil {
		return nil
	}
	src.mu.Lock()
	snapshot := append([]Announcement(nil), src.data.Announcements...)
	src.mu.Unlock()
	for _, item := range snapshot {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		blob, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(
			`INSERT INTO announcements(tenant_slug, id, data) VALUES(?, ?, ?) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			textutil.Slug(item.TenantSlug), item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}
