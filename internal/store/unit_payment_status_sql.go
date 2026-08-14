package store

import (
	"database/sql"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// UnitPaymentStatusStorage is the behaviour both the JSON UnitPaymentStatusStore
// and the SQLite SQLUnitPaymentStatusStore satisfy (HAUSV-168). The key is the
// (tenant, unit) pair.
type UnitPaymentStatusStorage interface {
	Set(item UnitPaymentStatus) (UnitPaymentStatus, error)
	Get(tenantSlug string, unitID string) (UnitPaymentStatus, bool)
	ListTenant(tenantSlug string) []UnitPaymentStatus
}

var (
	_ UnitPaymentStatusStorage = (*UnitPaymentStatusStore)(nil)
	_ UnitPaymentStatusStorage = (*SQLUnitPaymentStatusStore)(nil)
)

// SQLUnitPaymentStatusStore is the SQLite-backed manual payment-status marker,
// one row per (tenant, unit). Table from migration 0005.
type SQLUnitPaymentStatusStore struct {
	db *sql.DB
}

func NewSQLUnitPaymentStatusStore(db *sql.DB) *SQLUnitPaymentStatusStore {
	return &SQLUnitPaymentStatusStore{db: db}
}

func (s *SQLUnitPaymentStatusStore) Set(item UnitPaymentStatus) (UnitPaymentStatus, error) {
	if s == nil {
		return UnitPaymentStatus{}, nil
	}
	item, err := NormalizeUnitPaymentRecord(item)
	if err != nil {
		return UnitPaymentStatus{}, err
	}
	item.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	if _, err := s.db.Exec(
		`INSERT INTO unit_payment_status(tenant_slug, unit_id, status, updated_at, updated_by)
		 VALUES(?, ?, ?, ?, ?)
		 ON CONFLICT(tenant_slug, unit_id) DO UPDATE SET
		   status=excluded.status, updated_at=excluded.updated_at, updated_by=excluded.updated_by`,
		item.TenantSlug, item.UnitID, item.Status, item.UpdatedAt.Format(time.RFC3339Nano), item.UpdatedBy,
	); err != nil {
		return UnitPaymentStatus{}, err
	}
	return item, nil
}

func (s *SQLUnitPaymentStatusStore) Get(tenantSlug string, unitID string) (UnitPaymentStatus, bool) {
	if s == nil {
		return UnitPaymentStatus{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	unitID = NormalizeUnitID(unitID)
	if tenantSlug == "" || unitID == "" {
		return UnitPaymentStatus{}, false
	}
	item, ok := scanUnitPaymentRow(s.db.QueryRow(
		`SELECT tenant_slug, unit_id, status, updated_at, updated_by
		 FROM unit_payment_status WHERE tenant_slug = ? AND unit_id = ?`, tenantSlug, unitID))
	if !ok {
		return UnitPaymentStatus{}, false
	}
	normalized, err := NormalizeUnitPaymentRecord(item)
	if err != nil {
		return UnitPaymentStatus{}, false
	}
	return normalized, true
}

func (s *SQLUnitPaymentStatusStore) ListTenant(tenantSlug string) []UnitPaymentStatus {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	rows, err := s.db.Query(
		`SELECT tenant_slug, unit_id, status, updated_at, updated_by
		 FROM unit_payment_status WHERE tenant_slug = ?`, tenantSlug)
	if err != nil {
		return []UnitPaymentStatus{}
	}
	defer rows.Close()
	out := []UnitPaymentStatus{}
	for rows.Next() {
		item, ok := scanUnitPaymentRows(rows)
		if !ok {
			continue
		}
		normalized, err := NormalizeUnitPaymentRecord(item)
		if err != nil || normalized.TenantSlug != tenantSlug {
			continue
		}
		out = append(out, normalized)
	}
	SortUnitPaymentStatuses(out) // identical ordering to the JSON store
	return out
}

// ImportStatuses copies records from a JSON store, each only if absent
// (clobber-safe), preserving UpdatedAt/UpdatedBy (HAUSV-170).
func (s *SQLUnitPaymentStatusStore) ImportStatuses(src *UnitPaymentStatusStore) error {
	if src == nil {
		return nil
	}
	src.mu.Lock()
	snapshot := append([]UnitPaymentStatus(nil), src.data.Statuses...)
	src.mu.Unlock()
	for _, raw := range snapshot {
		item, err := NormalizeUnitPaymentRecord(raw)
		if err != nil {
			continue
		}
		updatedAt := item.UpdatedAt.UTC().Format(time.RFC3339Nano)
		if item.UpdatedAt.IsZero() {
			updatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		}
		if _, err := s.db.Exec(
			`INSERT INTO unit_payment_status(tenant_slug, unit_id, status, updated_at, updated_by)
			 VALUES(?, ?, ?, ?, ?) ON CONFLICT(tenant_slug, unit_id) DO NOTHING`,
			item.TenantSlug, item.UnitID, item.Status, updatedAt, item.UpdatedBy,
		); err != nil {
			return err
		}
	}
	return nil
}

type unitPaymentScanner interface {
	Scan(dest ...any) error
}

func scanUnitPaymentRow(row *sql.Row) (UnitPaymentStatus, bool)    { return scanUnitPayment(row) }
func scanUnitPaymentRows(rows *sql.Rows) (UnitPaymentStatus, bool) { return scanUnitPayment(rows) }

func scanUnitPayment(sc unitPaymentScanner) (UnitPaymentStatus, bool) {
	var (
		item      UnitPaymentStatus
		updatedAt string
	)
	if err := sc.Scan(&item.TenantSlug, &item.UnitID, &item.Status, &updatedAt, &item.UpdatedBy); err != nil {
		return UnitPaymentStatus{}, false
	}
	if t, err := time.Parse(time.RFC3339Nano, updatedAt); err == nil {
		item.UpdatedAt = t.UTC()
	}
	return item, true
}
