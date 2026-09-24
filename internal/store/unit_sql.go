package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

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
	// ListChecked reports a row that could not be read. List stays empty in that
	// case so money paths remain fail-closed; pages use this variant.
	ListChecked() ([]Unit, error)
	UnitCount() int
	UnitCountChecked() (int, error)
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
	listTenantChecked(tenant TenantRef) ([]Unit, error)
	unitCount(tenant TenantRef) int
	unitCountChecked(tenant TenantRef) (int, error)
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
	for _, u := range units {
		if err := ValidateUnitPartyContacts(u.PartyContacts); err != nil {
			return err
		}
	}
	return r.storage.setTenantUnits(r.tenant, units)
}

func (r *boundUnitRepository) UpsertUnit(origID string, item Unit) (bool, error) {
	if err := ValidateUnitPartyContacts(item.PartyContacts); err != nil {
		return false, err
	}
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

func (r *boundUnitRepository) ListChecked() ([]Unit, error) {
	return r.storage.listTenantChecked(r.tenant)
}

func (r *boundUnitRepository) UnitCount() int {
	return r.storage.unitCount(r.tenant)
}

func (r *boundUnitRepository) UnitCountChecked() (int, error) {
	return r.storage.unitCountChecked(r.tenant)
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

// UnitDataError is a unit row that could not be scanned, decoded, or iterated.
// A mutation returns it and writes nothing, instead of continuing with the rows
// that happened to parse.
type UnitDataError struct {
	TenantID string
	UnitID   string
	Op       string
	Err      error
}

func (e *UnitDataError) Error() string {
	if e == nil {
		return "unit data"
	}
	if e.UnitID == "" {
		return fmt.Sprintf("unit data %s for tenant %s: %v", e.Op, e.TenantID, e.Err)
	}
	return fmt.Sprintf("unit data %s for tenant %s unit %s: %v", e.Op, e.TenantID, e.UnitID, e.Err)
}

func (e *UnitDataError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// unitDataSeen logs each corrupt row once per process. List and count poll the
// same row; the log should name it, not repeat it on every read.
var unitDataSeen sync.Map

func unitDataErr(tenant TenantRef, unitID, op string, err error) error {
	wrapped := &UnitDataError{TenantID: tenant.ID, UnitID: unitID, Op: op, Err: err}
	key := tenant.ID + "\x00" + unitID + "\x00" + op
	if _, loaded := unitDataSeen.LoadOrStore(key, struct{}{}); !loaded {
		slog.Error("unit row could not be read", "tenant_id", tenant.ID, "house", tenant.Slug, "unit_id", unitID, "op", op, "error", err)
	}
	return wrapped
}

type unitRows interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type storedUnit struct {
	id   string
	unit Unit
}

func readStoredUnits(q unitRows, tenant TenantRef) ([]storedUnit, error) {
	rows, err := q.Query(`SELECT id, data, party_validity FROM units WHERE tenant_id=$1`, tenant.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storedUnit{}
	for rows.Next() {
		var id, data, validity string
		if err := rows.Scan(&id, &data, &validity); err != nil {
			return nil, unitDataErr(tenant, id, "scan", err)
		}
		var item Unit
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			return nil, unitDataErr(tenant, id, "decode", err)
		}
		if err := decodeUnitValidity(&item, validity); err != nil {
			return nil, unitDataErr(tenant, id, "dates", err)
		}
		out = append(out, storedUnit{id: id, unit: item})
	}
	if err := rows.Err(); err != nil {
		return nil, unitDataErr(tenant, "", "iterate", err)
	}
	return out, nil
}

func loadStoredUnit(q unitRows, tenant TenantRef, id string) (Unit, error) {
	var data, validity string
	if err := q.QueryRow(`SELECT data, party_validity FROM units WHERE tenant_id=$1 AND id=$2`, tenant.ID, id).Scan(&data, &validity); err != nil {
		return Unit{}, err
	}
	var item Unit
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return Unit{}, unitDataErr(tenant, id, "decode", err)
	}
	if err := decodeUnitValidity(&item, validity); err != nil {
		return Unit{}, unitDataErr(tenant, id, "dates", err)
	}
	if item.ID == "" {
		item.ID = id
	}
	return item, nil
}

func unitRowExists(tx *sql.Tx, tenant TenantRef, id string) (bool, error) {
	var one int
	err := tx.QueryRow(`SELECT 1 FROM units WHERE tenant_id=$1 AND id=$2`, tenant.ID, id).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func writeUnitRow(tx *sql.Tx, tenant TenantRef, item Unit) error {
	blob, validity, err := EncodeUnitWithValidity(item)
	if err != nil {
		return err
	}
	res, err := tx.Exec(`UPDATE units SET data=$1, party_validity=$4 WHERE tenant_id=$2 AND id=$3`, blob, tenant.ID, item.ID, validity)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = tx.Exec(
		`INSERT INTO units(tenant_id, tenant_slug, id, data, party_validity) VALUES($1, $2, $3, $4, $5)`,
		tenant.ID, tenant.Slug, item.ID, blob, validity,
	)
	return err
}

func (s *SQLUnitStore) setTenantUnits(tenant TenantRef, units []Unit) error {
	if s == nil {
		return nil
	}
	tenantSlug := textutil.Slug(tenant.Slug)
	if tenantSlug == "" {
		return nil
	}
	normalized := NormalizeUnits(units, tenantSlug)
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Lock the rows this replacement will rewrite so a second batch waits and
	// then reads the committed set. A row that fails to decode aborts the
	// transaction, so the unreadable unit is not deleted.
	if _, err := tx.Exec(`UPDATE units SET data = data WHERE tenant_id = $1`, tenant.ID); err != nil {
		return err
	}
	existing, err := readStoredUnits(tx, tenant)
	if err != nil {
		return err
	}
	want := map[string]Unit{}
	for _, item := range normalized {
		want[item.ID] = item
	}
	for _, row := range existing {
		if _, keep := want[row.id]; keep {
			continue
		}
		if _, err := tx.Exec(`DELETE FROM units WHERE tenant_id=$1 AND id=$2`, tenant.ID, row.id); err != nil {
			return err
		}
	}
	for _, item := range normalized {
		if err := writeUnitRow(tx, tenant, item); err != nil {
			return err
		}
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
	prepared := NormalizeUnits([]Unit{item}, tenantSlug)
	if len(prepared) != 1 {
		return false, nil
	}
	item = prepared[0]
	wasCreate := origID == ""
	if origID != "" {
		origID = NormalizeUnitID(origID)
	}

	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	targetTaken, err := unitRowExists(tx, tenant, item.ID)
	if err != nil {
		return false, err
	}
	if targetTaken && (wasCreate || origID != item.ID) {
		return true, nil
	}
	if !wasCreate && origID != item.ID {
		if _, err := tx.Exec(`DELETE FROM units WHERE tenant_id=$1 AND id=$2`, tenant.ID, origID); err != nil {
			return false, err
		}
	}
	if err := writeUnitRow(tx, tenant, item); err != nil {
		return false, err
	}
	return false, tx.Commit()
}

func (s *SQLUnitStore) updateUnitParties(tenant TenantRef, updates []UnitPartyUpdate) (bool, error) {
	if s == nil || len(updates) == 0 {
		return false, nil
	}
	for _, update := range updates {
		if err := ValidateUnitPartyContacts(update.Contacts); err != nil {
			return false, err
		}
	}
	normalized := normalizeUnitPartyUpdates(updates)
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	loaded, unknown, err := loadUnitsForUpdate(tx, tenant, keysOf(normalized))
	if err != nil || unknown {
		return unknown, err
	}
	for id, update := range normalized {
		item := loaded[id]
		if update.SetOwners {
			item.OwnerEmails = NormalizeEmailList(update.OwnerEmails)
		}
		if update.SetRenters {
			item.RenterEmails = NormalizeEmailList(update.RenterEmails)
		}
		item.PartyContacts = mergeUnitPartyContacts(item, update.Contacts)
		if err := writeUnitRow(tx, tenant, item); err != nil {
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
	loaded, unknown, err := loadUnitsForUpdate(tx, tenant, keysOf(normalized))
	if err != nil || unknown {
		return unknown, err
	}
	for id, update := range normalized {
		item := loaded[id]
		item.UsableAreaM2Hundredths = update.UsableAreaM2Hundredths
		item.UsableAreaRecorded = update.UsableAreaRecorded
		item.Persons = update.Persons
		item.PersonsRecorded = update.PersonsRecorded
		if err := writeUnitRow(tx, tenant, item); err != nil {
			return false, err
		}
	}
	return false, tx.Commit()
}

func keysOf[V any](items map[string]V) []string {
	out := make([]string, 0, len(items))
	for id := range items {
		out = append(out, id)
	}
	return out
}

// loadUnitsForUpdate reads only the rows a narrow update will change. A missing
// id is unknown and writes nothing. A row that does not decode aborts the
// mutation even when another id in the same batch is missing.
func loadUnitsForUpdate(tx *sql.Tx, tenant TenantRef, ids []string) (map[string]Unit, bool, error) {
	loaded := make(map[string]Unit, len(ids))
	unknown := false
	var dataErr error
	for _, id := range ids {
		item, err := loadStoredUnit(tx, tenant, id)
		if errors.Is(err, sql.ErrNoRows) {
			unknown = true
			continue
		}
		if err != nil {
			dataErr = err
			continue
		}
		item.ID = id
		loaded[id] = item
	}
	if dataErr != nil {
		return nil, false, dataErr
	}
	if unknown {
		return nil, true, nil
	}
	return loaded, false, nil
}

// DeleteUnit removes one unit. Returns removed=false if no unit had that ID.
func (s *SQLUnitStore) deleteUnit(tenant TenantRef, id string) (bool, Unit, error) {
	if s == nil {
		return false, Unit{}, nil
	}
	if textutil.Slug(tenant.Slug) == "" {
		return false, Unit{}, nil
	}
	id = NormalizeUnitID(id)
	if id == "" {
		return false, Unit{}, nil
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return false, Unit{}, err
	}
	defer tx.Rollback()
	removed, err := loadStoredUnit(tx, tenant, id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, Unit{}, nil
	}
	if err != nil {
		return false, Unit{}, err
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
	units, err := s.listTenantChecked(tenant)
	if err != nil {
		return nil
	}
	return units
}

func (s *SQLUnitStore) listTenantChecked(tenant TenantRef) ([]Unit, error) {
	if s == nil {
		return nil, nil
	}
	stored, err := readStoredUnits(s.db.For(tenant), tenant)
	if err != nil {
		return nil, err
	}
	out := make([]Unit, 0, len(stored))
	for _, row := range stored {
		out = append(out, CopyUnit(row.unit))
	}
	SortUnits(out)
	return out, nil
}

func (s *SQLUnitStore) unitCount(tenant TenantRef) int {
	count, err := s.unitCountChecked(tenant)
	if err != nil {
		return 0
	}
	return count
}

func (s *SQLUnitStore) unitCountChecked(tenant TenantRef) (int, error) {
	units, err := s.listTenantChecked(tenant)
	if err != nil {
		return 0, err
	}
	return len(units), nil
}

func (s *SQLUnitStore) billableUnitWeight(tenant TenantRef) int {
	// Fail closed: an unreadable row is logged once with the house and counts as
	// no billable weight, never as a partial house.
	units, err := s.listTenantChecked(tenant)
	if err != nil {
		return 0
	}
	return BillableUnitWeight(units)
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
	stored, err := readStoredUnits(s.db.For(tenant), tenant)
	if err != nil {
		return nil
	}
	out := []UnitMembership{}
	for _, row := range stored {
		item := row.unit
		relation := ""
		if !UnitPartyActive(item, email, time.Now()) {
			continue
		}
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
	item, err := loadStoredUnit(s.db.For(tenant), tenant, unitID)
	if err != nil || textutil.Slug(item.ID) != unitID {
		return UnitMembers{}
	}
	item = CopyUnit(item)
	return activeUnitMembers(item)
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
		blob, validity, err := EncodeUnitWithValidity(item)
		if err != nil {
			return err
		}
		if _, err := imports.Exec(
			`INSERT INTO units(tenant_id, tenant_slug, id, data, party_validity) VALUES($1, $2, $3, $4, $5) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			tenant.ID, tenant.Slug, item.ID, blob, validity,
		); err != nil {
			return err
		}
	}
	return nil
}
