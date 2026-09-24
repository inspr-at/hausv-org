package store

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// AnnualStatementCostType is one tenant-owned catalogue entry. Allocation
// keys, receipts and calculations deliberately remain outside this slice.
type AnnualStatementCostType struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Allocatable bool   `json:"allocatable"`
	// AllocationKey is the Verteilerschlüssel used to split an allocatable
	// cost type across units (HAUSV-577). It is required exactly when the
	// cost type is allocatable and empty otherwise.
	AllocationKey string `json:"allocation_key"`
	// VATRatePercent is 0, 10 or 20 for the period snapshot. The catalogue
	// leaves it unset; the period column carries the editable rate.
	VATRatePercent int       `json:"vat_rate_percent,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      string    `json:"updated_by"`
}

// Allocation keys. Nutzwert reuses the Miteigentumsanteil already recorded on
// each unit (§ 32 WEG default). Fläche and Personen read the per-unit bases
// recorded for the statement. Verbrauch will read measured values once
// HAUSV-578 wires them; until then every unit counts as unmapped for it.
const (
	AllocationKeyNutzwert  = "nutzwert"
	AllocationKeyFlaeche   = "flaeche"
	AllocationKeyPersonen  = "personen"
	AllocationKeyVerbrauch = "verbrauch"
	AllocationKeyAgreed    = "vereinbart"
)

// AllocationKeys lists the supported keys in display order.
var AllocationKeys = []string{AllocationKeyNutzwert, AllocationKeyFlaeche, AllocationKeyPersonen, AllocationKeyVerbrauch, AllocationKeyAgreed}

func ValidAllocationKey(key string) bool {
	for _, known := range AllocationKeys {
		if key == known {
			return true
		}
	}
	return false
}

type AnnualStatementCostTypeRepository interface {
	EnsureDefaults(updatedBy string) error
	Save(costType AnnualStatementCostType) (AnnualStatementCostType, error)
	List() []AnnualStatementCostType
}

type AnnualStatementCostTypeStorage interface {
	annualStatementCostTypeStorage()
}

type annualStatementCostTypeBackend interface {
	ensureAnnualStatementCostTypeDefaults(tenant TenantRef, updatedBy string) error
	saveAnnualStatementCostType(tenant TenantRef, costType AnnualStatementCostType) (AnnualStatementCostType, error)
	listAnnualStatementCostTypes(tenant TenantRef) []AnnualStatementCostType
}

type boundAnnualStatementCostTypeRepository struct {
	storage annualStatementCostTypeBackend
	tenant  TenantRef
}

func BindAnnualStatementCostTypeRepository(storage AnnualStatementCostTypeStorage, tenant TenantRef) (AnnualStatementCostTypeRepository, bool) {
	backend, ok := storage.(annualStatementCostTypeBackend)
	resolved, tenantOK := validTenantRef(tenant)
	if !ok || !tenantOK {
		return nil, false
	}
	return &boundAnnualStatementCostTypeRepository{storage: backend, tenant: resolved}, true
}

func (r *boundAnnualStatementCostTypeRepository) EnsureDefaults(updatedBy string) error {
	return r.storage.ensureAnnualStatementCostTypeDefaults(r.tenant, updatedBy)
}

func (r *boundAnnualStatementCostTypeRepository) Save(costType AnnualStatementCostType) (AnnualStatementCostType, error) {
	return r.storage.saveAnnualStatementCostType(r.tenant, costType)
}

func (r *boundAnnualStatementCostTypeRepository) List() []AnnualStatementCostType {
	return r.storage.listAnnualStatementCostTypes(r.tenant)
}

var annualStatementCostTypeKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

var defaultAnnualStatementCostTypes = []AnnualStatementCostType{
	{Key: "grundsteuer", Name: "Grundsteuer", Allocatable: true, AllocationKey: AllocationKeyNutzwert, VATRatePercent: 10},
	{Key: "muellabfuhr", Name: "Müllabfuhr", Allocatable: true, AllocationKey: AllocationKeyNutzwert, VATRatePercent: 10},
	{Key: "hausbetreuung", Name: "Hausbetreuung", Allocatable: true, AllocationKey: AllocationKeyNutzwert, VATRatePercent: 10},
	{Key: "gebaeudeversicherung", Name: "Gebäudeversicherung", Allocatable: true, AllocationKey: AllocationKeyNutzwert, VATRatePercent: 10},
	{Key: "gartenpflege", Name: "Gartenpflege", Allocatable: true, AllocationKey: AllocationKeyNutzwert, VATRatePercent: 10},
}

// AnnualStatementDefaultCostTypes returns the read-only starter catalogue for
// presentation before a manager creates the first period. Persisting these
// defaults is an explicit write-path concern; rendering must not mutate state.
func AnnualStatementDefaultCostTypes(updatedBy string) []AnnualStatementCostType {
	updatedBy = strings.ToLower(strings.TrimSpace(updatedBy))
	out := append([]AnnualStatementCostType(nil), defaultAnnualStatementCostTypes...)
	for index := range out {
		out[index].UpdatedBy = updatedBy
	}
	return out
}

// MemoryAnnualStatementCostTypeStore is used by isolated server tests.
// Production uses SQLAnnualStatementCostTypeStore.
type MemoryAnnualStatementCostTypeStore struct {
	mu     sync.Mutex
	byHome map[string]map[string]AnnualStatementCostType
}

func NewMemoryAnnualStatementCostTypeStore() *MemoryAnnualStatementCostTypeStore {
	return &MemoryAnnualStatementCostTypeStore{byHome: map[string]map[string]AnnualStatementCostType{}}
}

func (*MemoryAnnualStatementCostTypeStore) annualStatementCostTypeStorage() {}

func (s *MemoryAnnualStatementCostTypeStore) ensureAnnualStatementCostTypeDefaults(tenant TenantRef, updatedBy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byHome == nil {
		s.byHome = map[string]map[string]AnnualStatementCostType{}
	}
	if s.byHome[tenant.ID] == nil {
		s.byHome[tenant.ID] = map[string]AnnualStatementCostType{}
	}
	for _, starter := range defaultAnnualStatementCostTypes {
		if _, exists := s.byHome[tenant.ID][starter.Key]; exists {
			continue
		}
		starter.UpdatedAt = time.Now().UTC()
		starter.UpdatedBy = strings.ToLower(strings.TrimSpace(updatedBy))
		s.byHome[tenant.ID][starter.Key] = starter
	}
	return nil
}

func (s *MemoryAnnualStatementCostTypeStore) saveAnnualStatementCostType(tenant TenantRef, costType AnnualStatementCostType) (AnnualStatementCostType, error) {
	costType = normalizeAnnualStatementCostType(costType)
	if err := validateAnnualStatementCostType(costType); err != nil {
		return AnnualStatementCostType{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byHome == nil {
		s.byHome = map[string]map[string]AnnualStatementCostType{}
	}
	if s.byHome[tenant.ID] == nil {
		s.byHome[tenant.ID] = map[string]AnnualStatementCostType{}
	}
	s.byHome[tenant.ID][costType.Key] = costType
	return costType, nil
}

func (s *MemoryAnnualStatementCostTypeStore) listAnnualStatementCostTypes(tenant TenantRef) []AnnualStatementCostType {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]AnnualStatementCostType, 0, len(s.byHome[tenant.ID]))
	for _, costType := range s.byHome[tenant.ID] {
		out = append(out, costType)
	}
	sortAnnualStatementCostTypes(out)
	return out
}

// SQLAnnualStatementCostTypeStore persists catalogue entries in
// annual_statement_cost_types.
type SQLAnnualStatementCostTypeStore struct {
	db *TenantDB
}

func NewSQLAnnualStatementCostTypeStore(db *TenantDB) *SQLAnnualStatementCostTypeStore {
	return &SQLAnnualStatementCostTypeStore{db: db}
}

func (*SQLAnnualStatementCostTypeStore) annualStatementCostTypeStorage() {}

func (s *SQLAnnualStatementCostTypeStore) ensureAnnualStatementCostTypeDefaults(tenant TenantRef, updatedBy string) error {
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	updatedBy = strings.ToLower(strings.TrimSpace(updatedBy))
	for _, starter := range defaultAnnualStatementCostTypes {
		if _, err := tx.Exec(
			`INSERT INTO annual_statement_cost_types(tenant_id, tenant_slug, key, name, allocatable, allocation_key, updated_at, updated_by)
			 VALUES($1,$2,$3,$4,$5,$6,$7,$8)
			 ON CONFLICT(tenant_slug, key) DO NOTHING`,
			tenant.ID, tenant.Slug, starter.Key, starter.Name, starter.Allocatable, starter.AllocationKey, now, updatedBy,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLAnnualStatementCostTypeStore) saveAnnualStatementCostType(tenant TenantRef, costType AnnualStatementCostType) (AnnualStatementCostType, error) {
	costType = normalizeAnnualStatementCostType(costType)
	if err := validateAnnualStatementCostType(costType); err != nil {
		return AnnualStatementCostType{}, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return AnnualStatementCostType{}, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(
		`UPDATE annual_statement_cost_types
		 SET name=$1, allocatable=$2, allocation_key=$3, updated_at=$4, updated_by=$5
		 WHERE tenant_id=$6 AND key=$7`,
		costType.Name, costType.Allocatable, costType.AllocationKey, costType.UpdatedAt.Format(time.RFC3339Nano), costType.UpdatedBy,
		tenant.ID, costType.Key,
	)
	if err != nil {
		return AnnualStatementCostType{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return AnnualStatementCostType{}, err
	}
	if affected == 0 {
		if _, err := tx.Exec(
			`INSERT INTO annual_statement_cost_types(tenant_id, tenant_slug, key, name, allocatable, allocation_key, updated_at, updated_by)
			 VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
			tenant.ID, tenant.Slug, costType.Key, costType.Name, costType.Allocatable, costType.AllocationKey,
			costType.UpdatedAt.Format(time.RFC3339Nano), costType.UpdatedBy,
		); err != nil {
			return AnnualStatementCostType{}, err
		}
	}
	return costType, tx.Commit()
}

func (s *SQLAnnualStatementCostTypeStore) listAnnualStatementCostTypes(tenant TenantRef) []AnnualStatementCostType {
	rows, err := s.db.For(tenant).Query(
		`SELECT key, name, allocatable, allocation_key, updated_at, updated_by
		 FROM annual_statement_cost_types WHERE tenant_id=$1 ORDER BY name, key`, tenant.ID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []AnnualStatementCostType{}
	for rows.Next() {
		var costType AnnualStatementCostType
		var updatedAt string
		if err := rows.Scan(&costType.Key, &costType.Name, &costType.Allocatable, &costType.AllocationKey, &updatedAt, &costType.UpdatedBy); err != nil {
			continue
		}
		costType.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		out = append(out, costType)
	}
	return out
}

func normalizeAnnualStatementCostType(costType AnnualStatementCostType) AnnualStatementCostType {
	costType.Key = strings.ToLower(strings.TrimSpace(costType.Key))
	costType.Name = strings.TrimSpace(costType.Name)
	costType.AllocationKey = strings.ToLower(strings.TrimSpace(costType.AllocationKey))
	if !costType.Allocatable {
		costType.AllocationKey = ""
	}
	costType.UpdatedBy = strings.ToLower(strings.TrimSpace(costType.UpdatedBy))
	if costType.UpdatedAt.IsZero() {
		costType.UpdatedAt = time.Now().UTC()
	} else {
		costType.UpdatedAt = costType.UpdatedAt.UTC()
	}
	return costType
}

func validateAnnualStatementCostType(costType AnnualStatementCostType) error {
	if !annualStatementCostTypeKeyPattern.MatchString(costType.Key) {
		return fmt.Errorf("invalid annual statement cost type key")
	}
	if costType.Name == "" || len([]rune(costType.Name)) > 120 {
		return fmt.Errorf("invalid annual statement cost type name")
	}
	if costType.Allocatable && !ValidAllocationKey(costType.AllocationKey) {
		return fmt.Errorf("allocatable annual statement cost type needs one allocation key")
	}
	return nil
}

func sortAnnualStatementCostTypes(costTypes []AnnualStatementCostType) {
	sort.Slice(costTypes, func(i, j int) bool {
		left, right := strings.ToLower(costTypes[i].Name), strings.ToLower(costTypes[j].Name)
		if left == right {
			return costTypes[i].Key < costTypes[j].Key
		}
		return left < right
	})
}

var _ AnnualStatementCostTypeStorage = (*MemoryAnnualStatementCostTypeStore)(nil)
var _ AnnualStatementCostTypeStorage = (*SQLAnnualStatementCostTypeStore)(nil)
