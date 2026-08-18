package energy

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/tenantid"
)

type Storage interface {
	ForHome(homeKey string) Storage
	// ForTenant binds the identity the caller was authorized for. The SQL
	// implementation filters every tenant-scoped statement on that immutable
	// id; the in-memory one keys on the slug and simply keeps the reference.
	ForTenant(tenant store.TenantRef) Storage
	ListProfiles(tenantSlug string) ([]HomeProfile, error)
	Profile(tenantSlug string) (HomeProfile, bool, error)
	SaveProfile(profile HomeProfile) error
	ListAssets(tenantSlug string) ([]Asset, error)
	UpsertAsset(asset Asset) error
	// UpdateAssetPriorities writes the rail order (metadata key "priority",
	// 1-based) in one atomic step. It returns false without writing when any
	// id no longer belongs to this tenant's assets.
	UpdateAssetPriorities(tenantSlug string, order []string) (bool, error)
	DeleteAsset(tenantSlug, id string) (bool, error)
	ListMappings(tenantSlug string) ([]EntityMapping, error)
	UpsertMapping(mapping EntityMapping) error
	DeleteMapping(tenantSlug, id string) (bool, error)
	PutInterval(interval Interval) error
	ListIntervals(tenantSlug string, from, to time.Time) ([]Interval, error)
	PutImport(record ImportRecord, intervals []Interval) (bool, error)
	ListImports(tenantSlug string) ([]ImportRecord, error)
	ListImportsForExport(tenantSlug string) ([]ImportRecord, error)
	DeleteMeasurementData(tenantSlug string) (DeleteSummary, error)
	DeleteProfile(tenantSlug string) (DeleteSummary, error)
	PurgeExpired(rawImportBefore, intervalBefore, assessmentBefore time.Time) (DeleteSummary, error)
	ListMaintenance(tenantSlug string) ([]MaintenancePlan, error)
	UpsertMaintenance(plan MaintenancePlan) error
	DeleteMaintenance(tenantSlug, id string) (bool, error)
	SaveTariffAssessment(item TariffAssessment) error
	ListTariffAssessments(tenantSlug string) ([]TariffAssessment, error)
	UpsertMeasure(item Measure) error
	GetMeasure(tenantSlug, id string) (Measure, bool, error)
	ListMeasures(tenantSlug string) ([]Measure, error)
}

// DeleteSummary makes destructive and automatic lifecycle operations
// inspectable without exposing the deleted values themselves.
type DeleteSummary struct {
	Profiles          int
	Assets            int
	Mappings          int
	Intervals         int
	Imports           int
	Maintenance       int
	TariffAssessments int
	Measures          int
}

type MemoryStore struct {
	state   *memoryStoreState
	homeKey string
	tenant  store.TenantRef
}

type memoryStoreState struct {
	mu          sync.Mutex
	profiles    map[string]HomeProfile
	assets      map[string]Asset
	mappings    map[string]EntityMapping
	intervals   map[string]Interval
	imports     map[string]ImportRecord
	maintenance map[string]MaintenancePlan
	assessments map[string]TariffAssessment
	measures    map[string]Measure
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{state: &memoryStoreState{
		profiles:    map[string]HomeProfile{},
		assets:      map[string]Asset{},
		mappings:    map[string]EntityMapping{},
		intervals:   map[string]Interval{},
		imports:     map[string]ImportRecord{},
		maintenance: map[string]MaintenancePlan{},
		assessments: map[string]TariffAssessment{},
		measures:    map[string]Measure{},
	}}
}

func (s *MemoryStore) ForHome(homeKey string) Storage {
	return &MemoryStore{state: s.state, homeKey: NormalizeHomeKey(homeKey), tenant: s.tenant}
}

// ForTenant records the reference so a memory-backed store answers the same
// authorization question the SQL one does. The map keys stay slug-based — that
// is what the in-memory implementation IS — so the reference is remembered
// rather than used as a key.
func (s *MemoryStore) ForTenant(tenant store.TenantRef) Storage {
	return &MemoryStore{state: s.state, homeKey: s.homeKey, tenant: tenant}
}

func (s *MemoryStore) scopeHomeKey() string { return NormalizeHomeKey(s.homeKey) }

func (s *MemoryStore) ListProfiles(tenantSlug string) ([]HomeProfile, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	tenantSlug = normalizeSlug(tenantSlug)
	out := []HomeProfile{}
	for _, item := range s.state.profiles {
		if item.TenantSlug == tenantSlug {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].HomeKey < out[j].HomeKey })
	return out, nil
}

func (s *MemoryStore) Profile(tenantSlug string) (HomeProfile, bool, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	item, ok := s.state.profiles[normalizeSlug(tenantSlug)+"\x00"+s.scopeHomeKey()]
	return item, ok, nil
}

func (s *MemoryStore) SaveProfile(profile HomeProfile) error {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	profile.HomeKey = s.scopeHomeKey()
	profile = NormalizeProfile(profile, time.Now())
	if profile.TenantSlug == "" {
		return fmt.Errorf("energy: tenant required")
	}
	// The free-period timestamps are entitlement markers, not editable profile
	// content. Once set, re-onboarding or a stale client must not clear or
	// restart it. SQLStore enforces the same rule with COALESCE.
	key := profile.TenantSlug + "\x00" + profile.HomeKey
	if existing, ok := s.state.profiles[key]; ok {
		if existing.FreeStartedAt != nil {
			preserved := existing.FreeStartedAt.UTC()
			profile.FreeStartedAt = &preserved
		}
		if existing.FreeUntilAt != nil {
			preserved := existing.FreeUntilAt.UTC()
			profile.FreeUntilAt = &preserved
		}
	}
	s.state.profiles[key] = profile
	return nil
}

func (s *MemoryStore) ListAssets(tenantSlug string) ([]Asset, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	tenantSlug = normalizeSlug(tenantSlug)
	out := []Asset{}
	for _, item := range s.state.assets {
		if item.TenantSlug == tenantSlug && NormalizeHomeKey(item.HomeKey) == s.scopeHomeKey() {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind == out[j].Kind {
			return out[i].Name < out[j].Name
		}
		return out[i].Kind < out[j].Kind
	})
	return out, nil
}

func (s *MemoryStore) UpsertAsset(asset Asset) error {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	asset.HomeKey = s.scopeHomeKey()
	asset = NormalizeAsset(asset, time.Now())
	if asset.TenantSlug == "" {
		return fmt.Errorf("energy: tenant required")
	}
	// Asset-IDs sind global eindeutig, nicht nur je Haus: in SQLite ist `id`
	// Primärschlüssel, und der Upsert dort weist eine fremde ID mit
	// "asset id belongs to another tenant" ab. Ohne dieselbe Prüfung verhält
	// sich der Memory-Store abweichend, und Tests grün, wo Produktion bricht.
	for _, existing := range s.state.assets {
		if existing.ID == asset.ID && (existing.TenantSlug != asset.TenantSlug || NormalizeHomeKey(existing.HomeKey) != asset.HomeKey) {
			return ErrAssetIDTaken
		}
	}
	s.state.assets[asset.TenantSlug+"\x00"+asset.HomeKey+"\x00"+asset.ID] = asset
	return nil
}

func (s *MemoryStore) UpdateAssetPriorities(tenantSlug string, order []string) (bool, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	tenantSlug = normalizeSlug(tenantSlug)
	keys := make([]string, 0, len(order))
	for _, id := range order {
		key := tenantSlug + "\x00" + s.scopeHomeKey() + "\x00" + id
		if _, ok := s.state.assets[key]; !ok {
			return false, nil
		}
		keys = append(keys, key)
	}
	for position, key := range keys {
		asset := s.state.assets[key]
		metadata := map[string]string{}
		for name, value := range asset.Metadata {
			metadata[name] = value
		}
		metadata["priority"] = strconv.Itoa(position + 1)
		asset.Metadata = metadata
		asset.UpdatedAt = time.Now().UTC()
		s.state.assets[key] = asset
	}
	return true, nil
}

func (s *MemoryStore) DeleteAsset(tenantSlug, id string) (bool, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	tenantSlug = normalizeSlug(tenantSlug)
	id = strings.TrimSpace(id)
	key := tenantSlug + "\x00" + s.scopeHomeKey() + "\x00" + id
	if _, ok := s.state.assets[key]; !ok {
		return false, nil
	}
	delete(s.state.assets, key)
	for mappingKey, mapping := range s.state.mappings {
		if mapping.TenantSlug == tenantSlug && NormalizeHomeKey(mapping.HomeKey) == s.scopeHomeKey() && mapping.AssetID == id {
			mapping.AssetID = ""
			mapping.UpdatedAt = time.Now().UTC()
			s.state.mappings[mappingKey] = mapping
		}
	}
	for maintenanceKey, plan := range s.state.maintenance {
		if plan.TenantSlug == tenantSlug && NormalizeHomeKey(plan.HomeKey) == s.scopeHomeKey() && plan.AssetID == id {
			delete(s.state.maintenance, maintenanceKey)
		}
	}
	return true, nil
}

func (s *MemoryStore) ListMappings(tenantSlug string) ([]EntityMapping, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	tenantSlug = normalizeSlug(tenantSlug)
	out := []EntityMapping{}
	for _, item := range s.state.mappings {
		if item.TenantSlug == tenantSlug && NormalizeHomeKey(item.HomeKey) == s.scopeHomeKey() {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DisplayName < out[j].DisplayName })
	return out, nil
}

func (s *MemoryStore) UpsertMapping(mapping EntityMapping) error {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	mapping.HomeKey = s.scopeHomeKey()
	mapping = NormalizeMapping(mapping, time.Now())
	if mapping.TenantSlug == "" || mapping.EntityID == "" {
		return fmt.Errorf("energy: tenant and entity required")
	}
	if mapping.AssetID != "" {
		if _, ok := s.state.assets[mapping.TenantSlug+"\x00"+mapping.HomeKey+"\x00"+mapping.AssetID]; !ok {
			return ErrMappingAssetForeign
		}
	}
	// Entity id is the natural per-house key; rediscovery must update instead of
	// multiplying suggestions.
	for key, existing := range s.state.mappings {
		if existing.TenantSlug == mapping.TenantSlug && NormalizeHomeKey(existing.HomeKey) == mapping.HomeKey && existing.EntityID == mapping.EntityID {
			mapping.ID = existing.ID
			mapping.CreatedAt = existing.CreatedAt
			delete(s.state.mappings, key)
			break
		}
	}
	s.state.mappings[mapping.TenantSlug+"\x00"+mapping.HomeKey+"\x00"+mapping.ID] = mapping
	return nil
}

func (s *MemoryStore) DeleteMapping(tenantSlug, id string) (bool, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	key := normalizeSlug(tenantSlug) + "\x00" + s.scopeHomeKey() + "\x00" + strings.TrimSpace(id)
	if _, ok := s.state.mappings[key]; !ok {
		return false, nil
	}
	delete(s.state.mappings, key)
	return true, nil
}

func (s *MemoryStore) PutInterval(interval Interval) error {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	if interval.Duration <= 0 {
		interval.Duration = 15 * time.Minute
	}
	interval.TenantSlug = normalizeSlug(interval.TenantSlug)
	interval.HomeKey = s.scopeHomeKey()
	interval.StartsAt = interval.StartsAt.UTC()
	if interval.CreatedAt.IsZero() {
		interval.CreatedAt = time.Now().UTC()
	}
	if interval.Source == "" {
		interval.Source = "home-assistant"
	}
	if interval.Quality == "" {
		interval.Quality = "measured"
	}
	if interval.TenantSlug == "" || interval.StartsAt.IsZero() {
		return fmt.Errorf("energy: tenant and interval start required")
	}
	key := interval.TenantSlug + "\x00" + interval.HomeKey + "\x00" + interval.StartsAt.Format(time.RFC3339Nano) + "\x00" + interval.Source
	s.state.intervals[key] = interval
	return nil
}

func (s *MemoryStore) ListIntervals(tenantSlug string, from, to time.Time) ([]Interval, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	tenantSlug = normalizeSlug(tenantSlug)
	out := []Interval{}
	for _, item := range s.state.intervals {
		if item.TenantSlug != tenantSlug || NormalizeHomeKey(item.HomeKey) != s.scopeHomeKey() || item.StartsAt.Before(from) || !to.IsZero() && !item.StartsAt.Before(to) {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartsAt.Before(out[j].StartsAt) })
	return out, nil
}

func (s *MemoryStore) PutImport(record ImportRecord, intervals []Interval) (bool, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	record.TenantSlug = normalizeSlug(record.TenantSlug)
	record.HomeKey = s.scopeHomeKey()
	if record.TenantSlug == "" || record.SHA256 == "" {
		return false, fmt.Errorf("energy: tenant and import checksum required")
	}
	key := record.TenantSlug + "\x00" + record.HomeKey + "\x00" + record.SHA256
	if _, exists := s.state.imports[key]; exists {
		return false, nil
	}
	if record.ID == "" {
		record.ID = NewID("import")
	}
	if record.ImportedAt.IsZero() {
		record.ImportedAt = time.Now().UTC()
	}
	record.Payload = append([]byte(nil), record.Payload...)
	for _, interval := range intervals {
		if normalizeSlug(interval.TenantSlug) != record.TenantSlug {
			return false, fmt.Errorf("energy: interval tenant mismatch")
		}
	}
	s.state.imports[key] = record
	for _, interval := range intervals {
		if interval.Duration <= 0 {
			interval.Duration = 15 * time.Minute
		}
		interval.TenantSlug = record.TenantSlug
		interval.HomeKey = record.HomeKey
		interval.StartsAt = interval.StartsAt.UTC()
		if interval.Source == "" {
			interval.Source = "smart-meter"
		}
		if interval.Quality == "" {
			interval.Quality = QualityMeasured
		}
		intervalKey := interval.TenantSlug + "\x00" + interval.HomeKey + "\x00" + interval.StartsAt.Format(time.RFC3339Nano) + "\x00" + interval.Source
		s.state.intervals[intervalKey] = interval
	}
	return true, nil
}

func (s *MemoryStore) ListImports(tenantSlug string) ([]ImportRecord, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	tenantSlug = normalizeSlug(tenantSlug)
	out := []ImportRecord{}
	for _, record := range s.state.imports {
		if record.TenantSlug == tenantSlug && NormalizeHomeKey(record.HomeKey) == s.scopeHomeKey() {
			record.Payload = nil
			out = append(out, record)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ImportedAt.After(out[j].ImportedAt) })
	return out, nil
}

func (s *MemoryStore) ListImportsForExport(tenantSlug string) ([]ImportRecord, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	tenantSlug = normalizeSlug(tenantSlug)
	out := []ImportRecord{}
	for _, record := range s.state.imports {
		if record.TenantSlug == tenantSlug && NormalizeHomeKey(record.HomeKey) == s.scopeHomeKey() {
			record.Payload = append([]byte(nil), record.Payload...)
			out = append(out, record)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ImportedAt.After(out[j].ImportedAt) })
	return out, nil
}

func (s *MemoryStore) DeleteMeasurementData(tenantSlug string) (DeleteSummary, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	tenantSlug = normalizeSlug(tenantSlug)
	homeKey := s.scopeHomeKey()
	var summary DeleteSummary
	for key, item := range s.state.imports {
		if item.TenantSlug == tenantSlug && NormalizeHomeKey(item.HomeKey) == homeKey {
			delete(s.state.imports, key)
			summary.Imports++
		}
	}
	for key, item := range s.state.intervals {
		if item.TenantSlug == tenantSlug && NormalizeHomeKey(item.HomeKey) == homeKey {
			delete(s.state.intervals, key)
			summary.Intervals++
		}
	}
	for key, item := range s.state.assessments {
		if item.TenantSlug == tenantSlug && NormalizeHomeKey(item.HomeKey) == homeKey {
			delete(s.state.assessments, key)
			summary.TariffAssessments++
		}
	}
	for key, item := range s.state.measures {
		if item.TenantSlug != tenantSlug || NormalizeHomeKey(item.HomeKey) != homeKey {
			continue
		}
		if item.BeforeFrom == nil && item.BeforeTo == nil && item.AfterFrom == nil && item.AfterTo == nil &&
			item.BeforePeakKW == nil && item.AfterPeakKW == nil && item.BeforeQuality == "" && item.AfterQuality == "" {
			continue
		}
		item.BeforeFrom = nil
		item.BeforeTo = nil
		item.AfterFrom = nil
		item.AfterTo = nil
		item.BeforePeakKW = nil
		item.AfterPeakKW = nil
		item.BeforeQuality = ""
		item.AfterQuality = ""
		item.UpdatedAt = time.Now().UTC()
		s.state.measures[key] = item
		summary.Measures++
	}
	profileKey := tenantSlug + "\x00" + homeKey
	if profile, ok := s.state.profiles[profileKey]; ok {
		if profile.RecommendationID != "" || profile.RecommendationStatus != "" {
			profile.RecommendationID = ""
			profile.RecommendationStatus = ""
			profile.UpdatedAt = time.Now().UTC()
			s.state.profiles[profileKey] = profile
		}
	}
	return summary, nil
}

func (s *MemoryStore) DeleteProfile(tenantSlug string) (DeleteSummary, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	tenantSlug = normalizeSlug(tenantSlug)
	homeKey := s.scopeHomeKey()
	profileKey := tenantSlug + "\x00" + homeKey
	var summary DeleteSummary
	if current, ok := s.state.profiles[profileKey]; ok {
		freeStartedAt := current.FreeStartedAt
		freeUntilAt := current.FreeUntilAt
		delete(s.state.profiles, profileKey)
		// Keep an empty technical placeholder so a declarative pilot seed cannot
		// silently recreate user-deleted home data on the next restart.
		placeholder := DefaultProfileForHome(tenantSlug, homeKey, time.Now())
		// The commercial entitlement is contract metadata, not energy content.
		// Deleting and re-onboarding must not restart the agreed free period.
		placeholder.FreeStartedAt = freeStartedAt
		placeholder.FreeUntilAt = freeUntilAt
		s.state.profiles[profileKey] = placeholder
		summary.Profiles = 1
	}
	for key, item := range s.state.assets {
		if item.TenantSlug == tenantSlug && NormalizeHomeKey(item.HomeKey) == homeKey {
			delete(s.state.assets, key)
			summary.Assets++
		}
	}
	for key, item := range s.state.mappings {
		if item.TenantSlug == tenantSlug && NormalizeHomeKey(item.HomeKey) == homeKey {
			delete(s.state.mappings, key)
			summary.Mappings++
		}
	}
	for key, item := range s.state.intervals {
		if item.TenantSlug == tenantSlug && NormalizeHomeKey(item.HomeKey) == homeKey {
			delete(s.state.intervals, key)
			summary.Intervals++
		}
	}
	for key, item := range s.state.imports {
		if item.TenantSlug == tenantSlug && NormalizeHomeKey(item.HomeKey) == homeKey {
			delete(s.state.imports, key)
			summary.Imports++
		}
	}
	for key, item := range s.state.maintenance {
		if item.TenantSlug == tenantSlug && NormalizeHomeKey(item.HomeKey) == homeKey {
			delete(s.state.maintenance, key)
			summary.Maintenance++
		}
	}
	for key, item := range s.state.assessments {
		if item.TenantSlug == tenantSlug && NormalizeHomeKey(item.HomeKey) == homeKey {
			delete(s.state.assessments, key)
			summary.TariffAssessments++
		}
	}
	for key, item := range s.state.measures {
		if item.TenantSlug == tenantSlug && NormalizeHomeKey(item.HomeKey) == homeKey {
			delete(s.state.measures, key)
			summary.Measures++
		}
	}
	return summary, nil
}

func (s *MemoryStore) PurgeExpired(rawImportBefore, intervalBefore, assessmentBefore time.Time) (DeleteSummary, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	var summary DeleteSummary
	for key, item := range s.state.imports {
		if !rawImportBefore.IsZero() && item.ImportedAt.Before(rawImportBefore) {
			delete(s.state.imports, key)
			summary.Imports++
		}
	}
	for key, item := range s.state.intervals {
		if !intervalBefore.IsZero() && item.StartsAt.Before(intervalBefore) {
			delete(s.state.intervals, key)
			summary.Intervals++
		}
	}
	for key, item := range s.state.assessments {
		if !assessmentBefore.IsZero() && item.CreatedAt.Before(assessmentBefore) {
			delete(s.state.assessments, key)
			summary.TariffAssessments++
		}
	}
	return summary, nil
}

func (s *MemoryStore) ListMaintenance(tenantSlug string) ([]MaintenancePlan, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	tenantSlug = normalizeSlug(tenantSlug)
	homeKey := s.scopeHomeKey()
	out := []MaintenancePlan{}
	for _, item := range s.state.maintenance {
		if item.TenantSlug == tenantSlug && NormalizeHomeKey(item.HomeKey) == homeKey {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].NextDueAt.Equal(out[j].NextDueAt) {
			return out[i].Title < out[j].Title
		}
		return out[i].NextDueAt.Before(out[j].NextDueAt)
	})
	return out, nil
}

func (s *MemoryStore) UpsertMaintenance(plan MaintenancePlan) error {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	plan.HomeKey = s.scopeHomeKey()
	normalized, err := NormalizeMaintenancePlan(plan, time.Now())
	if err != nil {
		return err
	}
	for key, existing := range s.state.maintenance {
		if existing.TenantSlug == normalized.TenantSlug && NormalizeHomeKey(existing.HomeKey) == normalized.HomeKey && existing.AssetID == normalized.AssetID {
			normalized.ID = existing.ID
			normalized.CreatedAt = existing.CreatedAt
			delete(s.state.maintenance, key)
			break
		}
	}
	s.state.maintenance[normalized.TenantSlug+"\x00"+normalized.HomeKey+"\x00"+normalized.ID] = normalized
	return nil
}

func (s *MemoryStore) DeleteMaintenance(tenantSlug, id string) (bool, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	key := normalizeSlug(tenantSlug) + "\x00" + s.scopeHomeKey() + "\x00" + strings.TrimSpace(id)
	if _, ok := s.state.maintenance[key]; !ok {
		return false, nil
	}
	delete(s.state.maintenance, key)
	return true, nil
}

func (s *MemoryStore) SaveTariffAssessment(item TariffAssessment) error {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	item.HomeKey = s.scopeHomeKey()
	normalized, err := NormalizeTariffAssessment(item, time.Now())
	if err != nil {
		return err
	}
	s.state.assessments[normalized.TenantSlug+"\x00"+normalized.HomeKey+"\x00"+normalized.ID] = normalized
	return nil
}

func (s *MemoryStore) ListTariffAssessments(tenantSlug string) ([]TariffAssessment, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	tenantSlug = normalizeSlug(tenantSlug)
	homeKey := s.scopeHomeKey()
	out := []TariffAssessment{}
	for _, item := range s.state.assessments {
		if item.TenantSlug == tenantSlug && NormalizeHomeKey(item.HomeKey) == homeKey {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) UpsertMeasure(item Measure) error {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	item.HomeKey = s.scopeHomeKey()
	normalized, err := NormalizeMeasure(item, time.Now())
	if err != nil {
		return err
	}
	for key, existing := range s.state.measures {
		if existing.TenantSlug == normalized.TenantSlug && NormalizeHomeKey(existing.HomeKey) == normalized.HomeKey && existing.IssueID == normalized.IssueID {
			normalized.ID = existing.ID
			normalized.CreatedAt = existing.CreatedAt
			delete(s.state.measures, key)
			break
		}
	}
	s.state.measures[normalized.TenantSlug+"\x00"+normalized.HomeKey+"\x00"+normalized.ID] = normalized
	return nil
}

func (s *MemoryStore) GetMeasure(tenantSlug, id string) (Measure, bool, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	item, ok := s.state.measures[normalizeSlug(tenantSlug)+"\x00"+s.scopeHomeKey()+"\x00"+strings.TrimSpace(id)]
	return item, ok, nil
}

func (s *MemoryStore) ListMeasures(tenantSlug string) ([]Measure, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	tenantSlug = normalizeSlug(tenantSlug)
	homeKey := s.scopeHomeKey()
	out := []Measure{}
	for _, item := range s.state.measures {
		if item.TenantSlug == tenantSlug && NormalizeHomeKey(item.HomeKey) == homeKey {
			item.SharedFields = append([]string(nil), item.SharedFields...)
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out, nil
}

// tenantIdentityCache remembers the immutable id behind a slug for the life of
// the store it belongs to.
//
// Caching a HIT is safe by construction: an identity is minted once and never
// moves — if it did, every row pointing at the old one would be orphaned, which
// is the whole reason the id exists. Caching a MISS would not be: a house
// activated through the home portal mints its identity on its first write, and
// a remembered miss would keep that house invisible to every read afterwards.
type tenantIdentityCache struct {
	mu     sync.RWMutex
	byslug map[string]string
}

func newTenantIdentityCache() *tenantIdentityCache {
	return &tenantIdentityCache{byslug: map[string]string{}}
}

// id resolves a canonical slug to its identity, or "" when the slug has none
// yet. A nil cache still resolves — it just asks the database every time — so a
// zero-value SQLStore cannot panic here.
func (c *tenantIdentityCache) id(q tenantid.Querier, slug string) string {
	if c == nil {
		id, _ := tenantid.Lookup(q, slug)
		return id
	}
	c.mu.RLock()
	id, ok := c.byslug[slug]
	c.mu.RUnlock()
	if ok {
		return id
	}
	id, found := tenantid.Lookup(q, slug)
	if !found {
		return ""
	}
	c.mu.Lock()
	c.byslug[slug] = id
	c.mu.Unlock()
	return id
}

type SQLStore struct {
	db      *sql.DB
	homeKey string
	// tenant is the reference the caller was authorized with. When it is set,
	// every predicate below filters on tenant.ID without re-deriving the
	// identity from a slug that arrived later in the call.
	tenant     store.TenantRef
	identities *tenantIdentityCache
}

func NewSQLStore(db *sql.DB) *SQLStore {
	if db == nil {
		return nil
	}
	return &SQLStore{db: db, identities: newTenantIdentityCache()}
}

func (s *SQLStore) ForHome(homeKey string) Storage {
	return &SQLStore{db: s.db, homeKey: NormalizeHomeKey(homeKey), tenant: s.tenant, identities: s.identities}
}

// ForTenant binds the identity the caller proved it may act for. It is the same
// shape internal/store uses: the reference is taken once at the authorization
// boundary and travels with the handle, so a later slug argument cannot quietly
// re-point a query at another house.
func (s *SQLStore) ForTenant(tenant store.TenantRef) Storage {
	if s == nil {
		return nil
	}
	return &SQLStore{db: s.db, homeKey: s.homeKey, tenant: tenant, identities: s.identities}
}

func (s *SQLStore) scopeHomeKey() string { return NormalizeHomeKey(s.homeKey) }

// ErrTenantScopeMismatch reports a store bound to one house being asked about
// another. It is an error rather than a fallback because there is no legitimate
// caller for it: the reference comes from the authorization check, and a slug
// that disagrees with it means the check and the query are about different
// houses.
var ErrTenantScopeMismatch = errors.New("energy: tenant scope mismatch")

// scope resolves the immutable identity every tenant-scoped statement filters
// on.
//
// A bound reference is the authority. Without one — the boot seeds, the
// retention worker, the device connector endpoints — the slug is resolved once
// through tenantid, which is the same resolver every writer already uses, so
// there is still exactly one definition of who a slug is.
//
// A slug with no identity yet resolves to "", which matches no row. That is the
// intended answer and not an error: a caller that cannot be identified must
// read nothing rather than everything.
func (s *SQLStore) scope(tenantSlug string) (string, error) {
	slug := normalizeSlug(tenantSlug)
	if bound := normalizeSlug(s.tenant.Slug); s.tenant.ID != "" && bound != "" {
		if bound != slug {
			return "", fmt.Errorf("%w: bound to %q, asked for %q", ErrTenantScopeMismatch, bound, slug)
		}
		return s.tenant.ID, nil
	}
	if slug == "" {
		return "", nil
	}
	return s.identities.id(s.db, slug), nil
}

func (s *SQLStore) ListProfiles(tenantSlug string) ([]HomeProfile, error) {
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT tenant_slug,home_key,unit_id,home_type,household_name,operating_mode,automation_stage,onboarding_step,onboarding_complete,target_peak_kw,agreed_power_kw,recommendation_id,recommendation_status,free_started_at,free_until_at,created_at,updated_at
		FROM home_profiles WHERE tenant_id=$1 ORDER BY home_key`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HomeProfile{}
	for rows.Next() {
		item, err := scanHomeProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

type homeProfileScanner interface{ Scan(dest ...any) error }

func scanHomeProfile(scanner homeProfileScanner) (HomeProfile, error) {
	var item HomeProfile
	// onboarding_complete is `boolean` on PostgreSQL and INTEGER CHECK (x IN
	// (0,1)) on SQLite. A Go bool is the one destination both engines convert
	// to; an int scans the SQLite column and fails outright on PostgreSQL.
	var complete bool
	var target, agreed sql.NullFloat64
	var freeStarted, freeUntil, created, updated sql.NullString
	err := scanner.Scan(&item.TenantSlug, &item.HomeKey, &item.UnitID, &item.HomeType, &item.HouseholdName, &item.OperatingMode, &item.AutomationStage, &item.OnboardingStep, &complete, &target, &agreed, &item.RecommendationID, &item.RecommendationStatus, &freeStarted, &freeUntil, &created, &updated)
	if err != nil {
		return HomeProfile{}, err
	}
	item.OnboardingComplete = complete
	if target.Valid {
		item.TargetPeakKW = &target.Float64
	}
	if agreed.Valid {
		item.AgreedPowerKW = &agreed.Float64
	}
	if parsed, ok := parseTime(freeStarted.String); ok {
		item.FreeStartedAt = &parsed
	}
	if parsed, ok := parseTime(freeUntil.String); ok {
		item.FreeUntilAt = &parsed
	}
	item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created.String)
	item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated.String)
	return item, nil
}

func (s *SQLStore) Profile(tenantSlug string) (HomeProfile, bool, error) {
	if s == nil || s.db == nil {
		return HomeProfile{}, false, nil
	}
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return HomeProfile{}, false, err
	}
	item, err := scanHomeProfile(s.db.QueryRow(`SELECT tenant_slug,home_key,unit_id,home_type,household_name,operating_mode,automation_stage,onboarding_step,onboarding_complete,target_peak_kw,agreed_power_kw,recommendation_id,recommendation_status,free_started_at,free_until_at,created_at,updated_at
		FROM home_profiles WHERE tenant_id=$1 AND home_key=$2`, tenantID, s.scopeHomeKey()))
	if err == sql.ErrNoRows {
		return HomeProfile{}, false, nil
	}
	if err != nil {
		return HomeProfile{}, false, err
	}
	return item, true, nil
}

func (s *SQLStore) SaveProfile(profile HomeProfile) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("energy: database unavailable")
	}
	profile.HomeKey = s.scopeHomeKey()
	profile = NormalizeProfile(profile, time.Now())
	if profile.TenantSlug == "" {
		return fmt.Errorf("energy: tenant required")
	}
	if _, err := s.scope(profile.TenantSlug); err != nil {
		return err
	}
	var target any
	if profile.TargetPeakKW != nil {
		target = *profile.TargetPeakKW
	}
	var agreed any
	if profile.AgreedPowerKW != nil {
		agreed = *profile.AgreedPowerKW
	}
	var free any
	if profile.FreeStartedAt != nil {
		free = profile.FreeStartedAt.UTC().Format(time.RFC3339Nano)
	}
	var freeUntil any
	if profile.FreeUntilAt != nil {
		freeUntil = profile.FreeUntilAt.UTC().Format(time.RFC3339Nano)
	}
	tenantID, err := tenantid.Ensure(s.db, profile.TenantSlug)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO home_profiles
		(tenant_id,tenant_slug,home_key,unit_id,home_type,household_name,operating_mode,automation_stage,onboarding_step,onboarding_complete,target_peak_kw,agreed_power_kw,recommendation_id,recommendation_status,free_started_at,free_until_at,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		ON CONFLICT(tenant_slug,home_key) DO UPDATE SET
		unit_id=excluded.unit_id, home_type=excluded.home_type, household_name=excluded.household_name,
		operating_mode=excluded.operating_mode, automation_stage=excluded.automation_stage, onboarding_step=excluded.onboarding_step,
		onboarding_complete=excluded.onboarding_complete, target_peak_kw=excluded.target_peak_kw,
		agreed_power_kw=excluded.agreed_power_kw,
		recommendation_id=excluded.recommendation_id, recommendation_status=excluded.recommendation_status,
		free_started_at=COALESCE(home_profiles.free_started_at,excluded.free_started_at),
		free_until_at=COALESCE(home_profiles.free_until_at,excluded.free_until_at),
		tenant_id=coalesce(home_profiles.tenant_id,excluded.tenant_id),
		updated_at=excluded.updated_at`,
		tenantID, profile.TenantSlug, profile.HomeKey, profile.UnitID, profile.HomeType, profile.HouseholdName, profile.OperatingMode, profile.AutomationStage,
		profile.OnboardingStep, profile.OnboardingComplete, target, agreed, profile.RecommendationID, profile.RecommendationStatus, free, freeUntil,
		profile.CreatedAt.Format(time.RFC3339Nano), profile.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

// ensureProfile guarantees this home has a profile row the tenant_id filters can
// see, because every child table's insert depends on one existing.
func (s *SQLStore) ensureProfile(tenantSlug string) error {
	if _, ok, err := s.Profile(tenantSlug); err != nil {
		return err
	} else if ok {
		return nil
	}
	// A row written by the PREVIOUS release carries a NULL tenant_id and is
	// therefore invisible to the filter above. Adopting it has to happen before
	// the default profile below, or that upsert would overwrite a real
	// household's name, mode and onboarding state with defaults — the boot-time
	// backfill normally closes this window, and this is what keeps a store that
	// never saw one from destroying data.
	if adopted, err := s.adoptUnownedProfile(tenantSlug); err != nil {
		return err
	} else if adopted {
		return nil
	}
	return s.SaveProfile(DefaultProfile(tenantSlug, time.Now()))
}

// adoptUnownedProfile links this home's identity-less profile row, if it has
// one, and reports whether it found it.
func (s *SQLStore) adoptUnownedProfile(tenantSlug string) (bool, error) {
	slug := normalizeSlug(tenantSlug)
	if slug == "" {
		return false, nil
	}
	if _, err := s.scope(slug); err != nil {
		return false, err
	}
	tenantID, err := tenantid.Ensure(s.db, slug)
	if err != nil {
		return false, err
	}
	result, err := s.db.Exec(
		`UPDATE home_profiles SET tenant_id=$1 WHERE tenant_id IS NULL AND tenant_slug=$2 AND home_key=$3`,
		tenantID, slug, s.scopeHomeKey())
	if err != nil {
		return false, err
	}
	adopted, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return adopted > 0, nil
}

func (s *SQLStore) ListAssets(tenantSlug string) ([]Asset, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id,tenant_slug,home_key,kind,name,rated_power_kw,flexibility,source,confirmed,metadata_json,created_at,updated_at
		FROM energy_assets WHERE tenant_id=$1 AND home_key=$2 ORDER BY kind,name,id`, tenantID, s.scopeHomeKey())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Asset{}
	for rows.Next() {
		var item Asset
		var rated sql.NullFloat64
		var confirmed bool
		var metadata, created, updated string
		if err := rows.Scan(&item.ID, &item.TenantSlug, &item.HomeKey, &item.Kind, &item.Name, &rated, &item.Flexibility, &item.Source, &confirmed, &metadata, &created, &updated); err != nil {
			return nil, err
		}
		if rated.Valid {
			item.RatedPowerKW = &rated.Float64
		}
		item.Confirmed = confirmed
		_ = json.Unmarshal([]byte(metadata), &item.Metadata)
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *SQLStore) UpsertAsset(asset Asset) error {
	if err := s.ensureProfile(asset.TenantSlug); err != nil {
		return err
	}
	asset.HomeKey = s.scopeHomeKey()
	asset = NormalizeAsset(asset, time.Now())
	var rated any
	if asset.RatedPowerKW != nil {
		rated = *asset.RatedPowerKW
	}
	metadata, _ := json.Marshal(asset.Metadata)
	tenantID, err := tenantid.Ensure(s.db, asset.TenantSlug)
	if err != nil {
		return err
	}
	// ON CONFLICT(id) — not (tenant_slug,id) — is what makes the guard below
	// mean anything: an asset id is globally unique on BOTH engines (SQLite
	// declares `id TEXT PRIMARY KEY`, PostgreSQL migration 0005 adds the
	// matching unique index), so the same id offered for a second house
	// COLLIDES here instead of inserting a second row. Conflicting on
	// (tenant_slug,id) would delete that guarantee silently: the insert would
	// succeed, RowsAffected would be 1, and the check below would never fire.
	result, err := s.db.Exec(`INSERT INTO energy_assets
		(id,tenant_id,tenant_slug,home_key,kind,name,rated_power_kw,flexibility,source,confirmed,metadata_json,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT(id) DO UPDATE SET kind=excluded.kind,name=excluded.name,rated_power_kw=excluded.rated_power_kw,
		flexibility=excluded.flexibility,source=excluded.source,confirmed=excluded.confirmed,
		metadata_json=excluded.metadata_json,updated_at=excluded.updated_at,
		tenant_id=coalesce(energy_assets.tenant_id,excluded.tenant_id)
		WHERE energy_assets.tenant_slug=excluded.tenant_slug AND energy_assets.home_key=excluded.home_key`,
		asset.ID, tenantID, asset.TenantSlug, asset.HomeKey, asset.Kind, asset.Name, rated, asset.Flexibility,
		asset.Source, asset.Confirmed, string(metadata),
		asset.CreatedAt.Format(time.RFC3339Nano), asset.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrAssetIDTaken
	}
	return nil
}

// ErrAssetIDTaken reports an asset id that already belongs to another house or
// another home within the house. It is a named error because the only test that
// covers the cross-tenant guarantee used to accept ANY error — including a
// dialect failure — as proof that the guard had fired.
var ErrAssetIDTaken = errors.New("energy: asset id belongs to another tenant")

func (s *SQLStore) UpdateAssetPriorities(tenantSlug string, order []string) (bool, error) {
	if s == nil || s.db == nil {
		return false, fmt.Errorf("energy: sql store unavailable")
	}
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return false, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for position, id := range order {
		var raw string
		row := tx.QueryRow(`SELECT metadata_json FROM energy_assets WHERE tenant_id=$1 AND home_key=$2 AND id=$3`, tenantID, s.scopeHomeKey(), id)
		if err := row.Scan(&raw); err != nil {
			if err == sql.ErrNoRows {
				return false, nil
			}
			return false, err
		}
		metadata := map[string]string{}
		_ = json.Unmarshal([]byte(raw), &metadata)
		metadata["priority"] = strconv.Itoa(position + 1)
		encoded, _ := json.Marshal(metadata)
		if _, err := tx.Exec(`UPDATE energy_assets SET metadata_json=$1, updated_at=$2 WHERE tenant_id=$3 AND home_key=$4 AND id=$5`,
			string(encoded), now, tenantID, s.scopeHomeKey(), id); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *SQLStore) DeleteAsset(tenantSlug, id string) (bool, error) {
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return false, err
	}
	id = strings.TrimSpace(id)
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE energy_entity_mappings SET asset_id='',updated_at=$1 WHERE tenant_id=$2 AND home_key=$3 AND asset_id=$4`,
		time.Now().UTC().Format(time.RFC3339Nano), tenantID, s.scopeHomeKey(), id); err != nil {
		return false, err
	}
	result, err := tx.Exec(`DELETE FROM energy_assets WHERE tenant_id=$1 AND home_key=$2 AND id=$3`, tenantID, s.scopeHomeKey(), id)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *SQLStore) ListMappings(tenantSlug string) ([]EntityMapping, error) {
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id,tenant_slug,home_key,entity_id,asset_id,metric,display_name,unit,device_class,confirmed,last_seen_at,created_at,updated_at
		FROM energy_entity_mappings WHERE tenant_id=$1 AND home_key=$2 ORDER BY confirmed DESC,metric,display_name,entity_id`, tenantID, s.scopeHomeKey())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EntityMapping{}
	for rows.Next() {
		var item EntityMapping
		var confirmed bool
		var lastSeen sql.NullString
		var created, updated string
		if err := rows.Scan(&item.ID, &item.TenantSlug, &item.HomeKey, &item.EntityID, &item.AssetID, &item.Metric, &item.DisplayName, &item.Unit, &item.DeviceClass, &confirmed, &lastSeen, &created, &updated); err != nil {
			return nil, err
		}
		item.Confirmed = confirmed
		if parsed, ok := parseTime(lastSeen.String); ok {
			item.LastSeenAt = &parsed
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *SQLStore) UpsertMapping(mapping EntityMapping) error {
	if err := s.ensureProfile(mapping.TenantSlug); err != nil {
		return err
	}
	mapping.HomeKey = s.scopeHomeKey()
	mapping = NormalizeMapping(mapping, time.Now())
	tenantID, err := tenantid.Ensure(s.db, mapping.TenantSlug)
	if err != nil {
		return err
	}
	if mapping.AssetID != "" {
		var exists int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM energy_assets WHERE tenant_id=$1 AND home_key=$2 AND id=$3`, tenantID, mapping.HomeKey, mapping.AssetID).Scan(&exists); err != nil {
			return err
		}
		if exists != 1 {
			return ErrMappingAssetForeign
		}
	}
	var seen any
	if mapping.LastSeenAt != nil {
		seen = mapping.LastSeenAt.UTC().Format(time.RFC3339Nano)
	}
	_, err = s.db.Exec(`INSERT INTO energy_entity_mappings
		(id,tenant_id,tenant_slug,home_key,entity_id,asset_id,metric,display_name,unit,device_class,confirmed,last_seen_at,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT(tenant_slug,home_key,entity_id) DO UPDATE SET
		asset_id=excluded.asset_id,metric=excluded.metric,display_name=excluded.display_name,unit=excluded.unit,
		device_class=excluded.device_class,confirmed=excluded.confirmed,last_seen_at=excluded.last_seen_at,
		tenant_id=coalesce(energy_entity_mappings.tenant_id,excluded.tenant_id),
		updated_at=excluded.updated_at`,
		mapping.ID, tenantID, mapping.TenantSlug, mapping.HomeKey, mapping.EntityID, mapping.AssetID, mapping.Metric, mapping.DisplayName,
		mapping.Unit, mapping.DeviceClass, mapping.Confirmed, seen,
		mapping.CreatedAt.Format(time.RFC3339Nano), mapping.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

// ErrMappingAssetForeign reports a mapping pointed at an asset this house and
// home do not own. Named for the same reason as ErrAssetIDTaken: the test that
// covers it accepted any error at all, so a dialect failure looked identical to
// the guard doing its job.
var ErrMappingAssetForeign = errors.New("energy: mapping asset must belong to tenant")

func (s *SQLStore) DeleteMapping(tenantSlug, id string) (bool, error) {
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return false, err
	}
	result, err := s.db.Exec(`DELETE FROM energy_entity_mappings WHERE tenant_id=$1 AND home_key=$2 AND id=$3`, tenantID, s.scopeHomeKey(), strings.TrimSpace(id))
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func (s *SQLStore) PutInterval(interval Interval) error {
	if err := s.ensureProfile(interval.TenantSlug); err != nil {
		return err
	}
	if interval.Duration <= 0 {
		interval.Duration = 15 * time.Minute
	}
	interval.TenantSlug = normalizeSlug(interval.TenantSlug)
	interval.StartsAt = interval.StartsAt.UTC()
	if interval.Quality == "" {
		interval.Quality = "measured"
	}
	if interval.Source == "" {
		interval.Source = "home-assistant"
	}
	if interval.CreatedAt.IsZero() {
		interval.CreatedAt = time.Now().UTC()
	}
	interval.HomeKey = s.scopeHomeKey()
	tenantID, err := tenantid.Ensure(s.db, interval.TenantSlug)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO energy_intervals
		(tenant_id,tenant_slug,home_key,starts_at,duration_minutes,import_kwh,average_kw,quality,source,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT(tenant_slug,home_key,starts_at,source) DO UPDATE SET
		duration_minutes=excluded.duration_minutes,import_kwh=excluded.import_kwh,
		tenant_id=coalesce(energy_intervals.tenant_id,excluded.tenant_id),
		average_kw=excluded.average_kw,quality=excluded.quality,created_at=excluded.created_at`,
		tenantID, interval.TenantSlug, interval.HomeKey, interval.StartsAt.Format(time.RFC3339Nano), int(interval.Duration/time.Minute),
		interval.ImportKWh, interval.AverageKW, interval.Quality, interval.Source,
		interval.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *SQLStore) ListIntervals(tenantSlug string, from, to time.Time) ([]Interval, error) {
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return nil, err
	}
	// $4 and $5 carry the SAME value and must stay two placeholders. Both engines
	// bind $n POSITIONALLY — verified against modernc.org/sqlite, which does NOT
	// treat $4 as a named parameter — so reusing $4 in both slots would leave $5
	// unbound and shift nothing, failing at execution rather than tidying anything.
	// An earlier version of this comment asserted the named-parameter behaviour and
	// misled two readers into reasoning from it; the conclusion was right for the
	// wrong reason.
	rows, err := s.db.Query(`SELECT tenant_slug,home_key,starts_at,duration_minutes,import_kwh,average_kw,quality,source,created_at
		FROM energy_intervals WHERE tenant_id=$1 AND home_key=$2 AND starts_at>=$3 AND ($4='' OR starts_at<$5)
		ORDER BY starts_at`, tenantID, s.scopeHomeKey(), from.UTC().Format(time.RFC3339Nano), nullableTime(to), nullableTime(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Interval{}
	for rows.Next() {
		var item Interval
		var starts, created string
		var minutes int
		if err := rows.Scan(&item.TenantSlug, &item.HomeKey, &starts, &minutes, &item.ImportKWh, &item.AverageKW, &item.Quality, &item.Source, &created); err != nil {
			return nil, err
		}
		item.StartsAt, _ = time.Parse(time.RFC3339Nano, starts)
		item.Duration = time.Duration(minutes) * time.Minute
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *SQLStore) PutImport(record ImportRecord, intervals []Interval) (bool, error) {
	if err := s.ensureProfile(record.TenantSlug); err != nil {
		return false, err
	}
	record.TenantSlug = normalizeSlug(record.TenantSlug)
	record.HomeKey = s.scopeHomeKey()
	if record.ID == "" {
		record.ID = NewID("import")
	}
	if record.ImportedAt.IsZero() {
		record.ImportedAt = time.Now().UTC()
	}
	for _, interval := range intervals {
		if normalizeSlug(interval.TenantSlug) != record.TenantSlug {
			return false, fmt.Errorf("energy: interval tenant mismatch")
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	tenantID, err := tenantid.Ensure(tx, record.TenantSlug)
	if err != nil {
		return false, err
	}
	// energy_imports is the one upsert here that cannot heal in its conflict
	// clause: DO NOTHING is what makes RowsAffected==0 mean "this file was
	// already imported", and turning it into DO UPDATE would report every
	// re-import as new. So the repair is its own statement, before the insert.
	// Without it a row written by the previous release keeps a NULL identity
	// forever — invisible to the filters below and to the RLS policy — and
	// re-importing the same file would never fix it.
	if _, err := tx.Exec(`UPDATE energy_imports SET tenant_id=$1
		WHERE tenant_id IS NULL AND tenant_slug=$2 AND home_key=$3 AND sha256=$4`,
		tenantID, record.TenantSlug, record.HomeKey, record.SHA256); err != nil {
		return false, err
	}
	result, err := tx.Exec(`INSERT INTO energy_imports
		(id,tenant_id,tenant_slug,home_key,filename,sha256,format,payload,imported_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT(tenant_slug,home_key,sha256) DO NOTHING`,
		record.ID, tenantID, record.TenantSlug, record.HomeKey, record.Filename, record.SHA256, record.Format,
		record.Payload, record.ImportedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return false, err
	}
	inserted, _ := result.RowsAffected()
	if inserted == 0 {
		// Already imported. This path still COMMITS, because the identity
		// repair above happened inside this transaction and returning here
		// would hand it to the deferred rollback — leaving the row unowned
		// exactly on the one path that reaches it.
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}
	for _, interval := range intervals {
		if interval.Duration <= 0 {
			interval.Duration = 15 * time.Minute
		}
		if interval.Source == "" {
			interval.Source = "smart-meter"
		}
		if interval.Quality == "" {
			interval.Quality = QualityMeasured
		}
		if interval.CreatedAt.IsZero() {
			interval.CreatedAt = record.ImportedAt
		}
		if _, err := tx.Exec(`INSERT INTO energy_intervals
			(tenant_id,tenant_slug,home_key,starts_at,duration_minutes,import_kwh,average_kw,quality,source,created_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			ON CONFLICT(tenant_slug,home_key,starts_at,source) DO UPDATE SET
			duration_minutes=excluded.duration_minutes,import_kwh=excluded.import_kwh,
			tenant_id=coalesce(energy_intervals.tenant_id,excluded.tenant_id),
			average_kw=excluded.average_kw,quality=excluded.quality,created_at=excluded.created_at`,
			tenantID, record.TenantSlug, record.HomeKey, interval.StartsAt.UTC().Format(time.RFC3339Nano), int(interval.Duration/time.Minute),
			interval.ImportKWh, interval.AverageKW, interval.Quality, interval.Source,
			interval.CreatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *SQLStore) ListImports(tenantSlug string) ([]ImportRecord, error) {
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id,tenant_slug,home_key,filename,sha256,format,imported_at
		FROM energy_imports WHERE tenant_id=$1 AND home_key=$2 ORDER BY imported_at DESC,id`, tenantID, s.scopeHomeKey())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ImportRecord{}
	for rows.Next() {
		var record ImportRecord
		var imported string
		if err := rows.Scan(&record.ID, &record.TenantSlug, &record.HomeKey, &record.Filename, &record.SHA256, &record.Format, &imported); err != nil {
			return nil, err
		}
		record.ImportedAt, _ = time.Parse(time.RFC3339Nano, imported)
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *SQLStore) ListImportsForExport(tenantSlug string) ([]ImportRecord, error) {
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id,tenant_slug,home_key,filename,sha256,format,payload,imported_at
		FROM energy_imports WHERE tenant_id=$1 AND home_key=$2 ORDER BY imported_at DESC,id`, tenantID, s.scopeHomeKey())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ImportRecord{}
	for rows.Next() {
		var record ImportRecord
		var imported string
		if err := rows.Scan(&record.ID, &record.TenantSlug, &record.HomeKey, &record.Filename, &record.SHA256, &record.Format, &record.Payload, &imported); err != nil {
			return nil, err
		}
		record.ImportedAt, _ = time.Parse(time.RFC3339Nano, imported)
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *SQLStore) DeleteMeasurementData(tenantSlug string) (DeleteSummary, error) {
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return DeleteSummary{}, err
	}
	homeKey := s.scopeHomeKey()
	tx, err := s.db.Begin()
	if err != nil {
		return DeleteSummary{}, err
	}
	defer tx.Rollback()
	var summary DeleteSummary
	// Placeholder numbering RESETS per statement and the UPDATE's timestamp is
	// bound FIRST, which is why the branch below exists: $1 means the retention
	// timestamp there and the tenant everywhere else.
	for _, item := range []struct {
		query  string
		target *int
	}{
		{`DELETE FROM energy_imports WHERE tenant_id=$1 AND home_key=$2`, &summary.Imports},
		{`DELETE FROM energy_intervals WHERE tenant_id=$1 AND home_key=$2`, &summary.Intervals},
		{`DELETE FROM energy_tariff_assessments WHERE tenant_id=$1 AND home_key=$2`, &summary.TariffAssessments},
		{`UPDATE energy_measures SET before_from=NULL,before_to=NULL,after_from=NULL,after_to=NULL,
			before_peak_kw=NULL,after_peak_kw=NULL,before_quality='',after_quality='',updated_at=$1
			WHERE tenant_id=$2 AND home_key=$3 AND (before_from IS NOT NULL OR before_to IS NOT NULL OR after_from IS NOT NULL OR after_to IS NOT NULL
				OR before_peak_kw IS NOT NULL OR after_peak_kw IS NOT NULL OR before_quality<>'' OR after_quality<>'')`, &summary.Measures},
	} {
		var result sql.Result
		if strings.HasPrefix(strings.TrimSpace(item.query), "UPDATE") {
			result, err = tx.Exec(item.query, time.Now().UTC().Format(time.RFC3339Nano), tenantID, homeKey)
		} else {
			result, err = tx.Exec(item.query, tenantID, homeKey)
		}
		if err != nil {
			return DeleteSummary{}, err
		}
		n, _ := result.RowsAffected()
		*item.target = int(n)
	}
	if _, err := tx.Exec(`UPDATE home_profiles
		SET recommendation_id='',recommendation_status='',updated_at=$1
		WHERE tenant_id=$2 AND home_key=$3 AND (recommendation_id<>'' OR recommendation_status<>'')`,
		time.Now().UTC().Format(time.RFC3339Nano), tenantID, homeKey); err != nil {
		return DeleteSummary{}, err
	}
	if err := tx.Commit(); err != nil {
		return DeleteSummary{}, err
	}
	return summary, nil
}

func (s *SQLStore) DeleteProfile(tenantSlug string) (DeleteSummary, error) {
	slug := normalizeSlug(tenantSlug)
	tenantID, err := s.scope(slug)
	if err != nil {
		return DeleteSummary{}, err
	}
	homeKey := s.scopeHomeKey()
	tx, err := s.db.Begin()
	if err != nil {
		return DeleteSummary{}, err
	}
	defer tx.Rollback()
	var summary DeleteSummary
	var retainedFreeStartedAt, retainedFreeUntilAt sql.NullString
	if err := tx.QueryRow(`SELECT free_started_at,free_until_at FROM home_profiles WHERE tenant_id=$1 AND home_key=$2`, tenantID, homeKey).Scan(&retainedFreeStartedAt, &retainedFreeUntilAt); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return DeleteSummary{}, err
	}
	for _, item := range []struct {
		table  string
		target *int
	}{
		{"energy_assets", &summary.Assets},
		{"energy_entity_mappings", &summary.Mappings},
		{"energy_intervals", &summary.Intervals},
		{"energy_imports", &summary.Imports},
		{"energy_maintenance_plans", &summary.Maintenance},
		{"energy_tariff_assessments", &summary.TariffAssessments},
		{"energy_measures", &summary.Measures},
	} {
		// The table name is interpolated from the fixed slice above, never from
		// input; both placeholders sit in the trailing literal, so the
		// numbering is per-statement exactly as everywhere else.
		if err := tx.QueryRow(`SELECT COUNT(*) FROM `+item.table+` WHERE tenant_id=$1 AND home_key=$2`, tenantID, homeKey).Scan(item.target); err != nil {
			return DeleteSummary{}, err
		}
	}
	result, err := tx.Exec(`DELETE FROM home_profiles WHERE tenant_id=$1 AND home_key=$2`, tenantID, homeKey)
	if err != nil {
		return DeleteSummary{}, err
	}
	n, _ := result.RowsAffected()
	summary.Profiles = int(n)
	if summary.Profiles > 0 {
		// A deliberately empty row is the durable deletion marker. Profile seeds
		// only create missing rows, so they cannot resurrect a deleted household
		// name, inventory or history after restart.
		now := time.Now().UTC()
		var freeStartedAt any
		var freeUntilAt any
		if retainedFreeStartedAt.Valid {
			freeStartedAt = retainedFreeStartedAt.String
		}
		if retainedFreeUntilAt.Valid {
			freeUntilAt = retainedFreeUntilAt.String
		}
		tombstoneTenantID, idErr := tenantid.Ensure(tx, slug)
		if idErr != nil {
			return DeleteSummary{}, idErr
		}
		// onboarding_complete is `false`, not 0: the column is boolean on
		// PostgreSQL, and a bare integer literal in this positional list is the
		// easiest of the five boolean bindings to miss because it has no
		// conversion helper to grep for.
		if _, err := tx.Exec(`INSERT INTO home_profiles
			(tenant_id,tenant_slug,home_key,unit_id,home_type,household_name,operating_mode,automation_stage,onboarding_step,onboarding_complete,
			 target_peak_kw,agreed_power_kw,recommendation_id,recommendation_status,free_started_at,free_until_at,created_at,updated_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
			tombstoneTenantID, slug, homeKey, "", HomeApartment, "", ModeObserve, StageObserve, 1, false,
			nil, nil, "", "", freeStartedAt, freeUntilAt, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
			return DeleteSummary{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return DeleteSummary{}, err
	}
	return summary, nil
}

// PurgeExpired enforces retention across EVERY house, which is why it is the
// only method here with no tenant predicate at all — and why it must keep
// running on the process-wide handle. On a tenant-pinned PostgreSQL connection
// the row-level-security policy would narrow these deletes to one house and
// report the reduced count as success, turning a global retention sweep into a
// partial one with nothing to notice.
//
// It is on the fatal boot path (newApp returns its error), so a dialect defect
// here is not a degraded feature: the binary does not start.
func (s *SQLStore) PurgeExpired(rawImportBefore, intervalBefore, assessmentBefore time.Time) (DeleteSummary, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return DeleteSummary{}, err
	}
	defer tx.Rollback()
	var summary DeleteSummary
	// Three separate statements, each with ONE placeholder: numbering restarts
	// at $1 for every one of them.
	for _, item := range []struct {
		query  string
		before time.Time
		target *int
	}{
		{`DELETE FROM energy_imports WHERE imported_at<$1`, rawImportBefore, &summary.Imports},
		{`DELETE FROM energy_intervals WHERE starts_at<$1`, intervalBefore, &summary.Intervals},
		{`DELETE FROM energy_tariff_assessments WHERE created_at<$1`, assessmentBefore, &summary.TariffAssessments},
	} {
		if item.before.IsZero() {
			continue
		}
		result, err := tx.Exec(item.query, item.before.UTC().Format(time.RFC3339Nano))
		if err != nil {
			return DeleteSummary{}, err
		}
		n, _ := result.RowsAffected()
		*item.target = int(n)
	}
	if err := tx.Commit(); err != nil {
		return DeleteSummary{}, err
	}
	return summary, nil
}

func (s *SQLStore) ListMaintenance(tenantSlug string) ([]MaintenancePlan, error) {
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id,tenant_slug,home_key,asset_id,title,interval_months,last_completed_at,next_due_at,contact_id,document_id,issue_id,evidence_note,active,created_at,updated_at
		FROM energy_maintenance_plans WHERE tenant_id=$1 AND home_key=$2 ORDER BY next_due_at,title,id`, tenantID, s.scopeHomeKey())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MaintenancePlan{}
	for rows.Next() {
		var item MaintenancePlan
		var completed sql.NullString
		var nextDue, created, updated string
		var active bool
		if err := rows.Scan(&item.ID, &item.TenantSlug, &item.HomeKey, &item.AssetID, &item.Title, &item.IntervalMonths, &completed, &nextDue,
			&item.ContactID, &item.DocumentID, &item.IssueID, &item.EvidenceNote, &active, &created, &updated); err != nil {
			return nil, err
		}
		if parsed, ok := parseTime(completed.String); ok {
			item.LastCompletedAt = &parsed
		}
		item.NextDueAt, _ = time.Parse(time.RFC3339Nano, nextDue)
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		item.Active = active
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *SQLStore) UpsertMaintenance(plan MaintenancePlan) error {
	if err := s.ensureProfile(plan.TenantSlug); err != nil {
		return err
	}
	plan.HomeKey = s.scopeHomeKey()
	normalized, err := NormalizeMaintenancePlan(plan, time.Now())
	if err != nil {
		return err
	}
	var completed any
	if normalized.LastCompletedAt != nil {
		completed = normalized.LastCompletedAt.Format(time.RFC3339Nano)
	}
	tenantID, err := tenantid.Ensure(s.db, normalized.TenantSlug)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO energy_maintenance_plans
		(id,tenant_id,tenant_slug,home_key,asset_id,title,interval_months,last_completed_at,next_due_at,contact_id,document_id,issue_id,evidence_note,active,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT(tenant_slug,home_key,asset_id) DO UPDATE SET
		title=excluded.title,interval_months=excluded.interval_months,last_completed_at=excluded.last_completed_at,
		next_due_at=excluded.next_due_at,contact_id=excluded.contact_id,document_id=excluded.document_id,
		issue_id=excluded.issue_id,evidence_note=excluded.evidence_note,active=excluded.active,
		tenant_id=coalesce(energy_maintenance_plans.tenant_id,excluded.tenant_id),updated_at=excluded.updated_at`,
		normalized.ID, tenantID, normalized.TenantSlug, normalized.HomeKey, normalized.AssetID, normalized.Title, normalized.IntervalMonths, completed,
		normalized.NextDueAt.Format(time.RFC3339Nano), normalized.ContactID, normalized.DocumentID, normalized.IssueID,
		normalized.EvidenceNote, normalized.Active, normalized.CreatedAt.Format(time.RFC3339Nano), normalized.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

func (s *SQLStore) DeleteMaintenance(tenantSlug, id string) (bool, error) {
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return false, err
	}
	result, err := s.db.Exec(`DELETE FROM energy_maintenance_plans WHERE tenant_id=$1 AND home_key=$2 AND id=$3`, tenantID, s.scopeHomeKey(), strings.TrimSpace(id))
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func (s *SQLStore) SaveTariffAssessment(item TariffAssessment) error {
	if err := s.ensureProfile(item.TenantSlug); err != nil {
		return err
	}
	item.HomeKey = s.scopeHomeKey()
	normalized, err := NormalizeTariffAssessment(item, time.Now())
	if err != nil {
		return err
	}
	tenantID, err := tenantid.Ensure(s.db, normalized.TenantSlug)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO energy_tariff_assessments
		(id,tenant_id,tenant_slug,home_key,assessment_month,profile_id,profile_version,profile_status,source_url,peak_kw,billed_kw,annual_power_eur,data_quality,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		normalized.ID, tenantID, normalized.TenantSlug, normalized.HomeKey, normalized.AssessmentMonth, normalized.ProfileID, normalized.ProfileVersion,
		normalized.ProfileStatus, normalized.SourceURL, normalized.PeakKW, normalized.BilledKW, normalized.AnnualPowerEUR,
		normalized.DataQuality, normalized.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func (s *SQLStore) ListTariffAssessments(tenantSlug string) ([]TariffAssessment, error) {
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id,tenant_slug,home_key,assessment_month,profile_id,profile_version,profile_status,source_url,peak_kw,billed_kw,annual_power_eur,data_quality,created_at
		FROM energy_tariff_assessments WHERE tenant_id=$1 AND home_key=$2 ORDER BY created_at DESC,id`, tenantID, s.scopeHomeKey())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TariffAssessment{}
	for rows.Next() {
		var item TariffAssessment
		var created string
		if err := rows.Scan(&item.ID, &item.TenantSlug, &item.HomeKey, &item.AssessmentMonth, &item.ProfileID, &item.ProfileVersion,
			&item.ProfileStatus, &item.SourceURL, &item.PeakKW, &item.BilledKW, &item.AnnualPowerEUR,
			&item.DataQuality, &created); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *SQLStore) UpsertMeasure(item Measure) error {
	if err := s.ensureProfile(item.TenantSlug); err != nil {
		return err
	}
	item.HomeKey = s.scopeHomeKey()
	normalized, err := NormalizeMeasure(item, time.Now())
	if err != nil {
		return err
	}
	shared, _ := json.Marshal(normalized.SharedFields)
	tenantID, err := tenantid.Ensure(s.db, normalized.TenantSlug)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO energy_measures
		(id,tenant_id,tenant_slug,home_key,issue_id,recommendation_id,title,status,contact_id,shared_fields_json,offer_note,appointment_at,
		 work_note,completed_at,evidence_note,before_from,before_to,after_from,after_to,before_peak_kw,after_peak_kw,
		 before_quality,after_quality,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
		ON CONFLICT(tenant_slug,home_key,issue_id) DO UPDATE SET
		recommendation_id=excluded.recommendation_id,title=excluded.title,status=excluded.status,contact_id=excluded.contact_id,
		shared_fields_json=excluded.shared_fields_json,offer_note=excluded.offer_note,appointment_at=excluded.appointment_at,
		work_note=excluded.work_note,completed_at=excluded.completed_at,evidence_note=excluded.evidence_note,
		before_from=excluded.before_from,before_to=excluded.before_to,after_from=excluded.after_from,after_to=excluded.after_to,
		before_peak_kw=excluded.before_peak_kw,after_peak_kw=excluded.after_peak_kw,
		before_quality=excluded.before_quality,after_quality=excluded.after_quality,
		tenant_id=coalesce(energy_measures.tenant_id,excluded.tenant_id),updated_at=excluded.updated_at`,
		normalized.ID, tenantID, normalized.TenantSlug, normalized.HomeKey, normalized.IssueID, normalized.RecommendationID, normalized.Title,
		normalized.Status, normalized.ContactID, string(shared), normalized.OfferNote, nullableTimePtr(normalized.AppointmentAt),
		normalized.WorkNote, nullableTimePtr(normalized.CompletedAt), normalized.EvidenceNote,
		nullableTimePtr(normalized.BeforeFrom), nullableTimePtr(normalized.BeforeTo), nullableTimePtr(normalized.AfterFrom), nullableTimePtr(normalized.AfterTo),
		nullableFloat(normalized.BeforePeakKW), nullableFloat(normalized.AfterPeakKW), normalized.BeforeQuality, normalized.AfterQuality,
		normalized.CreatedAt.Format(time.RFC3339Nano), normalized.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

func (s *SQLStore) GetMeasure(tenantSlug, id string) (Measure, bool, error) {
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return Measure{}, false, err
	}
	row := s.db.QueryRow(`SELECT id,tenant_slug,home_key,issue_id,recommendation_id,title,status,contact_id,shared_fields_json,offer_note,appointment_at,
		work_note,completed_at,evidence_note,before_from,before_to,after_from,after_to,before_peak_kw,after_peak_kw,
		before_quality,after_quality,created_at,updated_at
		FROM energy_measures WHERE tenant_id=$1 AND home_key=$2 AND id=$3`, tenantID, s.scopeHomeKey(), strings.TrimSpace(id))
	item, err := scanMeasure(row)
	if err == sql.ErrNoRows {
		return Measure{}, false, nil
	}
	return item, err == nil, err
}

func (s *SQLStore) ListMeasures(tenantSlug string) ([]Measure, error) {
	tenantID, err := s.scope(tenantSlug)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id,tenant_slug,home_key,issue_id,recommendation_id,title,status,contact_id,shared_fields_json,offer_note,appointment_at,
		work_note,completed_at,evidence_note,before_from,before_to,after_from,after_to,before_peak_kw,after_peak_kw,
		before_quality,after_quality,created_at,updated_at
		FROM energy_measures WHERE tenant_id=$1 AND home_key=$2 ORDER BY updated_at DESC,id`, tenantID, s.scopeHomeKey())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Measure{}
	for rows.Next() {
		item, scanErr := scanMeasure(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

type measureScanner interface {
	Scan(dest ...any) error
}

func scanMeasure(scanner measureScanner) (Measure, error) {
	var item Measure
	var shared string
	var appointment, completed, beforeFrom, beforeTo, afterFrom, afterTo sql.NullString
	var beforePeak, afterPeak sql.NullFloat64
	var created, updated string
	err := scanner.Scan(&item.ID, &item.TenantSlug, &item.HomeKey, &item.IssueID, &item.RecommendationID, &item.Title, &item.Status,
		&item.ContactID, &shared, &item.OfferNote, &appointment, &item.WorkNote, &completed, &item.EvidenceNote,
		&beforeFrom, &beforeTo, &afterFrom, &afterTo, &beforePeak, &afterPeak, &item.BeforeQuality, &item.AfterQuality,
		&created, &updated)
	if err != nil {
		return Measure{}, err
	}
	_ = json.Unmarshal([]byte(shared), &item.SharedFields)
	for _, candidate := range []struct {
		raw    string
		target **time.Time
	}{
		{appointment.String, &item.AppointmentAt},
		{completed.String, &item.CompletedAt},
		{beforeFrom.String, &item.BeforeFrom},
		{beforeTo.String, &item.BeforeTo},
		{afterFrom.String, &item.AfterFrom},
		{afterTo.String, &item.AfterTo},
	} {
		if parsed, ok := parseTime(candidate.raw); ok {
			value := parsed
			*candidate.target = &value
		}
	}
	if beforePeak.Valid {
		item.BeforePeakKW = &beforePeak.Float64
	}
	if afterPeak.Valid {
		item.AfterPeakKW = &afterPeak.Float64
	}
	item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return item, nil
}

func parseTime(raw string) (time.Time, bool) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, false
	}
	value, err := time.Parse(time.RFC3339Nano, raw)
	return value, err == nil
}

func nullableTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func nullableTimePtr(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func nullableFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
