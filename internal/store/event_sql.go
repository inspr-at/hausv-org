package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

type EventRepository interface {
	Create(item HouseEvent) (HouseEvent, error)
	Update(id string, updated HouseEvent) (bool, error)
	Delete(id string) (bool, error)
	List() []HouseEvent
	Upcoming(now time.Time) []HouseEvent
}

type EventStorage interface {
	eventStorage()
}

var (
	_ EventStorage = (*EventStore)(nil)
	_ EventStorage = (*SQLEventStore)(nil)
)

type boundEventRepository struct {
	storage eventBackend
	tenant  TenantRef
}

type eventBackend interface {
	create(tenantSlug string, item HouseEvent) (HouseEvent, error)
	update(tenantSlug string, id string, updated HouseEvent) (bool, error)
	delete(tenantSlug string, id string) (bool, error)
	list(tenantSlug string) []HouseEvent
	upcoming(tenantSlug string, now time.Time) []HouseEvent
}

func BindEventRepository(storage EventStorage, tenant TenantRef) (EventRepository, bool) {
	resolvedTenant, tenantOK := validTenantRef(tenant)
	backend, ok := storage.(eventBackend)
	if !ok || !tenantOK {
		return nil, false
	}
	return &boundEventRepository{storage: backend, tenant: resolvedTenant}, true
}

func (r *boundEventRepository) Create(item HouseEvent) (HouseEvent, error) {
	return r.storage.create(r.tenant.Slug, item)
}
func (r *boundEventRepository) Update(id string, item HouseEvent) (bool, error) {
	return r.storage.update(r.tenant.Slug, id, item)
}
func (r *boundEventRepository) Delete(id string) (bool, error) {
	return r.storage.delete(r.tenant.Slug, id)
}
func (r *boundEventRepository) List() []HouseEvent { return r.storage.list(r.tenant.Slug) }
func (r *boundEventRepository) Upcoming(now time.Time) []HouseEvent {
	return r.storage.upcoming(r.tenant.Slug, now)
}

// SQLEventStore keeps each event as a JSON document keyed by (tenant, id).
// Table from migration 0009.
type SQLEventStore struct {
	db *sql.DB
}

func NewSQLEventStore(db *sql.DB) *SQLEventStore {
	return &SQLEventStore{db: db}
}

func (*SQLEventStore) eventStorage() {}

func (s *SQLEventStore) writeTx(tx *sql.Tx, item HouseEvent) error {
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO events(tenant_slug, id, data) VALUES($1, $2, $3)
		 ON CONFLICT(tenant_slug, id) DO UPDATE SET data=excluded.data`,
		textutil.Slug(item.TenantSlug), item.ID, string(blob),
	)
	return err
}

func (s *SQLEventStore) create(tenantSlug string, item HouseEvent) (HouseEvent, error) {
	item.TenantSlug = tenantSlug
	now := time.Now().UTC()
	item.ID = ""
	item.CreatedAt = now
	item.UpdatedAt = now
	normalized, ok := NormalizeHouseEvent(item)
	if !ok {
		return HouseEvent{}, fmt.Errorf("invalid event")
	}
	id, err := randomToken(12)
	if err != nil {
		return HouseEvent{}, err
	}
	normalized.ID = id
	tx, err := s.db.Begin()
	if err != nil {
		return HouseEvent{}, err
	}
	defer tx.Rollback()
	if err := s.writeTx(tx, normalized); err != nil {
		return HouseEvent{}, err
	}
	if err := tx.Commit(); err != nil {
		return HouseEvent{}, err
	}
	return normalized, nil
}

func (s *SQLEventStore) update(tenantSlug string, id string, updated HouseEvent) (bool, error) {
	updated.TenantSlug = tenantSlug
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	normalized, ok := NormalizeHouseEvent(updated)
	if !ok {
		return false, fmt.Errorf("invalid event")
	}
	tenant := textutil.Slug(normalized.TenantSlug)
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var data string
	if err := tx.QueryRow(`SELECT data FROM events WHERE tenant_slug=$1 AND id=$2`, tenant, id).Scan(&data); err != nil {
		return false, nil
	}
	var existing HouseEvent
	if err := json.Unmarshal([]byte(data), &existing); err != nil {
		return false, nil
	}
	normalized.ID = existing.ID
	normalized.CreatedAt = existing.CreatedAt
	normalized.UpdatedAt = time.Now().UTC()
	if normalized.AuthorEmail == "" {
		normalized.AuthorEmail = existing.AuthorEmail
	}
	if normalized.AuthorName == "" {
		normalized.AuthorName = existing.AuthorName
	}
	if err := s.writeTx(tx, normalized); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *SQLEventStore) delete(tenantSlug string, id string) (bool, error) {
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return false, nil
	}
	res, err := s.db.Exec(`DELETE FROM events WHERE tenant_slug=$1 AND id=$2`, tenantSlug, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SQLEventStore) allForTenant(tenantSlug string) []HouseEvent {
	tenantSlug = textutil.Slug(tenantSlug)
	rows, err := s.db.Query(`SELECT data FROM events WHERE tenant_slug=$1`, tenantSlug)
	if err != nil {
		return []HouseEvent{}
	}
	defer rows.Close()
	out := []HouseEvent{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var item HouseEvent
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}
		out = append(out, item)
	}
	return out
}

func (s *SQLEventStore) list(tenantSlug string) []HouseEvent {
	out := s.allForTenant(tenantSlug)
	SortEvents(out)
	return out
}

func (s *SQLEventStore) upcoming(tenantSlug string, now time.Time) []HouseEvent {
	out := []HouseEvent{}
	for _, item := range s.allForTenant(tenantSlug) {
		if EventRollsOffAt(item).After(now) {
			out = append(out, item)
		}
	}
	SortEvents(out)
	return out
}

// ImportEvents copies records from a JSON store, each only if absent
// (clobber-safe) (HAUSV-170).
func (s *SQLEventStore) ImportEvents(src *EventStore) error {
	if src == nil {
		return nil
	}
	src.mu.Lock()
	snapshot := append([]HouseEvent(nil), src.data.Events...)
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
			`INSERT INTO events(tenant_slug, id, data) VALUES($1, $2, $3) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			textutil.Slug(item.TenantSlug), item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}
