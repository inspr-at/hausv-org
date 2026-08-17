package store

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// AnnouncementRepository is an announcement store bound to one tenant.
type AnnouncementRepository interface {
	Create(item Announcement) (Announcement, error)
	Update(id string, updated Announcement) (bool, error)
	Delete(id string) (bool, error)
	Visible(now time.Time) []Announcement
	Archive(now time.Time) []Announcement
	List() []Announcement
}

// AnnouncementStorage is the unbound backend implemented by the JSON and
// SQLite stores. HTTP code receives only AnnouncementRepository.
type AnnouncementStorage interface {
	announcementStorage()
}

var (
	_ AnnouncementStorage = (*AnnouncementStore)(nil)
	_ AnnouncementStorage = (*SQLAnnouncementStore)(nil)
)

type boundAnnouncementRepository struct {
	storage    announcementBackend
	tenantSlug string
}

type announcementBackend interface {
	create(tenantSlug string, item Announcement) (Announcement, error)
	update(tenantSlug string, id string, updated Announcement) (bool, error)
	delete(tenantSlug string, id string) (bool, error)
	visible(tenantSlug string, now time.Time) []Announcement
	archive(tenantSlug string, now time.Time) []Announcement
	list(tenantSlug string) []Announcement
}

func BindAnnouncementRepository(storage AnnouncementStorage, tenantSlug string) (AnnouncementRepository, bool) {
	tenantSlug = textutil.Slug(tenantSlug)
	backend, ok := storage.(announcementBackend)
	if !ok || tenantSlug == "" {
		return nil, false
	}
	return &boundAnnouncementRepository{storage: backend, tenantSlug: tenantSlug}, true
}

func (r *boundAnnouncementRepository) Create(item Announcement) (Announcement, error) {
	return r.storage.create(r.tenantSlug, item)
}

func (r *boundAnnouncementRepository) Update(id string, updated Announcement) (bool, error) {
	return r.storage.update(r.tenantSlug, id, updated)
}

func (r *boundAnnouncementRepository) Delete(id string) (bool, error) {
	return r.storage.delete(r.tenantSlug, id)
}

func (r *boundAnnouncementRepository) Visible(now time.Time) []Announcement {
	return r.storage.visible(r.tenantSlug, now)
}

func (r *boundAnnouncementRepository) Archive(now time.Time) []Announcement {
	return r.storage.archive(r.tenantSlug, now)
}

func (r *boundAnnouncementRepository) List() []Announcement {
	return r.storage.list(r.tenantSlug)
}

// SQLAnnouncementStore keeps each announcement as a JSON document keyed by
// (tenant, id). Table from migration 0008.
type SQLAnnouncementStore struct {
	db *sql.DB
}

func NewSQLAnnouncementStore(db *sql.DB) *SQLAnnouncementStore {
	return &SQLAnnouncementStore{db: db}
}

func (*SQLAnnouncementStore) announcementStorage() {}

func (s *SQLAnnouncementStore) writeTx(tx *sql.Tx, item Announcement) error {
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO announcements(tenant_slug, id, data) VALUES($1, $2, $3)
		 ON CONFLICT(tenant_slug, id) DO UPDATE SET data=excluded.data`,
		textutil.Slug(item.TenantSlug), item.ID, string(blob),
	)
	return err
}

func (s *SQLAnnouncementStore) create(tenantSlug string, item Announcement) (Announcement, error) {
	item.TenantSlug = tenantSlug
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

func (s *SQLAnnouncementStore) update(tenantSlug string, id string, updated Announcement) (bool, error) {
	updated.TenantSlug = tenantSlug
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
	if err := tx.QueryRow(`SELECT data FROM announcements WHERE tenant_slug=$1 AND id=$2`, tenant, id).Scan(&data); err != nil {
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

func (s *SQLAnnouncementStore) delete(tenantSlug string, id string) (bool, error) {
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	res, err := s.db.Exec(`DELETE FROM announcements WHERE tenant_slug=$1 AND id=$2`, tenantSlug, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SQLAnnouncementStore) allForTenant(tenantSlug string) []Announcement {
	tenantSlug = textutil.Slug(tenantSlug)
	rows, err := s.db.Query(`SELECT data FROM announcements WHERE tenant_slug=$1`, tenantSlug)
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

func (s *SQLAnnouncementStore) visible(tenantSlug string, now time.Time) []Announcement {
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

func (s *SQLAnnouncementStore) archive(tenantSlug string, now time.Time) []Announcement {
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

func (s *SQLAnnouncementStore) list(tenantSlug string) []Announcement {
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
			`INSERT INTO announcements(tenant_slug, id, data) VALUES($1, $2, $3) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			textutil.Slug(item.TenantSlug), item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}
