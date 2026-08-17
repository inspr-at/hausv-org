package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

type ContactBookRepository interface {
	Upsert(item ManagedContact) (ManagedContact, bool, error)
	Deactivate(id string, at time.Time) (ManagedContact, error)
	List(includeInactive bool) []ManagedContact
}

type ContactBookStorage interface {
	contactBookStorage()
}

var (
	_ ContactBookStorage = (*ContactBookStore)(nil)
	_ ContactBookStorage = (*SQLContactBookStore)(nil)
)

type boundContactBookRepository struct {
	storage    contactBookBackend
	tenantSlug string
}

type contactBookBackend interface {
	upsert(tenantSlug string, item ManagedContact) (ManagedContact, bool, error)
	deactivate(tenantSlug string, id string, at time.Time) (ManagedContact, error)
	list(tenantSlug string, includeInactive bool) []ManagedContact
}

func BindContactBookRepository(storage ContactBookStorage, tenantSlug string) (ContactBookRepository, bool) {
	tenantSlug = textutil.Slug(tenantSlug)
	backend, ok := storage.(contactBookBackend)
	if !ok || tenantSlug == "" {
		return nil, false
	}
	return &boundContactBookRepository{storage: backend, tenantSlug: tenantSlug}, true
}

func (r *boundContactBookRepository) Upsert(item ManagedContact) (ManagedContact, bool, error) {
	return r.storage.upsert(r.tenantSlug, item)
}
func (r *boundContactBookRepository) Deactivate(id string, at time.Time) (ManagedContact, error) {
	return r.storage.deactivate(r.tenantSlug, id, at)
}
func (r *boundContactBookRepository) List(includeInactive bool) []ManagedContact {
	return r.storage.list(r.tenantSlug, includeInactive)
}

// SQLContactBookStore keeps each contact as a JSON document plus the columns it
// is filtered by (tenant, active). Read-modify-write (upsert preserving
// CreatedAt, deactivate) runs in a transaction — the atomicity the JSON store
// approximates with a mutex, now real. Table from migration 0006.
type SQLContactBookStore struct {
	db *sql.DB
}

func NewSQLContactBookStore(db *sql.DB) *SQLContactBookStore {
	return &SQLContactBookStore{db: db}
}

func (*SQLContactBookStore) contactBookStorage() {}

func writeContactTx(tx *sql.Tx, item ManagedContact) error {
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	active := 0
	if item.Active {
		active = 1
	}
	_, err = tx.Exec(
		`INSERT INTO contacts(tenant_slug, id, active, data) VALUES($1, $2, $3, $4)
		 ON CONFLICT(tenant_slug, id) DO UPDATE SET active=excluded.active, data=excluded.data`,
		item.TenantSlug, item.ID, active, string(blob),
	)
	return err
}

func (s *SQLContactBookStore) upsert(tenantSlug string, item ManagedContact) (ManagedContact, bool, error) {
	if s == nil {
		return ManagedContact{}, false, fmt.Errorf("contact store not configured")
	}
	item.TenantSlug = tenantSlug
	item, err := NormalizeManagedContact(item)
	if err != nil {
		return ManagedContact{}, false, err
	}
	now := time.Now().UTC()
	tx, err := s.db.Begin()
	if err != nil {
		return ManagedContact{}, false, err
	}
	defer tx.Rollback()

	if item.ID != "" {
		var data string
		if err := tx.QueryRow(`SELECT data FROM contacts WHERE tenant_slug=$1 AND id=$2`, item.TenantSlug, item.ID).Scan(&data); err == nil {
			var existing ManagedContact
			if json.Unmarshal([]byte(data), &existing) == nil {
				item.CreatedAt = existing.CreatedAt
			}
			if item.CreatedAt.IsZero() {
				item.CreatedAt = now
			}
			item.UpdatedAt = now
			if err := writeContactTx(tx, item); err != nil {
				return ManagedContact{}, false, err
			}
			if err := tx.Commit(); err != nil {
				return ManagedContact{}, false, err
			}
			return item, false, nil
		}
	}

	if item.ID == "" {
		id, err := randomToken(10)
		if err != nil {
			return ManagedContact{}, false, err
		}
		item.ID = id
	}
	item.CreatedAt = now
	item.UpdatedAt = now
	if err := writeContactTx(tx, item); err != nil {
		return ManagedContact{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return ManagedContact{}, false, err
	}
	return item, true, nil
}

func (s *SQLContactBookStore) deactivate(tenantSlug string, id string, at time.Time) (ManagedContact, error) {
	if s == nil {
		return ManagedContact{}, fmt.Errorf("contact store not configured")
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return ManagedContact{}, fmt.Errorf("invalid contact")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}
	tx, err := s.db.Begin()
	if err != nil {
		return ManagedContact{}, err
	}
	defer tx.Rollback()
	var data string
	if err := tx.QueryRow(`SELECT data FROM contacts WHERE tenant_slug=$1 AND id=$2`, tenantSlug, id).Scan(&data); err != nil {
		return ManagedContact{}, nil // not found -> empty, matching the JSON store
	}
	var existing ManagedContact
	if err := json.Unmarshal([]byte(data), &existing); err != nil {
		return ManagedContact{}, nil
	}
	existing.Active = false
	existing.UpdatedAt = at
	if err := writeContactTx(tx, existing); err != nil {
		return ManagedContact{}, err
	}
	if err := tx.Commit(); err != nil {
		return ManagedContact{}, err
	}
	return existing, nil
}

func (s *SQLContactBookStore) list(tenantSlug string, includeInactive bool) []ManagedContact {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	query := `SELECT data FROM contacts WHERE tenant_slug = $1`
	if !includeInactive {
		query += ` AND active = 1`
	}
	rows, err := s.db.Query(query, tenantSlug)
	if err != nil {
		return []ManagedContact{}
	}
	defer rows.Close()
	out := []ManagedContact{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var item ManagedContact
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}
		normalized, err := NormalizeManagedContact(item)
		if err != nil || normalized.TenantSlug != tenantSlug {
			continue
		}
		if !includeInactive && !normalized.Active {
			continue
		}
		out = append(out, normalized)
	}
	SortManagedContacts(out)
	return out
}

// ImportContacts copies records from a JSON store, each only if absent
// (clobber-safe) (HAUSV-170).
func (s *SQLContactBookStore) ImportContacts(src *ContactBookStore) error {
	if src == nil {
		return nil
	}
	src.mu.Lock()
	snapshot := append([]ManagedContact(nil), src.data.Contacts...)
	src.mu.Unlock()
	for _, raw := range snapshot {
		item, err := NormalizeManagedContact(raw)
		if err != nil || item.ID == "" {
			continue
		}
		blob, err := json.Marshal(item)
		if err != nil {
			return err
		}
		active := 0
		if item.Active {
			active = 1
		}
		if _, err := s.db.Exec(
			`INSERT INTO contacts(tenant_slug, id, active, data) VALUES($1, $2, $3, $4) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			item.TenantSlug, item.ID, active, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}
