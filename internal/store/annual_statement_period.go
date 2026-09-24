package store

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// AnnualStatementPeriod is the reusable time frame for one annual statement.
// It deliberately contains no allocation, legal, document, or calculation
// data; those belong to later product slices.
type AnnualStatementPeriod struct {
	Year      int       `json:"year"`
	StartsOn  string    `json:"starts_on"`
	EndsOn    string    `json:"ends_on"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}

// AnnualStatementPeriodUnitBasis freezes the allocation inputs that may vary
// between statement years. Unit identity, label and parties remain canonical
// on Unit and are joined by UnitID when a period is rendered.
type AnnualStatementPeriodUnitBasis struct {
	UnitID                 string `json:"unit_id"`
	MiteigentumsanteilPPM  int    `json:"miteigentumsanteil_ppm"`
	UsableAreaM2Hundredths int    `json:"usable_area_m2_hundredths"`
	UsableAreaRecorded     bool   `json:"usable_area_recorded"`
	Persons                int    `json:"persons"`
	PersonsRecorded        bool   `json:"persons_recorded"`
}

// AnnualStatementPeriodStructure is the non-monetary, independently editable
// template for one period. Receipts, prepayments and settlement results never
// enter this value and therefore cannot be carried into a follow-up year.
type AnnualStatementPeriodStructure struct {
	Legal     AnnualStatementLegalSettings     `json:"legal"`
	CostTypes []AnnualStatementCostType        `json:"cost_types"`
	UnitBases []AnnualStatementPeriodUnitBasis `json:"unit_bases"`
}

type AnnualStatementPeriodRepository interface {
	SaveLegal(year int, settings AnnualStatementLegalSettings) error
	Create(period AnnualStatementPeriod) (AnnualStatementPeriod, bool, error)
	CloneStructure(sourceYear int, period AnnualStatementPeriod) (AnnualStatementPeriod, bool, error)
	EnsureStructure(year int, costTypes []AnnualStatementCostType, units []Unit, updatedBy string) error
	Structure(year int) (AnnualStatementPeriodStructure, bool)
	SaveStructureCostType(year int, costType AnnualStatementCostType) (AnnualStatementCostType, error)
	SaveStructureUnitBases(year int, bases []AnnualStatementPeriodUnitBasis) error
	Save(period AnnualStatementPeriod) (AnnualStatementPeriod, error)
	SaveWithStructure(period AnnualStatementPeriod, costTypes []AnnualStatementCostType, units []Unit) (AnnualStatementPeriod, error)
	List() []AnnualStatementPeriod
}

type AnnualStatementPeriodStorage interface {
	annualStatementPeriodStorage()
}

type annualStatementPeriodBackend interface {
	saveAnnualStatementLegal(TenantRef, int, AnnualStatementLegalSettings) error
	createAnnualStatementPeriod(tenant TenantRef, period AnnualStatementPeriod) (AnnualStatementPeriod, bool, error)
	cloneAnnualStatementPeriodStructure(tenant TenantRef, sourceYear int, period AnnualStatementPeriod) (AnnualStatementPeriod, bool, error)
	ensureAnnualStatementPeriodStructure(tenant TenantRef, year int, costTypes []AnnualStatementCostType, units []Unit, updatedBy string) error
	annualStatementPeriodStructure(tenant TenantRef, year int) (AnnualStatementPeriodStructure, bool)
	saveAnnualStatementPeriodCostType(tenant TenantRef, year int, costType AnnualStatementCostType) (AnnualStatementCostType, error)
	saveAnnualStatementPeriodUnitBases(tenant TenantRef, year int, bases []AnnualStatementPeriodUnitBasis) error
	saveAnnualStatementPeriod(tenant TenantRef, period AnnualStatementPeriod) (AnnualStatementPeriod, error)
	saveAnnualStatementPeriodWithStructure(tenant TenantRef, period AnnualStatementPeriod, costTypes []AnnualStatementCostType, units []Unit) (AnnualStatementPeriod, error)
	listAnnualStatementPeriods(tenant TenantRef) []AnnualStatementPeriod
}

type boundAnnualStatementPeriodRepository struct {
	storage annualStatementPeriodBackend
	tenant  TenantRef
}

func BindAnnualStatementPeriodRepository(storage AnnualStatementPeriodStorage, tenant TenantRef) (AnnualStatementPeriodRepository, bool) {
	backend, ok := storage.(annualStatementPeriodBackend)
	resolved, tenantOK := validTenantRef(tenant)
	if !ok || !tenantOK {
		return nil, false
	}
	return &boundAnnualStatementPeriodRepository{storage: backend, tenant: resolved}, true
}

func (r *boundAnnualStatementPeriodRepository) Save(period AnnualStatementPeriod) (AnnualStatementPeriod, error) {
	return r.storage.saveAnnualStatementPeriod(r.tenant, period)
}

func (r *boundAnnualStatementPeriodRepository) SaveWithStructure(period AnnualStatementPeriod, costTypes []AnnualStatementCostType, units []Unit) (AnnualStatementPeriod, error) {
	return r.storage.saveAnnualStatementPeriodWithStructure(r.tenant, period, costTypes, units)
}

func (r *boundAnnualStatementPeriodRepository) Create(period AnnualStatementPeriod) (AnnualStatementPeriod, bool, error) {
	return r.storage.createAnnualStatementPeriod(r.tenant, period)
}

func (r *boundAnnualStatementPeriodRepository) CloneStructure(sourceYear int, period AnnualStatementPeriod) (AnnualStatementPeriod, bool, error) {
	return r.storage.cloneAnnualStatementPeriodStructure(r.tenant, sourceYear, period)
}

func (r *boundAnnualStatementPeriodRepository) EnsureStructure(year int, costTypes []AnnualStatementCostType, units []Unit, updatedBy string) error {
	return r.storage.ensureAnnualStatementPeriodStructure(r.tenant, year, costTypes, units, updatedBy)
}

func (r *boundAnnualStatementPeriodRepository) Structure(year int) (AnnualStatementPeriodStructure, bool) {
	return r.storage.annualStatementPeriodStructure(r.tenant, year)
}

func (r *boundAnnualStatementPeriodRepository) SaveStructureCostType(year int, costType AnnualStatementCostType) (AnnualStatementCostType, error) {
	return r.storage.saveAnnualStatementPeriodCostType(r.tenant, year, costType)
}

func (r *boundAnnualStatementPeriodRepository) SaveStructureUnitBases(year int, bases []AnnualStatementPeriodUnitBasis) error {
	return r.storage.saveAnnualStatementPeriodUnitBases(r.tenant, year, bases)
}

func (r *boundAnnualStatementPeriodRepository) List() []AnnualStatementPeriod {
	return r.storage.listAnnualStatementPeriods(r.tenant)
}

// MemoryAnnualStatementPeriodStore is used by isolated server tests. Production
// uses SQLAnnualStatementPeriodStore.
type MemoryAnnualStatementPeriodStore struct {
	mu         sync.Mutex
	byHome     map[string]map[int]AnnualStatementPeriod
	structures map[string]map[int]AnnualStatementPeriodStructure
}

func NewMemoryAnnualStatementPeriodStore() *MemoryAnnualStatementPeriodStore {
	return &MemoryAnnualStatementPeriodStore{
		byHome: map[string]map[int]AnnualStatementPeriod{}, structures: map[string]map[int]AnnualStatementPeriodStructure{},
	}
}

func (*MemoryAnnualStatementPeriodStore) annualStatementPeriodStorage() {}

func (s *MemoryAnnualStatementPeriodStore) createAnnualStatementPeriod(tenant TenantRef, period AnnualStatementPeriod) (AnnualStatementPeriod, bool, error) {
	period = normalizeAnnualStatementPeriod(period)
	if err := validateAnnualStatementPeriod(period); err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byHome == nil {
		s.byHome = map[string]map[int]AnnualStatementPeriod{}
	}
	if s.byHome[tenant.ID] == nil {
		s.byHome[tenant.ID] = map[int]AnnualStatementPeriod{}
	}
	if _, exists := s.byHome[tenant.ID][period.Year]; exists {
		return AnnualStatementPeriod{}, false, nil
	}
	if s.structures == nil {
		s.structures = map[string]map[int]AnnualStatementPeriodStructure{}
	}
	if s.structures[tenant.ID] == nil {
		s.structures[tenant.ID] = map[int]AnnualStatementPeriodStructure{}
	}
	s.byHome[tenant.ID][period.Year] = period
	s.structures[tenant.ID][period.Year] = annualStatementCompatibilityStructure(period)
	return period, true, nil
}

func (s *MemoryAnnualStatementPeriodStore) ensureAnnualStatementPeriodStructure(tenant TenantRef, year int, costTypes []AnnualStatementCostType, units []Unit, updatedBy string) error {
	structure, err := normalizeAnnualStatementPeriodStructure(costTypes, units, updatedBy)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, found := s.byHome[tenant.ID][year]; !found {
		return fmt.Errorf("annual statement period not found")
	}
	if s.structures == nil {
		s.structures = map[string]map[int]AnnualStatementPeriodStructure{}
	}
	if s.structures[tenant.ID] == nil {
		s.structures[tenant.ID] = map[int]AnnualStatementPeriodStructure{}
	}
	if _, exists := s.structures[tenant.ID][year]; !exists {
		s.structures[tenant.ID][year] = cloneAnnualStatementPeriodStructure(structure)
	}
	return nil
}

func (s *MemoryAnnualStatementPeriodStore) annualStatementPeriodStructure(tenant TenantRef, year int) (AnnualStatementPeriodStructure, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	structure, found := s.structures[tenant.ID][year]
	return cloneAnnualStatementPeriodStructure(structure), found
}

func (s *MemoryAnnualStatementPeriodStore) cloneAnnualStatementPeriodStructure(tenant TenantRef, sourceYear int, period AnnualStatementPeriod) (AnnualStatementPeriod, bool, error) {
	period = normalizeAnnualStatementPeriod(period)
	if err := validateAnnualStatementPeriod(period); err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	source, found := s.structures[tenant.ID][sourceYear]
	if !found {
		return AnnualStatementPeriod{}, false, fmt.Errorf("annual statement source structure not found")
	}
	if _, exists := s.byHome[tenant.ID][period.Year]; exists {
		return AnnualStatementPeriod{}, false, nil
	}
	s.byHome[tenant.ID][period.Year] = period
	if s.structures[tenant.ID] == nil {
		s.structures[tenant.ID] = map[int]AnnualStatementPeriodStructure{}
	}
	s.structures[tenant.ID][period.Year] = cloneAnnualStatementPeriodStructure(source)
	return period, true, nil
}

func (s *MemoryAnnualStatementPeriodStore) saveAnnualStatementPeriodCostType(tenant TenantRef, year int, costType AnnualStatementCostType) (AnnualStatementCostType, error) {
	costType = normalizeAnnualStatementCostType(costType)
	if err := validateAnnualStatementCostType(costType); err != nil {
		return AnnualStatementCostType{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	structure, found := s.structures[tenant.ID][year]
	if !found {
		return AnnualStatementCostType{}, fmt.Errorf("annual statement period structure not found")
	}
	updated := false
	for index := range structure.CostTypes {
		if structure.CostTypes[index].Key == costType.Key {
			structure.CostTypes[index] = costType
			updated = true
			break
		}
	}
	if !updated {
		structure.CostTypes = append(structure.CostTypes, costType)
	}
	sortAnnualStatementCostTypes(structure.CostTypes)
	s.structures[tenant.ID][year] = structure
	return costType, nil
}

func (s *MemoryAnnualStatementPeriodStore) saveAnnualStatementPeriodUnitBases(tenant TenantRef, year int, bases []AnnualStatementPeriodUnitBasis) error {
	normalized, err := normalizeAnnualStatementPeriodUnitBases(bases)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	structure, found := s.structures[tenant.ID][year]
	if !found {
		return fmt.Errorf("annual statement period structure not found")
	}
	structure.UnitBases = normalized
	s.structures[tenant.ID][year] = structure
	return nil
}

func (s *MemoryAnnualStatementPeriodStore) saveAnnualStatementPeriod(tenant TenantRef, period AnnualStatementPeriod) (AnnualStatementPeriod, error) {
	period = normalizeAnnualStatementPeriod(period)
	if err := validateAnnualStatementPeriod(period); err != nil {
		return AnnualStatementPeriod{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byHome == nil {
		s.byHome = map[string]map[int]AnnualStatementPeriod{}
	}
	if s.byHome[tenant.ID] == nil {
		s.byHome[tenant.ID] = map[int]AnnualStatementPeriod{}
	}
	_, exists := s.byHome[tenant.ID][period.Year]
	s.byHome[tenant.ID][period.Year] = period
	if !exists {
		if s.structures == nil {
			s.structures = map[string]map[int]AnnualStatementPeriodStructure{}
		}
		if s.structures[tenant.ID] == nil {
			s.structures[tenant.ID] = map[int]AnnualStatementPeriodStructure{}
		}
		s.structures[tenant.ID][period.Year] = annualStatementCompatibilityStructure(period)
	}
	return period, nil
}

func (s *MemoryAnnualStatementPeriodStore) saveAnnualStatementPeriodWithStructure(tenant TenantRef, period AnnualStatementPeriod, costTypes []AnnualStatementCostType, units []Unit) (AnnualStatementPeriod, error) {
	period = normalizeAnnualStatementPeriod(period)
	if err := validateAnnualStatementPeriod(period); err != nil {
		return AnnualStatementPeriod{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byHome == nil {
		s.byHome = map[string]map[int]AnnualStatementPeriod{}
	}
	if s.byHome[tenant.ID] == nil {
		s.byHome[tenant.ID] = map[int]AnnualStatementPeriod{}
	}
	if _, exists := s.byHome[tenant.ID][period.Year]; exists {
		s.byHome[tenant.ID][period.Year] = period
		return period, nil
	}
	structure, err := normalizeAnnualStatementPeriodStructure(costTypes, units, period.UpdatedBy)
	if err != nil {
		return AnnualStatementPeriod{}, err
	}
	if s.structures == nil {
		s.structures = map[string]map[int]AnnualStatementPeriodStructure{}
	}
	if s.structures[tenant.ID] == nil {
		s.structures[tenant.ID] = map[int]AnnualStatementPeriodStructure{}
	}
	s.byHome[tenant.ID][period.Year] = period
	s.structures[tenant.ID][period.Year] = cloneAnnualStatementPeriodStructure(structure)
	return period, nil
}

func (s *MemoryAnnualStatementPeriodStore) listAnnualStatementPeriods(tenant TenantRef) []AnnualStatementPeriod {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]AnnualStatementPeriod, 0, len(s.byHome[tenant.ID]))
	for _, period := range s.byHome[tenant.ID] {
		out = append(out, period)
	}
	sortAnnualStatementPeriods(out)
	return out
}

// SQLAnnualStatementPeriodStore persists periods in annual_statement_periods.
type SQLAnnualStatementPeriodStore struct {
	db *TenantDB
}

func NewSQLAnnualStatementPeriodStore(db *TenantDB) *SQLAnnualStatementPeriodStore {
	return &SQLAnnualStatementPeriodStore{db: db}
}

func (*SQLAnnualStatementPeriodStore) annualStatementPeriodStorage() {}

func (s *SQLAnnualStatementPeriodStore) ensureAnnualStatementPeriodStructure(tenant TenantRef, year int, costTypes []AnnualStatementCostType, units []Unit, updatedBy string) error {
	structure, err := normalizeAnnualStatementPeriodStructure(costTypes, units, updatedBy)
	if err != nil {
		return err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var periodExists, structureExists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM annual_statement_periods WHERE tenant_id=$1 AND year=$2)`, tenant.ID, year).Scan(&periodExists); err != nil {
		return err
	}
	if !periodExists {
		return fmt.Errorf("annual statement period not found")
	}
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM annual_statement_period_cost_types WHERE tenant_id=$1 AND period_year=$2)`, tenant.ID, year).Scan(&structureExists); err != nil {
		return err
	}
	if structureExists {
		return tx.Commit()
	}
	for _, costType := range structure.CostTypes {
		if _, err := tx.Exec(`INSERT INTO annual_statement_period_cost_types(
			tenant_id,tenant_slug,period_year,key,name,allocatable,allocation_key,updated_at,updated_by)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			tenant.ID, tenant.Slug, year, costType.Key, costType.Name, costType.Allocatable, costType.AllocationKey,
			costType.UpdatedAt.Format(time.RFC3339Nano), costType.UpdatedBy); err != nil {
			return err
		}
	}
	for _, basis := range structure.UnitBases {
		if err := insertAnnualStatementPeriodUnitBasis(tx, tenant, year, basis); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLAnnualStatementPeriodStore) annualStatementPeriodStructure(tenant TenantRef, year int) (AnnualStatementPeriodStructure, bool) {
	structure := AnnualStatementPeriodStructure{}
	legal, err := loadAnnualStatementLegal(s.db.For(tenant), tenant, year)
	if err != nil {
		return structure, false
	}
	structure.Legal = legal
	rows, err := s.db.For(tenant).Query(`SELECT key,name,allocatable,allocation_key,updated_at,updated_by
		FROM annual_statement_period_cost_types WHERE tenant_id=$1 AND period_year=$2 ORDER BY name,key`, tenant.ID, year)
	if err != nil {
		return structure, false
	}
	for rows.Next() {
		var item AnnualStatementCostType
		var updatedAt string
		if err := rows.Scan(&item.Key, &item.Name, &item.Allocatable, &item.AllocationKey, &updatedAt, &item.UpdatedBy); err != nil {
			rows.Close()
			return AnnualStatementPeriodStructure{}, false
		}
		item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		structure.CostTypes = append(structure.CostTypes, item)
	}
	if err := rows.Close(); err != nil || len(structure.CostTypes) == 0 {
		return AnnualStatementPeriodStructure{}, false
	}
	basisRows, err := s.db.For(tenant).Query(`SELECT unit_id,miteigentumsanteil_ppm,usable_area_m2_hundredths,usable_area_recorded,persons,persons_recorded
		FROM annual_statement_period_unit_bases WHERE tenant_id=$1 AND period_year=$2 ORDER BY unit_id`, tenant.ID, year)
	if err != nil {
		return AnnualStatementPeriodStructure{}, false
	}
	defer basisRows.Close()
	for basisRows.Next() {
		var basis AnnualStatementPeriodUnitBasis
		if err := basisRows.Scan(&basis.UnitID, &basis.MiteigentumsanteilPPM, &basis.UsableAreaM2Hundredths, &basis.UsableAreaRecorded, &basis.Persons, &basis.PersonsRecorded); err != nil {
			return AnnualStatementPeriodStructure{}, false
		}
		structure.UnitBases = append(structure.UnitBases, basis)
	}
	return structure, basisRows.Err() == nil
}

func (s *SQLAnnualStatementPeriodStore) cloneAnnualStatementPeriodStructure(tenant TenantRef, sourceYear int, period AnnualStatementPeriod) (AnnualStatementPeriod, bool, error) {
	period = normalizeAnnualStatementPeriod(period)
	if err := validateAnnualStatementPeriod(period); err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	defer tx.Rollback()
	var sourceExists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM annual_statement_period_cost_types WHERE tenant_id=$1 AND period_year=$2)`, tenant.ID, sourceYear).Scan(&sourceExists); err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	if !sourceExists {
		return AnnualStatementPeriod{}, false, fmt.Errorf("annual statement source structure not found")
	}
	result, err := tx.Exec(`INSERT INTO annual_statement_periods(tenant_id,tenant_slug,year,starts_on,ends_on,updated_at,updated_by)
		VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(tenant_slug,year) DO NOTHING`, tenant.ID, tenant.Slug, period.Year,
		period.StartsOn, period.EndsOn, period.UpdatedAt.Format(time.RFC3339Nano), period.UpdatedBy)
	if err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return AnnualStatementPeriod{}, false, err
	}
	if _, err := tx.Exec(`INSERT INTO annual_statement_period_cost_types(
		tenant_id,tenant_slug,period_year,key,name,allocatable,allocation_key,updated_at,updated_by)
		SELECT tenant_id,tenant_slug,$1,key,name,allocatable,allocation_key,updated_at,updated_by
		FROM annual_statement_period_cost_types WHERE tenant_id=$2 AND period_year=$3`, period.Year, tenant.ID, sourceYear); err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	if _, err := tx.Exec(`INSERT INTO annual_statement_period_unit_bases(
		tenant_id,tenant_slug,period_year,unit_id,miteigentumsanteil_ppm,usable_area_m2_hundredths,usable_area_recorded,persons,persons_recorded)
		SELECT tenant_id,tenant_slug,$1,unit_id,miteigentumsanteil_ppm,usable_area_m2_hundredths,usable_area_recorded,persons,persons_recorded
		FROM annual_statement_period_unit_bases WHERE tenant_id=$2 AND period_year=$3`, period.Year, tenant.ID, sourceYear); err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	if _, err := tx.Exec(`UPDATE annual_statement_periods SET legal_settings=(SELECT legal_settings FROM annual_statement_periods WHERE tenant_id=$1 AND year=$2) WHERE tenant_id=$1 AND year=$3`, tenant.ID, sourceYear, period.Year); err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	return period, true, nil
}

func (s *SQLAnnualStatementPeriodStore) saveAnnualStatementPeriodCostType(tenant TenantRef, year int, costType AnnualStatementCostType) (AnnualStatementCostType, error) {
	costType = normalizeAnnualStatementCostType(costType)
	if err := validateAnnualStatementCostType(costType); err != nil {
		return AnnualStatementCostType{}, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return AnnualStatementCostType{}, err
	}
	defer tx.Rollback()
	var structureExists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM annual_statement_period_cost_types WHERE tenant_id=$1 AND period_year=$2)`, tenant.ID, year).Scan(&structureExists); err != nil {
		return AnnualStatementCostType{}, err
	}
	if !structureExists {
		return AnnualStatementCostType{}, fmt.Errorf("annual statement period structure not found")
	}
	result, err := tx.Exec(`UPDATE annual_statement_period_cost_types
		SET name=$1,allocatable=$2,allocation_key=$3,updated_at=$4,updated_by=$5
		WHERE tenant_id=$6 AND period_year=$7 AND key=$8`, costType.Name, costType.Allocatable, costType.AllocationKey,
		costType.UpdatedAt.Format(time.RFC3339Nano), costType.UpdatedBy, tenant.ID, year, costType.Key)
	if err != nil {
		return AnnualStatementCostType{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return AnnualStatementCostType{}, err
	}
	if affected == 0 {
		if _, err := tx.Exec(`INSERT INTO annual_statement_period_cost_types(
			tenant_id,tenant_slug,period_year,key,name,allocatable,allocation_key,updated_at,updated_by)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, tenant.ID, tenant.Slug, year, costType.Key, costType.Name,
			costType.Allocatable, costType.AllocationKey, costType.UpdatedAt.Format(time.RFC3339Nano), costType.UpdatedBy); err != nil {
			return AnnualStatementCostType{}, err
		}
	}
	return costType, tx.Commit()
}

func (s *SQLAnnualStatementPeriodStore) saveAnnualStatementPeriodUnitBases(tenant TenantRef, year int, bases []AnnualStatementPeriodUnitBasis) error {
	normalized, err := normalizeAnnualStatementPeriodUnitBases(bases)
	if err != nil {
		return err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var structureExists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM annual_statement_period_cost_types WHERE tenant_id=$1 AND period_year=$2)`, tenant.ID, year).Scan(&structureExists); err != nil {
		return err
	}
	if !structureExists {
		return fmt.Errorf("annual statement period structure not found")
	}
	if _, err := tx.Exec(`DELETE FROM annual_statement_period_unit_bases WHERE tenant_id=$1 AND period_year=$2`, tenant.ID, year); err != nil {
		return err
	}
	for _, basis := range normalized {
		if err := insertAnnualStatementPeriodUnitBasis(tx, tenant, year, basis); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLAnnualStatementPeriodStore) createAnnualStatementPeriod(tenant TenantRef, period AnnualStatementPeriod) (AnnualStatementPeriod, bool, error) {
	period = normalizeAnnualStatementPeriod(period)
	if err := validateAnnualStatementPeriod(period); err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(
		`INSERT INTO annual_statement_periods(tenant_id, tenant_slug, year, starts_on, ends_on, updated_at, updated_by)
		 VALUES($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT(tenant_slug, year) DO NOTHING`,
		tenant.ID, tenant.Slug, period.Year, period.StartsOn, period.EndsOn,
		period.UpdatedAt.Format(time.RFC3339Nano), period.UpdatedBy,
	)
	if err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	if affected == 0 {
		return period, false, nil
	}
	if err := insertAnnualStatementPeriodCostTypes(tx, tenant, period.Year, annualStatementCompatibilityStructure(period).CostTypes); err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	return period, true, nil
}

func (s *SQLAnnualStatementPeriodStore) saveAnnualStatementPeriod(tenant TenantRef, period AnnualStatementPeriod) (AnnualStatementPeriod, error) {
	period = normalizeAnnualStatementPeriod(period)
	if err := validateAnnualStatementPeriod(period); err != nil {
		return AnnualStatementPeriod{}, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return AnnualStatementPeriod{}, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(
		`UPDATE annual_statement_periods
		 SET starts_on=$1, ends_on=$2, updated_at=$3, updated_by=$4
		 WHERE tenant_id=$5 AND year=$6`,
		period.StartsOn, period.EndsOn, period.UpdatedAt.Format(time.RFC3339Nano), period.UpdatedBy,
		tenant.ID, period.Year,
	)
	if err != nil {
		return AnnualStatementPeriod{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return AnnualStatementPeriod{}, err
	}
	if affected > 0 {
		return period, tx.Commit()
	}
	if _, err := tx.Exec(
		`INSERT INTO annual_statement_periods(tenant_id, tenant_slug, year, starts_on, ends_on, updated_at, updated_by)
		 VALUES($1,$2,$3,$4,$5,$6,$7)`,
		tenant.ID, tenant.Slug, period.Year, period.StartsOn, period.EndsOn,
		period.UpdatedAt.Format(time.RFC3339Nano), period.UpdatedBy,
	); err != nil {
		return AnnualStatementPeriod{}, err
	}
	if err := insertAnnualStatementPeriodCostTypes(tx, tenant, period.Year, annualStatementCompatibilityStructure(period).CostTypes); err != nil {
		return AnnualStatementPeriod{}, err
	}
	return period, tx.Commit()
}

func (s *SQLAnnualStatementPeriodStore) saveAnnualStatementPeriodWithStructure(tenant TenantRef, period AnnualStatementPeriod, costTypes []AnnualStatementCostType, units []Unit) (AnnualStatementPeriod, error) {
	period = normalizeAnnualStatementPeriod(period)
	if err := validateAnnualStatementPeriod(period); err != nil {
		return AnnualStatementPeriod{}, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return AnnualStatementPeriod{}, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE annual_statement_periods
		SET starts_on=$1,ends_on=$2,updated_at=$3,updated_by=$4
		WHERE tenant_id=$5 AND year=$6`, period.StartsOn, period.EndsOn, period.UpdatedAt.Format(time.RFC3339Nano), period.UpdatedBy, tenant.ID, period.Year)
	if err != nil {
		return AnnualStatementPeriod{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return AnnualStatementPeriod{}, err
	}
	if affected > 0 {
		return period, tx.Commit()
	}
	structure, err := normalizeAnnualStatementPeriodStructure(costTypes, units, period.UpdatedBy)
	if err != nil {
		return AnnualStatementPeriod{}, err
	}
	if _, err := tx.Exec(`INSERT INTO annual_statement_periods(
		tenant_id,tenant_slug,year,starts_on,ends_on,updated_at,updated_by)
		VALUES($1,$2,$3,$4,$5,$6,$7)`, tenant.ID, tenant.Slug, period.Year, period.StartsOn, period.EndsOn,
		period.UpdatedAt.Format(time.RFC3339Nano), period.UpdatedBy); err != nil {
		return AnnualStatementPeriod{}, err
	}
	for _, costType := range structure.CostTypes {
		if _, err := tx.Exec(`INSERT INTO annual_statement_period_cost_types(
			tenant_id,tenant_slug,period_year,key,name,allocatable,allocation_key,updated_at,updated_by)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, tenant.ID, tenant.Slug, period.Year, costType.Key, costType.Name,
			costType.Allocatable, costType.AllocationKey, costType.UpdatedAt.Format(time.RFC3339Nano), costType.UpdatedBy); err != nil {
			return AnnualStatementPeriod{}, err
		}
	}
	for _, basis := range structure.UnitBases {
		if err := insertAnnualStatementPeriodUnitBasis(tx, tenant, period.Year, basis); err != nil {
			return AnnualStatementPeriod{}, err
		}
	}
	return period, tx.Commit()
}

func (s *SQLAnnualStatementPeriodStore) listAnnualStatementPeriods(tenant TenantRef) []AnnualStatementPeriod {
	rows, err := s.db.For(tenant).Query(
		`SELECT year, starts_on, ends_on, updated_at, updated_by
		 FROM annual_statement_periods WHERE tenant_id=$1 ORDER BY year DESC`, tenant.ID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []AnnualStatementPeriod{}
	for rows.Next() {
		var period AnnualStatementPeriod
		var updatedAt string
		if err := rows.Scan(&period.Year, &period.StartsOn, &period.EndsOn, &updatedAt, &period.UpdatedBy); err != nil {
			continue
		}
		period.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		out = append(out, period)
	}
	return out
}

type annualStatementPeriodExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func insertAnnualStatementPeriodCostTypes(exec annualStatementPeriodExecer, tenant TenantRef, year int, costTypes []AnnualStatementCostType) error {
	for _, costType := range costTypes {
		if _, err := exec.Exec(`INSERT INTO annual_statement_period_cost_types(
			tenant_id,tenant_slug,period_year,key,name,allocatable,allocation_key,updated_at,updated_by)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, tenant.ID, tenant.Slug, year, costType.Key, costType.Name,
			costType.Allocatable, costType.AllocationKey, costType.UpdatedAt.Format(time.RFC3339Nano), costType.UpdatedBy); err != nil {
			return err
		}
	}
	return nil
}

func insertAnnualStatementPeriodUnitBasis(exec annualStatementPeriodExecer, tenant TenantRef, year int, basis AnnualStatementPeriodUnitBasis) error {
	_, err := exec.Exec(`INSERT INTO annual_statement_period_unit_bases(
		tenant_id,tenant_slug,period_year,unit_id,miteigentumsanteil_ppm,usable_area_m2_hundredths,usable_area_recorded,persons,persons_recorded)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, tenant.ID, tenant.Slug, year, basis.UnitID, basis.MiteigentumsanteilPPM,
		basis.UsableAreaM2Hundredths, basis.UsableAreaRecorded, basis.Persons, basis.PersonsRecorded)
	return err
}

func normalizeAnnualStatementPeriodStructure(costTypes []AnnualStatementCostType, units []Unit, updatedBy string) (AnnualStatementPeriodStructure, error) {
	updatedBy = strings.ToLower(strings.TrimSpace(updatedBy))
	if len(costTypes) == 0 || updatedBy == "" {
		return AnnualStatementPeriodStructure{}, fmt.Errorf("annual statement period structure requires cost types and actor")
	}
	structure := AnnualStatementPeriodStructure{Legal: DefaultAnnualStatementLegalSettings(), CostTypes: make([]AnnualStatementCostType, 0, len(costTypes))}
	for _, costType := range costTypes {
		costType = normalizeAnnualStatementCostType(costType)
		if costType.UpdatedBy == "" {
			costType.UpdatedBy = updatedBy
		}
		if err := validateAnnualStatementCostType(costType); err != nil {
			return AnnualStatementPeriodStructure{}, err
		}
		structure.CostTypes = append(structure.CostTypes, costType)
	}
	sortAnnualStatementCostTypes(structure.CostTypes)
	bases := make([]AnnualStatementPeriodUnitBasis, 0, len(units))
	for _, unit := range units {
		bases = append(bases, AnnualStatementPeriodUnitBasis{
			UnitID: unit.ID, MiteigentumsanteilPPM: unit.MiteigentumsanteilPPM,
			UsableAreaM2Hundredths: unit.UsableAreaM2Hundredths, UsableAreaRecorded: unit.UsableAreaRecorded,
			Persons: unit.Persons, PersonsRecorded: unit.PersonsRecorded,
		})
	}
	var err error
	structure.UnitBases, err = normalizeAnnualStatementPeriodUnitBases(bases)
	if err != nil {
		return AnnualStatementPeriodStructure{}, err
	}
	return structure, nil
}

// annualStatementCompatibilityStructure is the write-time bridge for callers
// that still use Save/Create without supplying an explicit catalogue. Before
// persisted catalogues existed, those callers saw the starter rows read-only.
// Freezing them while the period is written keeps later reads pure and gives
// the compatibility period the same immutable semantics as SaveWithStructure.
func annualStatementCompatibilityStructure(period AnnualStatementPeriod) AnnualStatementPeriodStructure {
	costTypes := AnnualStatementDefaultCostTypes(period.UpdatedBy)
	for index := range costTypes {
		costTypes[index].UpdatedAt = period.UpdatedAt
	}
	sortAnnualStatementCostTypes(costTypes)
	return AnnualStatementPeriodStructure{Legal: DefaultAnnualStatementLegalSettings(), CostTypes: costTypes}
}

func normalizeAnnualStatementPeriodUnitBases(bases []AnnualStatementPeriodUnitBasis) ([]AnnualStatementPeriodUnitBasis, error) {
	out := make([]AnnualStatementPeriodUnitBasis, 0, len(bases))
	seen := map[string]struct{}{}
	for _, basis := range bases {
		basis.UnitID = NormalizeUnitID(basis.UnitID)
		if _, duplicate := seen[basis.UnitID]; basis.UnitID == "" || duplicate || basis.MiteigentumsanteilPPM < 0 || basis.MiteigentumsanteilPPM > MiteigentumsanteilTotalPPM ||
			basis.UsableAreaM2Hundredths < 0 || basis.UsableAreaM2Hundredths > 9_999_999 || basis.Persons < 0 || basis.Persons > 10_000 {
			return nil, fmt.Errorf("invalid annual statement period unit basis")
		}
		seen[basis.UnitID] = struct{}{}
		if !basis.UsableAreaRecorded {
			basis.UsableAreaM2Hundredths = 0
		}
		if !basis.PersonsRecorded {
			basis.Persons = 0
		}
		out = append(out, basis)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UnitID < out[j].UnitID })
	return out, nil
}

func cloneAnnualStatementPeriodStructure(structure AnnualStatementPeriodStructure) AnnualStatementPeriodStructure {
	return AnnualStatementPeriodStructure{
		Legal:     cloneAnnualStatementLegal(structure.Legal),
		CostTypes: append([]AnnualStatementCostType(nil), structure.CostTypes...),
		UnitBases: append([]AnnualStatementPeriodUnitBasis(nil), structure.UnitBases...),
	}
}

func normalizeAnnualStatementPeriod(period AnnualStatementPeriod) AnnualStatementPeriod {
	period.StartsOn = strings.TrimSpace(period.StartsOn)
	period.EndsOn = strings.TrimSpace(period.EndsOn)
	period.UpdatedBy = strings.ToLower(strings.TrimSpace(period.UpdatedBy))
	if period.UpdatedAt.IsZero() {
		period.UpdatedAt = time.Now().UTC()
	} else {
		period.UpdatedAt = period.UpdatedAt.UTC()
	}
	return period
}

func validateAnnualStatementPeriod(period AnnualStatementPeriod) error {
	if period.Year < 1 || period.Year > 9999 {
		return fmt.Errorf("invalid annual statement year")
	}
	startsOn, startErr := time.Parse("2006-01-02", period.StartsOn)
	endsOn, endErr := time.Parse("2006-01-02", period.EndsOn)
	if startErr != nil || endErr != nil || endsOn.Before(startsOn) {
		return fmt.Errorf("invalid annual statement period")
	}
	return nil
}

func sortAnnualStatementPeriods(periods []AnnualStatementPeriod) {
	sort.Slice(periods, func(i, j int) bool { return periods[i].Year > periods[j].Year })
}

var _ AnnualStatementPeriodStorage = (*MemoryAnnualStatementPeriodStore)(nil)
var _ AnnualStatementPeriodStorage = (*SQLAnnualStatementPeriodStore)(nil)
