package store

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// HomeConnectorReading is the deliberately small, read-only subset of one
// Home Assistant sensor that the local connector may disclose to the portal.
// Attributes that are unrelated to energy never enter this model.
type HomeConnectorReading struct {
	Slug        string
	EntityID    string
	State       string
	DisplayName string
	Unit        string
	DeviceClass string
	StateClass  string
	LastUpdated time.Time
	ReceivedAt  time.Time
}

type HomeConnectorReadingStorage interface {
	Upsert(slug string, readings []HomeConnectorReading, receivedAt time.Time) error
	List(slug string) ([]HomeConnectorReading, error)
	Clear(slug string) error
}

type MemoryHomeConnectorReadingStore struct {
	mu    sync.Mutex
	items map[string]HomeConnectorReading
}

func NewMemoryHomeConnectorReadingStore() *MemoryHomeConnectorReadingStore {
	return &MemoryHomeConnectorReadingStore{items: map[string]HomeConnectorReading{}}
}

func (s *MemoryHomeConnectorReadingStore) Upsert(slug string, readings []HomeConnectorReading, receivedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	slug = textutil.Slug(slug)
	if slug == "" {
		return fmt.Errorf("home connector reading: slug required")
	}
	for _, reading := range readings {
		reading.Slug = slug
		reading.EntityID = strings.ToLower(strings.TrimSpace(reading.EntityID))
		reading.ReceivedAt = receivedAt.UTC()
		s.items[slug+"\x00"+reading.EntityID] = reading
	}
	return nil
}

func (s *MemoryHomeConnectorReadingStore) List(slug string) ([]HomeConnectorReading, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	slug = textutil.Slug(slug)
	out := []HomeConnectorReading{}
	for _, item := range s.items {
		if item.Slug == slug {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EntityID < out[j].EntityID })
	return out, nil
}

func (s *MemoryHomeConnectorReadingStore) Clear(slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	slug = textutil.Slug(slug)
	for key, item := range s.items {
		if item.Slug == slug {
			delete(s.items, key)
		}
	}
	return nil
}

type SQLHomeConnectorReadingStore struct{ db *TenantDB }

func NewSQLHomeConnectorReadingStore(db *TenantDB) *SQLHomeConnectorReadingStore {
	return &SQLHomeConnectorReadingStore{db: db}
}

func (s *SQLHomeConnectorReadingStore) Upsert(slug string, readings []HomeConnectorReading, receivedAt time.Time) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("home connector reading store unavailable")
	}
	slug = textutil.Slug(slug)
	if slug == "" {
		return fmt.Errorf("home connector reading: slug required")
	}
	// Readings arrive from a paired connector, so the house is activated and the
	// identity exists. The lookup stays on the registry; the write is the lane.
	registry := s.db.Unscoped(slugRegistryReason)
	tenant, err := tenantRefFor(registry, slug)
	if err != nil {
		return err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, reading := range readings {
		_, err = tx.Exec(`INSERT INTO home_connector_readings
			(tenant_id,slug,entity_id,state,display_name,unit,device_class,state_class,last_updated,received_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			ON CONFLICT(slug,entity_id) DO UPDATE SET state=excluded.state,
			display_name=excluded.display_name,unit=excluded.unit,device_class=excluded.device_class,
			state_class=excluded.state_class,last_updated=excluded.last_updated,received_at=excluded.received_at,
			tenant_id=coalesce(home_connector_readings.tenant_id,excluded.tenant_id)`,
			tenant.ID, slug, strings.ToLower(strings.TrimSpace(reading.EntityID)), reading.State, reading.DisplayName,
			reading.Unit, reading.DeviceClass, reading.StateClass,
			homeReservationTimestamp(reading.LastUpdated.UTC()), homeReservationTimestamp(receivedAt.UTC()))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLHomeConnectorReadingStore) List(slug string) ([]HomeConnectorReading, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("home connector reading store unavailable")
	}
	slug = textutil.Slug(slug)
	rows, err := s.db.For(existingTenantRef(s.db, slug)).Query(`SELECT slug,entity_id,state,display_name,unit,device_class,state_class,last_updated,received_at
		FROM home_connector_readings WHERE slug=$1 ORDER BY entity_id`, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HomeConnectorReading{}
	for rows.Next() {
		var item HomeConnectorReading
		var updated, received string
		if err := rows.Scan(&item.Slug, &item.EntityID, &item.State, &item.DisplayName, &item.Unit,
			&item.DeviceClass, &item.StateClass, &updated, &received); err != nil {
			return nil, err
		}
		item.LastUpdated = parseHomeReservationTimestamp(updated)
		item.ReceivedAt = parseHomeReservationTimestamp(received)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *SQLHomeConnectorReadingStore) Clear(slug string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("home connector reading store unavailable")
	}
	slug = textutil.Slug(slug)
	_, err := s.db.For(existingTenantRef(s.db, slug)).Exec(`DELETE FROM home_connector_readings WHERE slug=$1`, slug)
	return err
}
