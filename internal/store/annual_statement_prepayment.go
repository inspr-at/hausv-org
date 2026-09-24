package store

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// AnnualStatementPrepayment is the amount one unit paid in advance for one
// annual-statement period. Absence means "not recorded"; an explicit zero is
// a valid recorded amount.
type AnnualStatementPrepayment struct {
	PeriodYear  int       `json:"period_year"`
	UnitID      string    `json:"unit_id"`
	AmountCents int64     `json:"amount_cents"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   string    `json:"updated_by"`
}

type AnnualStatementPrepaymentRepository interface {
	Save(item AnnualStatementPrepayment) (saved AnnualStatementPrepayment, previous *AnnualStatementPrepayment, err error)
	Get(year int, unitID string) (AnnualStatementPrepayment, bool)
	ListByPeriod(year int) []AnnualStatementPrepayment
}

type AnnualStatementPrepaymentStorage interface{ annualStatementPrepaymentStorage() }

type annualStatementPrepaymentBackend interface {
	saveAnnualStatementPrepayment(TenantRef, AnnualStatementPrepayment) (AnnualStatementPrepayment, *AnnualStatementPrepayment, error)
	getAnnualStatementPrepayment(TenantRef, int, string) (AnnualStatementPrepayment, bool)
	listAnnualStatementPrepayments(TenantRef, int) []AnnualStatementPrepayment
}

type boundAnnualStatementPrepaymentRepository struct {
	storage annualStatementPrepaymentBackend
	tenant  TenantRef
}

func BindAnnualStatementPrepaymentRepository(storage AnnualStatementPrepaymentStorage, tenant TenantRef) (AnnualStatementPrepaymentRepository, bool) {
	backend, ok := storage.(annualStatementPrepaymentBackend)
	resolved, tenantOK := validTenantRef(tenant)
	if !ok || !tenantOK {
		return nil, false
	}
	return &boundAnnualStatementPrepaymentRepository{storage: backend, tenant: resolved}, true
}

func (r *boundAnnualStatementPrepaymentRepository) Save(item AnnualStatementPrepayment) (AnnualStatementPrepayment, *AnnualStatementPrepayment, error) {
	return r.storage.saveAnnualStatementPrepayment(r.tenant, item)
}

func (r *boundAnnualStatementPrepaymentRepository) Get(year int, unitID string) (AnnualStatementPrepayment, bool) {
	return r.storage.getAnnualStatementPrepayment(r.tenant, year, unitID)
}

func (r *boundAnnualStatementPrepaymentRepository) ListByPeriod(year int) []AnnualStatementPrepayment {
	return r.storage.listAnnualStatementPrepayments(r.tenant, year)
}

type MemoryAnnualStatementPrepaymentStore struct {
	mu      sync.Mutex
	byHome  map[string]map[string]AnnualStatementPrepayment
	periods AnnualStatementPeriodStorage
	units   UnitStorage
}

func NewMemoryAnnualStatementPrepaymentStore(periods AnnualStatementPeriodStorage, units UnitStorage) *MemoryAnnualStatementPrepaymentStore {
	return &MemoryAnnualStatementPrepaymentStore{byHome: map[string]map[string]AnnualStatementPrepayment{}, periods: periods, units: units}
}

func (*MemoryAnnualStatementPrepaymentStore) annualStatementPrepaymentStorage() {}

func (s *MemoryAnnualStatementPrepaymentStore) saveAnnualStatementPrepayment(tenant TenantRef, item AnnualStatementPrepayment) (AnnualStatementPrepayment, *AnnualStatementPrepayment, error) {
	item, err := normalizeAnnualStatementPrepayment(item)
	if err != nil || !annualStatementPrepaymentReferencesValid(tenant, item, s.periods, s.units) {
		return AnnualStatementPrepayment{}, nil, fmt.Errorf("invalid annual statement prepayment")
	}
	key := annualStatementPrepaymentKey(item.PeriodYear, item.UnitID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byHome == nil {
		s.byHome = map[string]map[string]AnnualStatementPrepayment{}
	}
	if s.byHome[tenant.ID] == nil {
		s.byHome[tenant.ID] = map[string]AnnualStatementPrepayment{}
	}
	var previous *AnnualStatementPrepayment
	if existing, found := s.byHome[tenant.ID][key]; found {
		copy := existing
		previous = &copy
	}
	s.byHome[tenant.ID][key] = item
	return item, previous, nil
}

func (s *MemoryAnnualStatementPrepaymentStore) getAnnualStatementPrepayment(tenant TenantRef, year int, unitID string) (AnnualStatementPrepayment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, found := s.byHome[tenant.ID][annualStatementPrepaymentKey(year, NormalizeUnitID(unitID))]
	return item, found
}

func (s *MemoryAnnualStatementPrepaymentStore) listAnnualStatementPrepayments(tenant TenantRef, year int) []AnnualStatementPrepayment {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []AnnualStatementPrepayment{}
	for _, item := range s.byHome[tenant.ID] {
		if year == 0 || item.PeriodYear == year {
			out = append(out, item)
		}
	}
	sortAnnualStatementPrepayments(out)
	return out
}

type SQLAnnualStatementPrepaymentStore struct{ db *TenantDB }

func NewSQLAnnualStatementPrepaymentStore(db *TenantDB) *SQLAnnualStatementPrepaymentStore {
	return &SQLAnnualStatementPrepaymentStore{db: db}
}

func (*SQLAnnualStatementPrepaymentStore) annualStatementPrepaymentStorage() {}

func (s *SQLAnnualStatementPrepaymentStore) saveAnnualStatementPrepayment(tenant TenantRef, item AnnualStatementPrepayment) (AnnualStatementPrepayment, *AnnualStatementPrepayment, error) {
	item, err := normalizeAnnualStatementPrepayment(item)
	if err != nil {
		return AnnualStatementPrepayment{}, nil, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return AnnualStatementPrepayment{}, nil, err
	}
	defer tx.Rollback()
	if ok, err := sqlAnnualStatementPrepaymentReferencesValid(tx, tenant, item); err != nil || !ok {
		if err != nil {
			return AnnualStatementPrepayment{}, nil, err
		}
		return AnnualStatementPrepayment{}, nil, fmt.Errorf("invalid annual statement prepayment reference")
	}
	previous, found, err := getAnnualStatementPrepaymentQuery(tx.QueryRow(
		`SELECT period_year, unit_id, amount_cents, updated_at, updated_by FROM annual_statement_prepayments WHERE tenant_id=$1 AND period_year=$2 AND unit_id=$3`,
		tenant.ID, item.PeriodYear, item.UnitID,
	))
	if err != nil {
		return AnnualStatementPrepayment{}, nil, err
	}
	if found {
		if _, err := tx.Exec(`UPDATE annual_statement_prepayments SET amount_cents=$1, updated_at=$2, updated_by=$3 WHERE tenant_id=$4 AND period_year=$5 AND unit_id=$6`,
			item.AmountCents, item.UpdatedAt.Format(time.RFC3339Nano), item.UpdatedBy, tenant.ID, item.PeriodYear, item.UnitID); err != nil {
			return AnnualStatementPrepayment{}, nil, err
		}
	} else if _, err := tx.Exec(`INSERT INTO annual_statement_prepayments(tenant_id, tenant_slug, period_year, unit_id, amount_cents, updated_at, updated_by) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		tenant.ID, tenant.Slug, item.PeriodYear, item.UnitID, item.AmountCents, item.UpdatedAt.Format(time.RFC3339Nano), item.UpdatedBy); err != nil {
		return AnnualStatementPrepayment{}, nil, err
	}
	if err := tx.Commit(); err != nil {
		return AnnualStatementPrepayment{}, nil, err
	}
	if !found {
		return item, nil, nil
	}
	return item, &previous, nil
}

func (s *SQLAnnualStatementPrepaymentStore) getAnnualStatementPrepayment(tenant TenantRef, year int, unitID string) (AnnualStatementPrepayment, bool) {
	item, found, _ := getAnnualStatementPrepaymentQuery(s.db.For(tenant).QueryRow(
		`SELECT period_year, unit_id, amount_cents, updated_at, updated_by FROM annual_statement_prepayments WHERE tenant_id=$1 AND period_year=$2 AND unit_id=$3`,
		tenant.ID, year, NormalizeUnitID(unitID),
	))
	return item, found
}

func (s *SQLAnnualStatementPrepaymentStore) listAnnualStatementPrepayments(tenant TenantRef, year int) []AnnualStatementPrepayment {
	rows, err := s.db.For(tenant).Query(`SELECT period_year, unit_id, amount_cents, updated_at, updated_by FROM annual_statement_prepayments WHERE tenant_id=$1 AND ($2=0 OR period_year=$2) ORDER BY period_year DESC, unit_id`, tenant.ID, year)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []AnnualStatementPrepayment{}
	for rows.Next() {
		item, found, err := getAnnualStatementPrepaymentQuery(rows)
		if err == nil && found {
			out = append(out, item)
		}
	}
	return out
}

type annualStatementPrepaymentScanner interface{ Scan(...any) error }

func getAnnualStatementPrepaymentQuery(row annualStatementPrepaymentScanner) (AnnualStatementPrepayment, bool, error) {
	var item AnnualStatementPrepayment
	var updatedAt string
	if err := row.Scan(&item.PeriodYear, &item.UnitID, &item.AmountCents, &updatedAt, &item.UpdatedBy); err != nil {
		if err == sql.ErrNoRows {
			return AnnualStatementPrepayment{}, false, nil
		}
		return AnnualStatementPrepayment{}, false, err
	}
	item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return item, true, nil
}

func normalizeAnnualStatementPrepayment(item AnnualStatementPrepayment) (AnnualStatementPrepayment, error) {
	item.UnitID = NormalizeUnitID(item.UnitID)
	item.UpdatedBy = strings.ToLower(strings.TrimSpace(item.UpdatedBy))
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = time.Now().UTC()
	} else {
		item.UpdatedAt = item.UpdatedAt.UTC()
	}
	if item.PeriodYear < 1 || item.PeriodYear > 9999 || item.UnitID == "" || item.AmountCents < 0 || item.UpdatedBy == "" {
		return AnnualStatementPrepayment{}, fmt.Errorf("invalid annual statement prepayment")
	}
	return item, nil
}

func annualStatementPrepaymentReferencesValid(tenant TenantRef, item AnnualStatementPrepayment, periods AnnualStatementPeriodStorage, units UnitStorage) bool {
	periodRepo, periodOK := BindAnnualStatementPeriodRepository(periods, tenant)
	unitRepo, unitOK := BindUnitRepository(units, tenant)
	if !periodOK || !unitOK {
		return false
	}
	periodFound := false
	for _, period := range periodRepo.List() {
		periodFound = periodFound || period.Year == item.PeriodYear
	}
	unitFound := false
	for _, unit := range unitRepo.List() {
		unitFound = unitFound || NormalizeUnitID(unit.ID) == item.UnitID
	}
	return periodFound && unitFound
}

func sqlAnnualStatementPrepaymentReferencesValid(tx *sql.Tx, tenant TenantRef, item AnnualStatementPrepayment) (bool, error) {
	var periodCount, unitCount int
	if err := tx.QueryRow(`SELECT count(*) FROM annual_statement_periods WHERE tenant_id=$1 AND year=$2`, tenant.ID, item.PeriodYear).Scan(&periodCount); err != nil {
		return false, err
	}
	if err := tx.QueryRow(`SELECT count(*) FROM units WHERE tenant_id=$1 AND id=$2`, tenant.ID, item.UnitID).Scan(&unitCount); err != nil {
		return false, err
	}
	return periodCount == 1 && unitCount == 1, nil
}

func annualStatementPrepaymentKey(year int, unitID string) string {
	return fmt.Sprintf("%04d:%s", year, unitID)
}

func sortAnnualStatementPrepayments(items []AnnualStatementPrepayment) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].PeriodYear == items[j].PeriodYear {
			return items[i].UnitID < items[j].UnitID
		}
		return items[i].PeriodYear > items[j].PeriodYear
	})
}

// AnnualStatementSettlementUnit is a current, non-persisted preview of the
// allocatable receipt amount assigned to one unit. HAUSV-580 owns the actual
// versioned statement run.
type AnnualStatementSettlementUnit struct {
	UnitID         string
	Label          string
	AllocatedCents int64
}

func AnnualStatementSettlementPreview(costTypes []AnnualStatementCostType, receipts []AnnualStatementReceipt, units []Unit, consumption ...map[string]AnnualStatementConsumptionVector) ([]AnnualStatementSettlementUnit, bool) {
	return annualStatementSettlementPreview(costTypes, receipts, units, nil, consumption...)
}

func AnnualStatementSettlementPreviewWithAgreed(costTypes []AnnualStatementCostType, receipts []AnnualStatementReceipt, units []Unit, consumption map[string]AnnualStatementConsumptionVector, agreed map[string]map[string]int) ([]AnnualStatementSettlementUnit, bool) {
	return annualStatementSettlementPreview(costTypes, receipts, units, agreed, consumption)
}

func annualStatementSettlementPreview(costTypes []AnnualStatementCostType, receipts []AnnualStatementReceipt, units []Unit, agreed map[string]map[string]int, consumption ...map[string]AnnualStatementConsumptionVector) ([]AnnualStatementSettlementUnit, bool) {
	if len(units) == 0 || len(AnnualStatementCostTypesWithoutKey(costTypes)) > 0 {
		return nil, false
	}
	// Match the run's tie-break order, independent of register display order.
	units = append([]Unit(nil), units...)
	sort.Slice(units, func(i, j int) bool { return units[i].ID < units[j].ID })
	costTypeKey := map[string]string{}
	for _, item := range costTypes {
		if item.Allocatable && ValidAllocationKey(item.AllocationKey) {
			costTypeKey[item.Key] = item.AllocationKey
		}
	}
	totalByCost := map[string]int64{}
	var total int64
	for _, receipt := range receipts {
		if key := costTypeKey[receipt.CostTypeKey]; key != "" {
			if receipt.AmountCents < 0 || total > math.MaxInt64-receipt.AmountCents {
				return nil, false
			}
			total += receipt.AmountCents
			totalByCost[receipt.CostTypeKey] += receipt.AmountCents
		}
	}
	out := make([]AnnualStatementSettlementUnit, len(units))
	indexByUnit := map[string]int{}
	for index, unit := range units {
		out[index] = AnnualStatementSettlementUnit{UnitID: unit.ID, Label: unit.Label}
		indexByUnit[unit.ID] = index
	}
	for _, preview := range AnnualStatementAllocationPreviews(costTypes, units, agreed) {
		if preview.Key != AllocationKeyVerbrauch && (preview.Blocked || (preview.Key == AllocationKeyNutzwert && preview.BasisTotal != MiteigentumsanteilTotalPPM)) {
			return nil, false
		}
		// Round each cost type separately, exactly as the stored run does.
		for _, cost := range preview.CostTypeKeys {
			shares := preview.Shares
			if preview.Key == AllocationKeyVerbrauch {
				if len(consumption) != 1 || (cost != "heizung" && cost != "warmwasser") {
					return nil, false
				}
				vector, found := consumption[0][cost]
				if !found || vector.CostTypeKey != cost {
					return nil, false
				}
				for _, receipt := range receipts {
					if receipt.CostTypeKey == cost && receipt.PeriodYear != vector.PeriodYear {
						return nil, false
					}
				}
				var complete bool
				shares, complete = AnnualStatementConsumptionShares(vector, units)
				if !complete {
					return nil, false
				}
			}
			cents := annualStatementRunCents(shares, totalByCost[cost])
			for i, share := range shares {
				out[indexByUnit[share.UnitID]].AllocatedCents += cents[i]
			}
		}
	}
	return out, true
}

var _ AnnualStatementPrepaymentStorage = (*MemoryAnnualStatementPrepaymentStore)(nil)
var _ AnnualStatementPrepaymentStorage = (*SQLAnnualStatementPrepaymentStore)(nil)
