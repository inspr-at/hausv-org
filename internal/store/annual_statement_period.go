package store

import (
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

type AnnualStatementPeriodRepository interface {
	Create(period AnnualStatementPeriod) (AnnualStatementPeriod, bool, error)
	Save(period AnnualStatementPeriod) (AnnualStatementPeriod, error)
	List() []AnnualStatementPeriod
}

type AnnualStatementPeriodStorage interface {
	annualStatementPeriodStorage()
}

type annualStatementPeriodBackend interface {
	createAnnualStatementPeriod(tenant TenantRef, period AnnualStatementPeriod) (AnnualStatementPeriod, bool, error)
	saveAnnualStatementPeriod(tenant TenantRef, period AnnualStatementPeriod) (AnnualStatementPeriod, error)
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

func (r *boundAnnualStatementPeriodRepository) Create(period AnnualStatementPeriod) (AnnualStatementPeriod, bool, error) {
	return r.storage.createAnnualStatementPeriod(r.tenant, period)
}

func (r *boundAnnualStatementPeriodRepository) List() []AnnualStatementPeriod {
	return r.storage.listAnnualStatementPeriods(r.tenant)
}

// MemoryAnnualStatementPeriodStore is used by isolated server tests. Production
// uses SQLAnnualStatementPeriodStore.
type MemoryAnnualStatementPeriodStore struct {
	mu     sync.Mutex
	byHome map[string]map[int]AnnualStatementPeriod
}

func NewMemoryAnnualStatementPeriodStore() *MemoryAnnualStatementPeriodStore {
	return &MemoryAnnualStatementPeriodStore{byHome: map[string]map[int]AnnualStatementPeriod{}}
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
	s.byHome[tenant.ID][period.Year] = period
	return period, true, nil
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
	s.byHome[tenant.ID][period.Year] = period
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

func (s *SQLAnnualStatementPeriodStore) createAnnualStatementPeriod(tenant TenantRef, period AnnualStatementPeriod) (AnnualStatementPeriod, bool, error) {
	period = normalizeAnnualStatementPeriod(period)
	if err := validateAnnualStatementPeriod(period); err != nil {
		return AnnualStatementPeriod{}, false, err
	}
	result, err := s.db.For(tenant).Exec(
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
	return period, affected == 1, nil
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
