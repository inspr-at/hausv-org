package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// EventStorage is the behaviour both the JSON EventStore and the SQLite
// SQLEventStore satisfy (HAUSV-168).
type EventStorage interface {
	Create(item HouseEvent) (HouseEvent, error)
	Update(id string, updated HouseEvent) (bool, error)
	Delete(tenantSlug string, id string) (bool, error)
	ListTenant(tenantSlug string) []HouseEvent
	Upcoming(tenantSlug string, now time.Time) []HouseEvent
}

var (
	_ EventStorage = (*EventStore)(nil)
	_ EventStorage = (*SQLEventStore)(nil)
)

// SQLEventStore keeps each event as a JSON document keyed by (tenant, id).
// Table from migration 0009.
type SQLEventStore struct {
	db *sql.DB
}

func NewSQLEventStore(db *sql.DB) *SQLEventStore {
	return &SQLEventStore{db: db}
}

func (s *SQLEventStore) writeTx(tx *sql.Tx, item HouseEvent) error {
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO events(tenant_slug, id, data) VALUES(?, ?, ?)
		 ON CONFLICT(tenant_slug, id) DO UPDATE SET data=excluded.data`,
		textutil.Slug(item.TenantSlug), item.ID, string(blob),
	)
	return err
}

func (s *SQLEventStore) Create(item HouseEvent) (HouseEvent, error) {
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

func (s *SQLEventStore) Update(id string, updated HouseEvent) (bool, error) {
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
	if err := tx.QueryRow(`SELECT data FROM events WHERE tenant_slug=? AND id=?`, tenant, id).Scan(&data); err != nil {
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

func (s *SQLEventStore) Delete(tenantSlug string, id string) (bool, error) {
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return false, nil
	}
	res, err := s.db.Exec(`DELETE FROM events WHERE tenant_slug=? AND id=?`, tenantSlug, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SQLEventStore) allForTenant(tenantSlug string) []HouseEvent {
	tenantSlug = textutil.Slug(tenantSlug)
	rows, err := s.db.Query(`SELECT data FROM events WHERE tenant_slug=?`, tenantSlug)
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

func (s *SQLEventStore) ListTenant(tenantSlug string) []HouseEvent {
	out := s.allForTenant(tenantSlug)
	SortEvents(out)
	return out
}

func (s *SQLEventStore) Upcoming(tenantSlug string, now time.Time) []HouseEvent {
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
			`INSERT INTO events(tenant_slug, id, data) VALUES(?, ?, ?) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			textutil.Slug(item.TenantSlug), item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}
