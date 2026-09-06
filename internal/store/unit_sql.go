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
	UpdateParties(updates []UnitPartyUpdate) (unknownUnit bool, err error)
	UpdateAllocationBases(updates []UnitAllocationBasisUpdate) (unknownUnit bool, err error)
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
	storage unitBackend
	tenant  TenantRef
}

type unitBackend interface {
	setTenantUnits(tenant TenantRef, units []Unit) error
	upsertUnit(tenant TenantRef, origID string, item Unit) (duplicate bool, err error)
	updateUnitParties(tenant TenantRef, updates []UnitPartyUpdate) (unknownUnit bool, err error)
	updateUnitAllocationBases(tenant TenantRef, updates []UnitAllocationBasisUpdate) (unknownUnit bool, err error)
	deleteUnit(tenant TenantRef, id string) (removed bool, removedUnit Unit, err error)
	listTenant(tenant TenantRef) []Unit
	unitCount(tenant TenantRef) int
	billableUnitWeight(tenant TenantRef) int
	unitsForEmail(tenant TenantRef, email string) []UnitMembership
	membersForUnit(tenant TenantRef, unitID string) UnitMembers
}

// BindUnitRepository binds all unit operations to one tenant.
func BindUnitRepository(storage UnitStorage, tenant TenantRef) (UnitRepository, bool) {
	resolvedTenant, tenantOK := validTenantRef(tenant)
	backend, ok := storage.(unitBackend)
	if !ok || !tenantOK {
		return nil, false
	}
	return &boundUnitRepository{storage: backend, tenant: resolvedTenant}, true
}

func (r *boundUnitRepository) SetUnits(units []Unit) error {
	return r.storage.setTenantUnits(r.tenant, units)
}

func (r *boundUnitRepository) UpsertUnit(origID string, item Unit) (bool, error) {
	return r.storage.upsertUnit(r.tenant, origID, item)
}

func (r *boundUnitRepository) UpdateParties(updates []UnitPartyUpdate) (bool, error) {
	return r.storage.updateUnitParties(r.tenant, updates)
}

func (r *boundUnitRepository) UpdateAllocationBases(updates []UnitAllocationBasisUpdate) (bool, error) {
	return r.storage.updateUnitAllocationBases(r.tenant, updates)
}

func (r *boundUnitRepository) DeleteUnit(id string) (bool, Unit, error) {
	return r.storage.deleteUnit(r.tenant, id)
}

func (r *boundUnitRepository) List() []Unit {
	return r.storage.listTenant(r.tenant)
}

func (r *boundUnitRepository) UnitCount() int {
	return r.storage.unitCount(r.tenant)
}

func (r *boundUnitRepository) BillableUnitWeight() int {
	return r.storage.billableUnitWeight(r.tenant)
}

func (r *boundUnitRepository) UnitsForEmail(email string) []UnitMembership {
	return r.storage.unitsForEmail(r.tenant, email)
}

func (r *boundUnitRepository) MembersForUnit(unitID string) UnitMembers {
	return r.storage.membersForUnit(r.tenant, unitID)
}

// SQLUnitStore keeps each unit as a JSON document keyed by (tenant, id). Table
// from migration 0014.
type SQLUnitStore struct {
	db *TenantDB
}

func NewSQLUnitStore(db *TenantDB) *SQLUnitStore {
	return &SQLUnitStore{db: db}
}

func (*SQLUnitStore) unitStorage() {}

// replaceTenantTx rewrites a tenant's whole unit set inside a transaction. The
// JSON store re-normalizes the full tenant slice on every write (which also
// deduplicates), so mirroring that keeps the two backends byte-identical.
func (s *SQLUnitStore) replaceTenantTx(tx *sql.Tx, tenant TenantRef, units []Unit) error {
	// A whole-tenant wipe: getting this predicate wrong destroys a house's unit
	// set, which is why it is keyed on the identity rather than the label.
	if _, err := tx.Exec(`DELETE FROM units WHERE tenant_id=$1`, tenant.ID); err != nil {
		return err
	}
	for _, item := range NormalizeUnits(units, tenant.Slug) {
		blob, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(
			`INSERT INTO units(tenant_id, tenant_slug, id, data) VALUES($1, $2, $3, $4)
			 ON CONFLICT(tenant_slug, id) DO UPDATE SET data=excluded.data,
			   tenant_id=coalesce(units.tenant_id, excluded.tenant_id)`,
			tenant.ID, tenant.Slug, item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLUnitStore) tenantUnits(tenant TenantRef) []Unit {
	rows, err := s.db.For(tenant).Query(`SELECT data FROM units WHERE tenant_id=$1`, tenant.ID)
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

func tenantUnitsTx(tx *sql.Tx, tenant TenantRef) ([]Unit, error) {
	rows, err := tx.Query(`SELECT data FROM units WHERE tenant_id=$1`, tenant.ID)
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

func (s *SQLUnitStore) setTenantUnits(tenant TenantRef, units []Unit) error {
	tenantSlug := tenant.Slug
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	if tenantSlug == "" {
		return nil
	}
	// The whole transaction, not just one statement: replaceTenantTx upserts by
	// (tenant_slug, id), which is exactly where a legacy row without an identity
	// sits. See HealOrphanReason.
	tx, err := s.db.Unscoped(HealOrphanReason).Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.replaceTenantTx(tx, tenant, units); err != nil {
		return err
	}
	return tx.Commit()
}

// UpsertUnit adds or replaces a single unit in one transaction, so a concurrent
// add/delete of a different unit is not clobbered (HAUSV-145). origID is the
// unit's previous ID ("" for a new unit); duplicate=true means the target ID
// collides with a different existing unit.
func (s *SQLUnitStore) upsertUnit(tenant TenantRef, origID string, item Unit) (bool, error) {
	tenantSlug := tenant.Slug
	if s == nil {
		return false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	if tenantSlug == "" {
		return false, nil
	}
	item.TenantSlug = tenantSlug
	wasCreate := origID == ""

	// The whole transaction, not just one statement: replaceTenantTx upserts by
	// (tenant_slug, id), which is exactly where a legacy row without an identity
	// sits. See HealOrphanReason.
	tx, err := s.db.Unscoped(HealOrphanReason).Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	mine, err := tenantUnitsTx(tx, tenant)
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
	if err := s.replaceTenantTx(tx, tenant, mine); err != nil {
		return false, err
	}
	return false, tx.Commit()
}

func (s *SQLUnitStore) updateUnitParties(tenant TenantRef, updates []UnitPartyUpdate) (bool, error) {
	if s == nil || len(updates) == 0 {
		return false, nil
	}
	normalized := normalizeUnitPartyUpdates(updates)
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	units, err := tenantUnitsTx(tx, tenant)
	if err != nil {
		return false, err
	}
	indexes := map[string]int{}
	for index, item := range units {
		indexes[NormalizeUnitID(item.ID)] = index
	}
	for id := range normalized {
		if _, found := indexes[id]; !found {
			return true, nil
		}
	}
	for id, update := range normalized {
		item := units[indexes[id]]
		if update.SetOwners {
			item.OwnerEmails = NormalizeEmailList(update.OwnerEmails)
		}
		if update.SetRenters {
			item.RenterEmails = NormalizeEmailList(update.RenterEmails)
		}
		item.PartyContacts = mergeUnitPartyContacts(item, update.Contacts)
		blob, err := json.Marshal(item)
		if err != nil {
			return false, err
		}
		if _, err := tx.Exec(`UPDATE units SET data=$1 WHERE tenant_id=$2 AND id=$3`, string(blob), tenant.ID, item.ID); err != nil {
			return false, err
		}
	}
	return false, tx.Commit()
}

func (s *SQLUnitStore) updateUnitAllocationBases(tenant TenantRef, updates []UnitAllocationBasisUpdate) (bool, error) {
	if s == nil || len(updates) == 0 {
		return false, nil
	}
	normalized, malformed := normalizeUnitAllocationBasisUpdates(updates)
	if malformed {
		return true, nil
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	units, err := tenantUnitsTx(tx, tenant)
	if err != nil {
		return false, err
	}
	indexes := map[string]int{}
	for index, item := range units {
		indexes[NormalizeUnitID(item.ID)] = index
	}
	for id := range normalized {
		if _, found := indexes[id]; !found {
			return true, nil
		}
	}
	for id, update := range normalized {
		item := units[indexes[id]]
		item.UsableAreaM2Hundredths = update.UsableAreaM2Hundredths
		item.UsableAreaRecorded = update.UsableAreaRecorded
		item.Persons = update.Persons
		item.PersonsRecorded = update.PersonsRecorded
		blob, err := json.Marshal(item)
		if err != nil {
			return false, err
		}
		if _, err := tx.Exec(`UPDATE units SET data=$1 WHERE tenant_id=$2 AND id=$3`, string(blob), tenant.ID, item.ID); err != nil {
			return false, err
		}
	}
	return false, tx.Commit()
}

// DeleteUnit removes one unit. Returns removed=false if no unit had that ID.
func (s *SQLUnitStore) deleteUnit(tenant TenantRef, id string) (bool, Unit, error) {
	tenantSlug := tenant.Slug
	if s == nil {
		return false, Unit{}, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	if tenantSlug == "" {
		return false, Unit{}, nil
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return false, Unit{}, err
	}
	defer tx.Rollback()
	var data string
	if err := tx.QueryRow(`SELECT data FROM units WHERE tenant_id=$1 AND id=$2`, tenant.ID, id).Scan(&data); err != nil {
		return false, Unit{}, nil
	}
	var removed Unit
	if err := json.Unmarshal([]byte(data), &removed); err != nil {
		return false, Unit{}, nil
	}
	if _, err := tx.Exec(`DELETE FROM units WHERE tenant_id=$1 AND id=$2`, tenant.ID, id); err != nil {
		return false, Unit{}, err
	}
	if err := tx.Commit(); err != nil {
		return false, Unit{}, err
	}
	return true, removed, nil
}

func (s *SQLUnitStore) listTenant(tenant TenantRef) []Unit {
	if s == nil {
		return nil
	}
	units := s.tenantUnits(tenant)
	out := []Unit{}
	for _, item := range units {
		out = append(out, CopyUnit(item))
	}
	SortUnits(out)
	return out
}

func (s *SQLUnitStore) unitCount(tenant TenantRef) int {
	return len(s.listTenant(tenant))
}

func (s *SQLUnitStore) billableUnitWeight(tenant TenantRef) int {
	return BillableUnitWeight(s.listTenant(tenant))
}

func (s *SQLUnitStore) unitsForEmail(tenant TenantRef, email string) []UnitMembership {
	tenantSlug := tenant.Slug
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	email = textutil.Email(email)
	if tenantSlug == "" || email == "" {
		return nil
	}
	out := []UnitMembership{}
	for _, item := range s.tenantUnits(tenant) {
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

func (s *SQLUnitStore) membersForUnit(tenant TenantRef, unitID string) UnitMembers {
	tenantSlug := tenant.Slug
	if s == nil {
		return UnitMembers{}
	}
	tenantSlug = textutil.Slug(tenantSlug)
	unitID = NormalizeUnitID(unitID)
	if tenantSlug == "" || unitID == "" {
		return UnitMembers{}
	}
	for _, item := range s.tenantUnits(tenant) {
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
	imports := s.db.Unscoped("boot import replay of the JSON unit snapshot: it spans every tenant and runs before the first request")
	tenants := newTenantIDCache(imports)
	for _, item := range snapshot {
		slug := textutil.Slug(item.TenantSlug)
		if slug == "" || item.ID == "" {
			continue
		}
		tenant, err := tenants.ref(slug)
		if err != nil {
			return err
		}
		blob, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if _, err := imports.Exec(
			`INSERT INTO units(tenant_id, tenant_slug, id, data) VALUES($1, $2, $3, $4) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			tenant.ID, tenant.Slug, item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}
