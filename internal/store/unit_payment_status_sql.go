package store

import (
	"database/sql"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// UnitPaymentStatusRepository is a payment-status store already bound to one
// tenant.
type UnitPaymentStatusRepository interface {
	Set(item UnitPaymentStatus) (UnitPaymentStatus, error)
	Get(unitID string) (UnitPaymentStatus, bool)
	List() []UnitPaymentStatus
}

// UnitPaymentStatusStorage is the unbound backend implemented by the JSON and
// SQLite stores. HTTP code receives only UnitPaymentStatusRepository.
type UnitPaymentStatusStorage interface {
	unitPaymentStatusStorage()
}

var (
	_ UnitPaymentStatusStorage = (*UnitPaymentStatusStore)(nil)
	_ UnitPaymentStatusStorage = (*SQLUnitPaymentStatusStore)(nil)
)

type boundUnitPaymentStatusRepository struct {
	storage unitPaymentStatusBackend
	tenant  TenantRef
}

type unitPaymentStatusBackend interface {
	set(tenant TenantRef, item UnitPaymentStatus) (UnitPaymentStatus, error)
	get(tenant TenantRef, unitID string) (UnitPaymentStatus, bool)
	listTenant(tenant TenantRef) []UnitPaymentStatus
}

// BindUnitPaymentStatusRepository binds all payment-status operations to one
// tenant.
func BindUnitPaymentStatusRepository(storage UnitPaymentStatusStorage, tenant TenantRef) (UnitPaymentStatusRepository, bool) {
	resolvedTenant, tenantOK := validTenantRef(tenant)
	backend, ok := storage.(unitPaymentStatusBackend)
	if !ok || !tenantOK {
		return nil, false
	}
	return &boundUnitPaymentStatusRepository{storage: backend, tenant: resolvedTenant}, true
}

func (r *boundUnitPaymentStatusRepository) Set(item UnitPaymentStatus) (UnitPaymentStatus, error) {
	return r.storage.set(r.tenant, item)
}

func (r *boundUnitPaymentStatusRepository) Get(unitID string) (UnitPaymentStatus, bool) {
	return r.storage.get(r.tenant, unitID)
}

func (r *boundUnitPaymentStatusRepository) List() []UnitPaymentStatus {
	return r.storage.listTenant(r.tenant)
}

// SQLUnitPaymentStatusStore is the SQLite-backed manual payment-status marker,
// one row per (tenant, unit). Table from migration 0005.
type SQLUnitPaymentStatusStore struct {
	db *TenantDB
}

func NewSQLUnitPaymentStatusStore(db *TenantDB) *SQLUnitPaymentStatusStore {
	return &SQLUnitPaymentStatusStore{db: db}
}

func (*SQLUnitPaymentStatusStore) unitPaymentStatusStorage() {}

func (s *SQLUnitPaymentStatusStore) set(tenant TenantRef, item UnitPaymentStatus) (UnitPaymentStatus, error) {
	tenantSlug := tenant.Slug
	if s == nil {
		return UnitPaymentStatus{}, nil
	}
	item.TenantSlug = tenantSlug
	item, err := NormalizeUnitPaymentRecord(item)
	if err != nil {
		return UnitPaymentStatus{}, err
	}
	item.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	if _, err := s.db.Unscoped(healOrphanReason).Exec(
		`INSERT INTO unit_payment_status(tenant_id, tenant_slug, unit_id, status, updated_at, updated_by)
		 VALUES($1, $2, $3, $4, $5, $6)
		 ON CONFLICT(tenant_slug, unit_id) DO UPDATE SET
		   status=excluded.status, updated_at=excluded.updated_at, updated_by=excluded.updated_by,
		   tenant_id=coalesce(unit_payment_status.tenant_id, excluded.tenant_id)`,
		tenant.ID, tenant.Slug, item.UnitID, item.Status, item.UpdatedAt.Format(time.RFC3339Nano), item.UpdatedBy,
	); err != nil {
		return UnitPaymentStatus{}, err
	}
	return item, nil
}

func (s *SQLUnitPaymentStatusStore) get(tenant TenantRef, unitID string) (UnitPaymentStatus, bool) {
	tenantSlug := tenant.Slug
	if s == nil {
		return UnitPaymentStatus{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	unitID = NormalizeUnitID(unitID)
	if tenantSlug == "" || unitID == "" {
		return UnitPaymentStatus{}, false
	}
	item, ok := scanUnitPaymentRow(s.db.For(tenant).QueryRow(
		`SELECT tenant_slug, unit_id, status, updated_at, updated_by
		 FROM unit_payment_status WHERE tenant_id = $1 AND unit_id = $2`, tenant.ID, unitID))
	if !ok {
		return UnitPaymentStatus{}, false
	}
	normalized, err := NormalizeUnitPaymentRecord(item)
	if err != nil {
		return UnitPaymentStatus{}, false
	}
	return normalized, true
}

func (s *SQLUnitPaymentStatusStore) listTenant(tenant TenantRef) []UnitPaymentStatus {
	tenantSlug := tenant.Slug
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	rows, err := s.db.For(tenant).Query(
		`SELECT tenant_slug, unit_id, status, updated_at, updated_by
		 FROM unit_payment_status WHERE tenant_id = $1`, tenant.ID)
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
	imports := s.db.Unscoped("boot import replay of the JSON payment-status snapshot: it spans every tenant and runs before the first request")
	tenants := newTenantIDCache(imports)
	for _, raw := range snapshot {
		item, err := NormalizeUnitPaymentRecord(raw)
		if err != nil {
			continue
		}
		tenant, err := tenants.ref(item.TenantSlug)
		if err != nil {
			return err
		}
		updatedAt := item.UpdatedAt.UTC().Format(time.RFC3339Nano)
		if item.UpdatedAt.IsZero() {
			updatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		}
		if _, err := imports.Exec(
			`INSERT INTO unit_payment_status(tenant_id, tenant_slug, unit_id, status, updated_at, updated_by)
			 VALUES($1, $2, $3, $4, $5, $6) ON CONFLICT(tenant_slug, unit_id) DO NOTHING`,
			tenant.ID, tenant.Slug, item.UnitID, item.Status, updatedAt, item.UpdatedBy,
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
