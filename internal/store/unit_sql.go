package store

import (
	"database/sql"
	"encoding/json"
	"sort"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// UnitRepository is a unit store already bound to one tenant.
type UnitRepository interface {
	SetUnits(units []Unit) error
	UpsertUnit(origID string, item Unit) (duplicate bool, err error)
	DeleteUnit(id string) (removed bool, removedUnit Unit, err error)
	List() []Unit
	UnitCount() int
	BillableUnitWeight() int
	UnitsForEmail(email string) []UnitMembership
	MembersForUnit(unitID string) UnitMembers
}

// UnitStorage is the unbound backend implemented by the JSON and SQLite
// stores. HTTP code receives only UnitRepository.
type UnitStorage interface {
	unitStorage()
}

var (
	_ UnitStorage = (*UnitStore)(nil)
	_ UnitStorage = (*SQLUnitStore)(nil)
)

type boundUnitRepository struct {
	storage    unitBackend
	tenantSlug string
}

type unitBackend interface {
	setTenantUnits(tenantSlug string, units []Unit) error
	upsertUnit(tenantSlug string, origID string, item Unit) (duplicate bool, err error)
	deleteUnit(tenantSlug string, id string) (removed bool, removedUnit Unit, err error)
	listTenant(tenantSlug string) []Unit
	unitCount(tenantSlug string) int
	billableUnitWeight(tenantSlug string) int
	unitsForEmail(tenantSlug string, email string) []UnitMembership
	membersForUnit(tenantSlug string, unitID string) UnitMembers
}

// BindUnitRepository binds all unit operations to one tenant.
func BindUnitRepository(storage UnitStorage, tenantSlug string) (UnitRepository, bool) {
	tenantSlug = textutil.Slug(tenantSlug)
	backend, ok := storage.(unitBackend)
	if !ok || tenantSlug == "" {
		return nil, false
	}
	return &boundUnitRepository{storage: backend, tenantSlug: tenantSlug}, true
}

func (r *boundUnitRepository) SetUnits(units []Unit) error {
	return r.storage.setTenantUnits(r.tenantSlug, units)
}

func (r *boundUnitRepository) UpsertUnit(origID string, item Unit) (bool, error) {
	return r.storage.upsertUnit(r.tenantSlug, origID, item)
}

func (r *boundUnitRepository) DeleteUnit(id string) (bool, Unit, error) {
	return r.storage.deleteUnit(r.tenantSlug, id)
}

func (r *boundUnitRepository) List() []Unit {
	return r.storage.listTenant(r.tenantSlug)
}

func (r *boundUnitRepository) UnitCount() int {
	return r.storage.unitCount(r.tenantSlug)
}

func (r *boundUnitRepository) BillableUnitWeight() int {
	return r.storage.billableUnitWeight(r.tenantSlug)
}

func (r *boundUnitRepository) UnitsForEmail(email string) []UnitMembership {
	return r.storage.unitsForEmail(r.tenantSlug, email)
}

func (r *boundUnitRepository) MembersForUnit(unitID string) UnitMembers {
	return r.storage.membersForUnit(r.tenantSlug, unitID)
}

// SQLUnitStore keeps each unit as a JSON document keyed by (tenant, id). Table
// from migration 0014.
type SQLUnitStore struct {
	db *sql.DB
}

func NewSQLUnitStore(db *sql.DB) *SQLUnitStore {
	return &SQLUnitStore{db: db}
}

func (*SQLUnitStore) unitStorage() {}

// replaceTenantTx rewrites a tenant's whole unit set inside a transaction. The
// JSON store re-normalizes the full tenant slice on every write (which also
// deduplicates), so mirroring that keeps the two backends byte-identical.
func (s *SQLUnitStore) replaceTenantTx(tx *sql.Tx, tenantSlug string, units []Unit) error {
	if _, err := tx.Exec(`DELETE FROM units WHERE tenant_slug=$1`, tenantSlug); err != nil {
		return err
	}
	for _, item := range NormalizeUnits(units, tenantSlug) {
		blob, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(
			`INSERT INTO units(tenant_slug, id, data) VALUES($1, $2, $3)
			 ON CONFLICT(tenant_slug, id) DO UPDATE SET data=excluded.data`,
			textutil.Slug(item.TenantSlug), item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLUnitStore) tenantUnits(tenantSlug string) []Unit {
	rows, err := s.db.Query(`SELECT data FROM units WHERE tenant_slug=$1`, tenantSlug)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []Unit{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var item Unit
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}
		out = append(out, item)
	}
	SortUnits(out)
	return out
}

func tenantUnitsTx(tx *sql.Tx, tenantSlug string) ([]Unit, error) {
	rows, err := tx.Query(`SELECT data FROM units WHERE tenant_slug=$1`, tenantSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Unit{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var item Unit
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}
		out = append(out, item)
	}
	SortUnits(out)
	return out, nil
}

func (s *SQLUnitStore) setTenantUnits(tenantSlug string, units []Unit) error {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	if tenantSlug == "" {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.replaceTenantTx(tx, tenantSlug, units); err != nil {
		return err
	}
	return tx.Commit()
}

// UpsertUnit adds or replaces a single unit in one transaction, so a concurrent
// add/delete of a different unit is not clobbered (HAUSV-145). origID is the
// unit's previous ID ("" for a new unit); duplicate=true means the target ID
// collides with a different existing unit.
func (s *SQLUnitStore) upsertUnit(tenantSlug string, origID string, item Unit) (bool, error) {
	if s == nil {
		return false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	if tenantSlug == "" {
		return false, nil
	}
	item.TenantSlug = tenantSlug
	wasCreate := origID == ""

	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	mine, err := tenantUnitsTx(tx, tenantSlug)
	if err != nil {
		return false, err
	}
	for _, u := range mine {
		if u.ID == item.ID && (wasCreate || origID != item.ID) {
			return true, nil
		}
	}
	if wasCreate {
		origID = item.ID
	}
	replaced := false
	for i := range mine {
		if mine[i].ID == origID {
			mine[i] = item
			replaced = true
			break
		}
	}
	if !replaced {
		mine = append(mine, item)
	}
	if err := s.replaceTenantTx(tx, tenantSlug, mine); err != nil {
		return false, err
	}
	return false, tx.Commit()
}

// DeleteUnit removes one unit. Returns removed=false if no unit had that ID.
func (s *SQLUnitStore) deleteUnit(tenantSlug string, id string) (bool, Unit, error) {
	if s == nil {
		return false, Unit{}, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	if tenantSlug == "" {
		return false, Unit{}, nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return false, Unit{}, err
	}
	defer tx.Rollback()
	var data string
	if err := tx.QueryRow(`SELECT data FROM units WHERE tenant_slug=$1 AND id=$2`, tenantSlug, id).Scan(&data); err != nil {
		return false, Unit{}, nil
	}
	var removed Unit
	if err := json.Unmarshal([]byte(data), &removed); err != nil {
		return false, Unit{}, nil
	}
	if _, err := tx.Exec(`DELETE FROM units WHERE tenant_slug=$1 AND id=$2`, tenantSlug, id); err != nil {
		return false, Unit{}, err
	}
	if err := tx.Commit(); err != nil {
		return false, Unit{}, err
	}
	return true, removed, nil
}

func (s *SQLUnitStore) listTenant(tenantSlug string) []Unit {
	if s == nil {
		return nil
	}
	units := s.tenantUnits(textutil.Slug(tenantSlug))
	out := []Unit{}
	for _, item := range units {
		out = append(out, CopyUnit(item))
	}
	SortUnits(out)
	return out
}

func (s *SQLUnitStore) unitCount(tenantSlug string) int {
	return len(s.listTenant(tenantSlug))
}

func (s *SQLUnitStore) billableUnitWeight(tenantSlug string) int {
	return BillableUnitWeight(s.listTenant(tenantSlug))
}

func (s *SQLUnitStore) unitsForEmail(tenantSlug string, email string) []UnitMembership {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	email = textutil.Email(email)
	if tenantSlug == "" || email == "" {
		return nil
	}
	out := []UnitMembership{}
	for _, item := range s.tenantUnits(tenantSlug) {
		relation := ""
		if EmailListContains(item.OwnerEmails, email) {
			relation = RoleOwner
		} else if EmailListContains(item.RenterEmails, email) {
			relation = RoleRenter
		}
		if relation != "" {
			out = append(out, UnitMembership{Unit: CopyUnit(item), Relation: relation})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return UnitLess(out[i].Unit, out[j].Unit)
	})
	return out
}

func (s *SQLUnitStore) membersForUnit(tenantSlug string, unitID string) UnitMembers {
	if s == nil {
		return UnitMembers{}
	}
	tenantSlug = textutil.Slug(tenantSlug)
	unitID = NormalizeUnitID(unitID)
	if tenantSlug == "" || unitID == "" {
		return UnitMembers{}
	}
	for _, item := range s.tenantUnits(tenantSlug) {
		if textutil.Slug(item.ID) == unitID {
			item = CopyUnit(item)
			return UnitMembers{Unit: item, Owners: append([]string(nil), item.OwnerEmails...), Renters: append([]string(nil), item.RenterEmails...), Found: true}
		}
	}
	return UnitMembers{}
}

// ImportUnits copies records from a JSON store, each only if absent
// (clobber-safe) (HAUSV-170).
func (s *SQLUnitStore) ImportUnits(src *UnitStore) error {
	if src == nil {
		return nil
	}
	src.mu.Lock()
	snapshot := append([]Unit(nil), src.data.Units...)
	src.mu.Unlock()
	for _, item := range snapshot {
		tenant := textutil.Slug(item.TenantSlug)
		if tenant == "" || item.ID == "" {
			continue
		}
		blob, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(
			`INSERT INTO units(tenant_slug, id, data) VALUES($1, $2, $3) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			tenant, item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}
