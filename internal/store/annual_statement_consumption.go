package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	ConsumptionSourceEntity = "entity"
	ConsumptionSourceAsset  = "asset"
)

type AnnualStatementConsumptionGapReason string

const (
	ConsumptionGapMissingSourceMapping     AnnualStatementConsumptionGapReason = "missing-source-mapping"
	ConsumptionGapMissingStartEvidence     AnnualStatementConsumptionGapReason = "missing-start-evidence"
	ConsumptionGapMissingEndEvidence       AnnualStatementConsumptionGapReason = "missing-end-evidence"
	ConsumptionGapAmbiguousSource          AnnualStatementConsumptionGapReason = "ambiguous-source"
	ConsumptionGapAmbiguousMeasurementUnit AnnualStatementConsumptionGapReason = "ambiguous-measurement-unit"
	ConsumptionGapCounterReset             AnnualStatementConsumptionGapReason = "counter-reset"
)

var (
	ErrAnnualStatementConsumptionConflict        = errors.New("annual statement consumption evidence conflict")
	ErrAnnualStatementConsumptionInvalidEvidence = errors.New("invalid annual statement consumption evidence")
	ErrAnnualStatementConsumptionInvalidQuery    = errors.New("invalid annual statement consumption query")
)

// AnnualStatementConsumptionEvidence is one immutable cumulative meter fact.
// SourceKey is derived by the store from source identity and measurement time;
// it deliberately excludes the reported value, unit and annual-statement
// mapping so a changed retry conflicts instead of creating a second fact.
type AnnualStatementConsumptionEvidence struct {
	SourceKey       string    `json:"source_key"`
	UnitID          string    `json:"unit_id"`
	CostTypeKey     string    `json:"cost_type_key"`
	SourceKind      string    `json:"source_kind"`
	SourceID        string    `json:"source_id"`
	MeasuredAt      time.Time `json:"measured_at"`
	ValueMicros     int64     `json:"value_micros"`
	MeasurementUnit string    `json:"measurement_unit"`
	ReceivedAt      time.Time `json:"received_at"`
}

type AnnualStatementUnitConsumption struct {
	UnitID          string `json:"unit_id"`
	ValueMicros     int64  `json:"value_micros"`
	MeasurementUnit string `json:"measurement_unit"`
}

// AnnualStatementConsumptionGap is intentionally value-free. A caller may
// explain why a vector is unavailable without disclosing a disputed reading.
type AnnualStatementConsumptionGap struct {
	UnitID string                              `json:"unit_id"`
	Reason AnnualStatementConsumptionGapReason `json:"reason"`
}

// AnnualStatementConsumptionVector is a derived, cost-type-specific input for
// a later annual-statement run. When any expected unit has a gap, Units is empty
// so a partial vector cannot be mistaken for a complete allocation basis.
type AnnualStatementConsumptionVector struct {
	PeriodYear  int                              `json:"period_year"`
	CostTypeKey string                           `json:"cost_type_key"`
	Units       []AnnualStatementUnitConsumption `json:"units,omitempty"`
	Gaps        []AnnualStatementConsumptionGap  `json:"gaps,omitempty"`
}

type AnnualStatementConsumptionRepository interface {
	Append(evidence AnnualStatementConsumptionEvidence) (stored AnnualStatementConsumptionEvidence, inserted bool, err error)
	ConsumptionVector(period AnnualStatementPeriod, costTypeKey string, expectedUnitIDs []string, location *time.Location) (AnnualStatementConsumptionVector, error)
}

type AnnualStatementConsumptionStorage interface {
	annualStatementConsumptionStorage()
}

type annualStatementConsumptionBackend interface {
	appendAnnualStatementConsumption(tenant TenantRef, evidence AnnualStatementConsumptionEvidence) (AnnualStatementConsumptionEvidence, bool, error)
	listAnnualStatementConsumption(tenant TenantRef, costTypeKey string, from, through time.Time) ([]AnnualStatementConsumptionEvidence, error)
}

type boundAnnualStatementConsumptionRepository struct {
	storage annualStatementConsumptionBackend
	tenant  TenantRef
}

func BindAnnualStatementConsumptionRepository(storage AnnualStatementConsumptionStorage, tenant TenantRef) (AnnualStatementConsumptionRepository, bool) {
	backend, backendOK := storage.(annualStatementConsumptionBackend)
	resolved, tenantOK := validTenantRef(tenant)
	if !backendOK || !tenantOK {
		return nil, false
	}
	return &boundAnnualStatementConsumptionRepository{storage: backend, tenant: resolved}, true
}

func (r *boundAnnualStatementConsumptionRepository) Append(evidence AnnualStatementConsumptionEvidence) (AnnualStatementConsumptionEvidence, bool, error) {
	return r.storage.appendAnnualStatementConsumption(r.tenant, evidence)
}

func (r *boundAnnualStatementConsumptionRepository) ConsumptionVector(period AnnualStatementPeriod, costTypeKey string, expectedUnitIDs []string, location *time.Location) (AnnualStatementConsumptionVector, error) {
	costTypeKey = strings.ToLower(strings.TrimSpace(costTypeKey))
	start, endExclusive, expected, err := annualStatementConsumptionQuery(period, costTypeKey, expectedUnitIDs, location)
	if err != nil {
		return AnnualStatementConsumptionVector{}, err
	}
	evidence, err := r.storage.listAnnualStatementConsumption(r.tenant, costTypeKey, start, endExclusive)
	if err != nil {
		return AnnualStatementConsumptionVector{}, err
	}
	return buildAnnualStatementConsumptionVector(period.Year, costTypeKey, expected, start, endExclusive, evidence), nil
}

func annualStatementConsumptionQuery(period AnnualStatementPeriod, costTypeKey string, expectedUnitIDs []string, location *time.Location) (time.Time, time.Time, []string, error) {
	if location == nil || !annualStatementCostTypeKeyPattern.MatchString(costTypeKey) || validateAnnualStatementPeriod(period) != nil || len(expectedUnitIDs) == 0 {
		return time.Time{}, time.Time{}, nil, ErrAnnualStatementConsumptionInvalidQuery
	}
	start, startErr := time.ParseInLocation("2006-01-02", period.StartsOn, location)
	lastDay, endErr := time.ParseInLocation("2006-01-02", period.EndsOn, location)
	if startErr != nil || endErr != nil {
		return time.Time{}, time.Time{}, nil, ErrAnnualStatementConsumptionInvalidQuery
	}
	seen := map[string]bool{}
	expected := make([]string, 0, len(expectedUnitIDs))
	for _, raw := range expectedUnitIDs {
		unitID := NormalizeUnitID(raw)
		if unitID == "" || seen[unitID] {
			return time.Time{}, time.Time{}, nil, ErrAnnualStatementConsumptionInvalidQuery
		}
		seen[unitID] = true
		expected = append(expected, unitID)
	}
	sort.Strings(expected)
	endExclusive := lastDay.AddDate(0, 0, 1)
	if !annualStatementConsumptionUnixNanoRepresentable(start) || !annualStatementConsumptionUnixNanoRepresentable(endExclusive) {
		return time.Time{}, time.Time{}, nil, ErrAnnualStatementConsumptionInvalidQuery
	}
	return start, endExclusive, expected, nil
}

func buildAnnualStatementConsumptionVector(periodYear int, costTypeKey string, expected []string, start, endExclusive time.Time, evidence []AnnualStatementConsumptionEvidence) AnnualStatementConsumptionVector {
	vector := AnnualStatementConsumptionVector{PeriodYear: periodYear, CostTypeKey: costTypeKey}
	byUnit := map[string][]AnnualStatementConsumptionEvidence{}
	for _, item := range evidence {
		byUnit[item.UnitID] = append(byUnit[item.UnitID], item)
	}
	for _, unitID := range expected {
		items := byUnit[unitID]
		if len(items) == 0 {
			vector.Gaps = append(vector.Gaps, AnnualStatementConsumptionGap{UnitID: unitID, Reason: ConsumptionGapMissingSourceMapping})
			continue
		}
		sources := map[string]bool{}
		units := map[string]bool{}
		for _, item := range items {
			sources[item.SourceKind+"\x00"+item.SourceID] = true
			units[item.MeasurementUnit] = true
		}
		if len(sources) != 1 {
			vector.Gaps = append(vector.Gaps, AnnualStatementConsumptionGap{UnitID: unitID, Reason: ConsumptionGapAmbiguousSource})
			continue
		}
		if len(units) != 1 || units[""] {
			vector.Gaps = append(vector.Gaps, AnnualStatementConsumptionGap{UnitID: unitID, Reason: ConsumptionGapAmbiguousMeasurementUnit})
			continue
		}
		sort.Slice(items, func(i, j int) bool {
			if items[i].MeasuredAt.Equal(items[j].MeasuredAt) {
				return items[i].SourceKey < items[j].SourceKey
			}
			return items[i].MeasuredAt.Before(items[j].MeasuredAt)
		})
		startIndex, endIndex := -1, -1
		reset := false
		for index, item := range items {
			if item.MeasuredAt.Equal(start) {
				startIndex = index
			}
			if item.MeasuredAt.Equal(endExclusive) {
				endIndex = index
			}
			if index > 0 && item.ValueMicros < items[index-1].ValueMicros {
				reset = true
			}
		}
		if startIndex < 0 {
			vector.Gaps = append(vector.Gaps, AnnualStatementConsumptionGap{UnitID: unitID, Reason: ConsumptionGapMissingStartEvidence})
			continue
		}
		if endIndex < 0 {
			vector.Gaps = append(vector.Gaps, AnnualStatementConsumptionGap{UnitID: unitID, Reason: ConsumptionGapMissingEndEvidence})
			continue
		}
		if reset || items[endIndex].ValueMicros < items[startIndex].ValueMicros {
			vector.Gaps = append(vector.Gaps, AnnualStatementConsumptionGap{UnitID: unitID, Reason: ConsumptionGapCounterReset})
			continue
		}
		vector.Units = append(vector.Units, AnnualStatementUnitConsumption{
			UnitID: unitID, ValueMicros: items[endIndex].ValueMicros - items[startIndex].ValueMicros,
			MeasurementUnit: items[startIndex].MeasurementUnit,
		})
	}
	if len(vector.Gaps) == 0 {
		measurementUnits := map[string]bool{}
		for _, unit := range vector.Units {
			measurementUnits[unit.MeasurementUnit] = true
		}
		if len(measurementUnits) != 1 {
			vector.Gaps = make([]AnnualStatementConsumptionGap, 0, len(expected))
			for _, unitID := range expected {
				vector.Gaps = append(vector.Gaps, AnnualStatementConsumptionGap{UnitID: unitID, Reason: ConsumptionGapAmbiguousMeasurementUnit})
			}
		}
	}
	if len(vector.Gaps) > 0 {
		vector.Units = nil
	}
	return vector
}

type MemoryAnnualStatementConsumptionStore struct {
	mu      sync.Mutex
	records map[string]annualStatementConsumptionRecord
}

func NewMemoryAnnualStatementConsumptionStore() *MemoryAnnualStatementConsumptionStore {
	return &MemoryAnnualStatementConsumptionStore{records: map[string]annualStatementConsumptionRecord{}}
}

func (*MemoryAnnualStatementConsumptionStore) annualStatementConsumptionStorage() {}

func (s *MemoryAnnualStatementConsumptionStore) appendAnnualStatementConsumption(tenant TenantRef, evidence AnnualStatementConsumptionEvidence) (AnnualStatementConsumptionEvidence, bool, error) {
	evidence, err := normalizeAnnualStatementConsumptionEvidence(evidence)
	if err != nil {
		return AnnualStatementConsumptionEvidence{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := tenant.ID + "\x00" + evidence.SourceKey
	if existing, found := s.records[key]; found {
		return repeatedAnnualStatementConsumption(existing.Evidence, evidence)
	}
	s.records[key] = annualStatementConsumptionRecord{TenantID: tenant.ID, TenantSlug: tenant.Slug, Evidence: evidence}
	return evidence, true, nil
}

func (s *MemoryAnnualStatementConsumptionStore) listAnnualStatementConsumption(tenant TenantRef, costTypeKey string, from, through time.Time) ([]AnnualStatementConsumptionEvidence, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []AnnualStatementConsumptionEvidence{}
	for _, record := range s.records {
		item := record.Evidence
		if record.TenantID == tenant.ID && item.CostTypeKey == costTypeKey && !item.MeasuredAt.Before(from) && !item.MeasuredAt.After(through) {
			out = append(out, item)
		}
	}
	return out, nil
}

type annualStatementConsumptionRecord struct {
	TenantID   string                             `json:"tenant_id"`
	TenantSlug string                             `json:"tenant_slug"`
	Evidence   AnnualStatementConsumptionEvidence `json:"evidence"`
}

type annualStatementConsumptionJSONData struct {
	Evidence []annualStatementConsumptionRecord `json:"evidence"`
}

type JSONAnnualStatementConsumptionStore struct {
	mu   sync.Mutex
	path string
	data annualStatementConsumptionJSONData
}

func NewJSONAnnualStatementConsumptionStore(path string) (*JSONAnnualStatementConsumptionStore, error) {
	store := &JSONAnnualStatementConsumptionStore{path: path, data: annualStatementConsumptionJSONData{Evidence: []annualStatementConsumptionRecord{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read annual statement consumption data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("could not decode annual statement consumption data")
	}
	seen := map[string]bool{}
	for index, record := range store.data.Evidence {
		tenant, ok := validTenantRef(TenantRef{ID: record.TenantID, Slug: record.TenantSlug})
		persisted := record.Evidence
		if persisted.ReceivedAt.IsZero() {
			return nil, fmt.Errorf("%w: invalid persisted data", ErrAnnualStatementConsumptionInvalidEvidence)
		}
		normalized, normalizeErr := normalizeAnnualStatementConsumptionEvidence(record.Evidence)
		key := tenant.ID + "\x00" + normalized.SourceKey
		if !ok || normalizeErr != nil || record.TenantID != tenant.ID || record.TenantSlug != tenant.Slug ||
			persisted != normalized || seen[key] {
			return nil, fmt.Errorf("%w: invalid persisted data", ErrAnnualStatementConsumptionInvalidEvidence)
		}
		seen[key] = true
		store.data.Evidence[index] = annualStatementConsumptionRecord{TenantID: tenant.ID, TenantSlug: tenant.Slug, Evidence: normalized}
	}
	return store, nil
}

func (*JSONAnnualStatementConsumptionStore) annualStatementConsumptionStorage() {}

func (s *JSONAnnualStatementConsumptionStore) appendAnnualStatementConsumption(tenant TenantRef, evidence AnnualStatementConsumptionEvidence) (AnnualStatementConsumptionEvidence, bool, error) {
	evidence, err := normalizeAnnualStatementConsumptionEvidence(evidence)
	if err != nil {
		return AnnualStatementConsumptionEvidence{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, record := range s.data.Evidence {
		if record.TenantID == tenant.ID && record.Evidence.SourceKey == evidence.SourceKey {
			return repeatedAnnualStatementConsumption(record.Evidence, evidence)
		}
	}
	s.data.Evidence = append(s.data.Evidence, annualStatementConsumptionRecord{TenantID: tenant.ID, TenantSlug: tenant.Slug, Evidence: evidence})
	if err := SaveJSONAtomic(s.path, s.data, "annual statement consumption"); err != nil {
		s.data.Evidence = s.data.Evidence[:len(s.data.Evidence)-1]
		return AnnualStatementConsumptionEvidence{}, false, err
	}
	return evidence, true, nil
}

func (s *JSONAnnualStatementConsumptionStore) listAnnualStatementConsumption(tenant TenantRef, costTypeKey string, from, through time.Time) ([]AnnualStatementConsumptionEvidence, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []AnnualStatementConsumptionEvidence{}
	for _, record := range s.data.Evidence {
		item := record.Evidence
		if record.TenantID == tenant.ID && item.CostTypeKey == costTypeKey && !item.MeasuredAt.Before(from) && !item.MeasuredAt.After(through) {
			out = append(out, item)
		}
	}
	return out, nil
}

type SQLAnnualStatementConsumptionStore struct{ db *TenantDB }

func NewSQLAnnualStatementConsumptionStore(db *TenantDB) *SQLAnnualStatementConsumptionStore {
	return &SQLAnnualStatementConsumptionStore{db: db}
}

func (*SQLAnnualStatementConsumptionStore) annualStatementConsumptionStorage() {}

func (s *SQLAnnualStatementConsumptionStore) appendAnnualStatementConsumption(tenant TenantRef, evidence AnnualStatementConsumptionEvidence) (AnnualStatementConsumptionEvidence, bool, error) {
	if s == nil || s.db == nil {
		return AnnualStatementConsumptionEvidence{}, false, fmt.Errorf("annual statement consumption store unavailable")
	}
	evidence, err := normalizeAnnualStatementConsumptionEvidence(evidence)
	if err != nil {
		return AnnualStatementConsumptionEvidence{}, false, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return AnnualStatementConsumptionEvidence{}, false, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`INSERT INTO annual_statement_consumption_evidence
		(tenant_id,tenant_slug,source_key,unit_id,cost_type_key,source_kind,source_id,measured_at_ns,value_micros,measurement_unit,received_at_ns)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT(tenant_id,source_key) DO NOTHING`,
		tenant.ID, tenant.Slug, evidence.SourceKey, evidence.UnitID, evidence.CostTypeKey,
		evidence.SourceKind, evidence.SourceID, evidence.MeasuredAt.UnixNano(), evidence.ValueMicros,
		evidence.MeasurementUnit, evidence.ReceivedAt.UnixNano())
	if err != nil {
		return AnnualStatementConsumptionEvidence{}, false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return AnnualStatementConsumptionEvidence{}, false, err
	}
	if affected == 1 {
		return evidence, true, tx.Commit()
	}
	existing, err := scanAnnualStatementConsumption(tx.QueryRow(
		`SELECT source_key,unit_id,cost_type_key,source_kind,source_id,measured_at_ns,value_micros,measurement_unit,received_at_ns
		 FROM annual_statement_consumption_evidence WHERE tenant_id=$1 AND source_key=$2`, tenant.ID, evidence.SourceKey))
	if err != nil {
		return AnnualStatementConsumptionEvidence{}, false, err
	}
	stored, inserted, err := repeatedAnnualStatementConsumption(existing, evidence)
	if err != nil {
		return AnnualStatementConsumptionEvidence{}, false, err
	}
	return stored, inserted, tx.Commit()
}

func (s *SQLAnnualStatementConsumptionStore) listAnnualStatementConsumption(tenant TenantRef, costTypeKey string, from, through time.Time) ([]AnnualStatementConsumptionEvidence, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("annual statement consumption store unavailable")
	}
	rows, err := s.db.For(tenant).Query(
		`SELECT source_key,unit_id,cost_type_key,source_kind,source_id,measured_at_ns,value_micros,measurement_unit,received_at_ns
		 FROM annual_statement_consumption_evidence
		 WHERE tenant_id=$1 AND cost_type_key=$2 AND measured_at_ns >= $3 AND measured_at_ns <= $4
		 ORDER BY unit_id,source_kind,source_id,measured_at_ns,source_key`,
		tenant.ID, costTypeKey, from.UnixNano(), through.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AnnualStatementConsumptionEvidence{}
	for rows.Next() {
		item, err := scanAnnualStatementConsumption(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

type annualStatementConsumptionScanner interface {
	Scan(dest ...any) error
}

func scanAnnualStatementConsumption(scanner annualStatementConsumptionScanner) (AnnualStatementConsumptionEvidence, error) {
	var item AnnualStatementConsumptionEvidence
	var measuredAtNS, receivedAtNS int64
	if err := scanner.Scan(&item.SourceKey, &item.UnitID, &item.CostTypeKey, &item.SourceKind, &item.SourceID,
		&measuredAtNS, &item.ValueMicros, &item.MeasurementUnit, &receivedAtNS); err != nil {
		return AnnualStatementConsumptionEvidence{}, err
	}
	item.MeasuredAt = time.Unix(0, measuredAtNS).UTC()
	item.ReceivedAt = time.Unix(0, receivedAtNS).UTC()
	return item, nil
}

func normalizeAnnualStatementConsumptionEvidence(evidence AnnualStatementConsumptionEvidence) (AnnualStatementConsumptionEvidence, error) {
	evidence.UnitID = NormalizeUnitID(evidence.UnitID)
	evidence.CostTypeKey = strings.ToLower(strings.TrimSpace(evidence.CostTypeKey))
	evidence.SourceKind = strings.ToLower(strings.TrimSpace(evidence.SourceKind))
	evidence.SourceID = strings.ToLower(strings.TrimSpace(evidence.SourceID))
	evidence.MeasurementUnit = canonicalConsumptionMeasurementUnit(evidence.MeasurementUnit)
	evidence.MeasuredAt = evidence.MeasuredAt.UTC()
	if evidence.ReceivedAt.IsZero() {
		evidence.ReceivedAt = time.Now().UTC()
	} else {
		evidence.ReceivedAt = evidence.ReceivedAt.UTC()
	}
	if evidence.UnitID == "" || !annualStatementCostTypeKeyPattern.MatchString(evidence.CostTypeKey) ||
		(evidence.SourceKind != ConsumptionSourceEntity && evidence.SourceKind != ConsumptionSourceAsset) ||
		evidence.SourceID == "" || len(evidence.SourceID) > 255 || evidence.MeasuredAt.IsZero() || evidence.ValueMicros < 0 ||
		!annualStatementConsumptionUnixNanoRepresentable(evidence.MeasuredAt) || !annualStatementConsumptionUnixNanoRepresentable(evidence.ReceivedAt) {
		return AnnualStatementConsumptionEvidence{}, ErrAnnualStatementConsumptionInvalidEvidence
	}
	evidence.SourceKey = annualStatementConsumptionSourceKey(evidence)
	return evidence, nil
}

var (
	annualStatementConsumptionUnixNanoMinimum = time.Unix(0, -1<<63).UTC()
	annualStatementConsumptionUnixNanoMaximum = time.Unix(0, 1<<63-1).UTC()
)

func annualStatementConsumptionUnixNanoRepresentable(value time.Time) bool {
	value = value.UTC()
	return !value.IsZero() && !value.Before(annualStatementConsumptionUnixNanoMinimum) && !value.After(annualStatementConsumptionUnixNanoMaximum)
}

func canonicalConsumptionMeasurementUnit(raw string) string {
	raw = strings.TrimSpace(raw)
	switch strings.ToLower(strings.ReplaceAll(raw, " ", "")) {
	case "wh":
		return "Wh"
	case "kwh":
		return "kWh"
	case "mwh":
		return "MWh"
	case "m3", "m^3", "m³":
		return "m³"
	case "l", "liter", "litre":
		return "l"
	default:
		return raw
	}
}

func annualStatementConsumptionSourceKey(evidence AnnualStatementConsumptionEvidence) string {
	identity := evidence.SourceKind + "\x00" + evidence.SourceID + "\x00" + evidence.MeasuredAt.Format(time.RFC3339Nano)
	digest := sha256.Sum256([]byte(identity))
	return hex.EncodeToString(digest[:])
}

func repeatedAnnualStatementConsumption(existing, incoming AnnualStatementConsumptionEvidence) (AnnualStatementConsumptionEvidence, bool, error) {
	if equalAnnualStatementConsumptionPayload(existing, incoming) {
		return existing, false, nil
	}
	return AnnualStatementConsumptionEvidence{}, false, fmt.Errorf("%w: source key %s", ErrAnnualStatementConsumptionConflict, incoming.SourceKey)
}

func equalAnnualStatementConsumptionPayload(left, right AnnualStatementConsumptionEvidence) bool {
	return left.SourceKey == right.SourceKey && left.UnitID == right.UnitID && left.CostTypeKey == right.CostTypeKey &&
		left.SourceKind == right.SourceKind && left.SourceID == right.SourceID && left.MeasuredAt.Equal(right.MeasuredAt) &&
		left.ValueMicros == right.ValueMicros && left.MeasurementUnit == right.MeasurementUnit
}

var _ AnnualStatementConsumptionStorage = (*MemoryAnnualStatementConsumptionStore)(nil)
var _ AnnualStatementConsumptionStorage = (*JSONAnnualStatementConsumptionStore)(nil)
var _ AnnualStatementConsumptionStorage = (*SQLAnnualStatementConsumptionStore)(nil)
