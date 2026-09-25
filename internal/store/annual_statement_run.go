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

var ErrAnnualStatementArchivedDraft = errors.New("archived draft requires a new revision")

type AnnualStatementRunApproval struct {
	ApprovedAt   time.Time `json:"approved_at"`
	ApprovedBy   string    `json:"approved_by"`
	Role         string    `json:"role"`
	ApprovedName string    `json:"approved_name,omitempty"`
}

type AnnualStatementRun struct {
	Approval           *AnnualStatementRunApproval `json:"approval,omitempty"`
	ID                 string                      `json:"id"`
	PeriodYear         int                         `json:"period_year"`
	Revision           int                         `json:"revision"`
	CalculationVersion int                         `json:"calculation_version"`
	CreatedAt          time.Time                   `json:"created_at"`
	CreatedBy          string                      `json:"created_by"`
	InputHash          string                      `json:"input_hash"`
	Input              AnnualStatementRunInput     `json:"input"`
	Result             AnnualStatementRunResult    `json:"result"`
}

type AnnualStatementRunRepository interface {
	Approve(id, actor, role string, now time.Time, displayName ...string) (AnnualStatementRun, bool, error)
	// Preview reuses page-loaded vectors when supplied; nil loads current reports.
	Preview(year int, consumption map[string]AnnualStatementConsumptionVector) (AnnualStatementRunInput, AnnualStatementRunResult, error)
	Create(year int, actor string, now time.Time, presentation ...AnnualStatementRunPresentation) (AnnualStatementRun, error)
	List(year int) ([]AnnualStatementRun, error)
	Get(id string) (AnnualStatementRun, bool, error)
}

type AnnualStatementRunStorage interface{ annualStatementRunStorage() }
type annualStatementRunBackend interface {
	approveAnnualStatementRun(TenantRef, string, string, string, time.Time, ...string) (AnnualStatementRun, bool, error)
	previewAnnualStatementRun(TenantRef, int, map[string]AnnualStatementConsumptionVector) (AnnualStatementRunInput, AnnualStatementRunResult, error)
	createAnnualStatementRun(TenantRef, int, string, time.Time, ...AnnualStatementRunPresentation) (AnnualStatementRun, error)
	listAnnualStatementRuns(TenantRef, int) ([]AnnualStatementRun, error)
	getAnnualStatementRun(TenantRef, string) (AnnualStatementRun, bool, error)
}

type boundAnnualStatementRunRepository struct {
	storage annualStatementRunBackend
	tenant  TenantRef
}

func BindAnnualStatementRunRepository(storage AnnualStatementRunStorage, tenant TenantRef) (AnnualStatementRunRepository, bool) {
	backend, ok := storage.(annualStatementRunBackend)
	resolved, valid := validTenantRef(tenant)
	if !ok || !valid {
		return nil, false
	}
	return &boundAnnualStatementRunRepository{backend, resolved}, true
}
func (r *boundAnnualStatementRunRepository) Preview(year int, consumption map[string]AnnualStatementConsumptionVector) (AnnualStatementRunInput, AnnualStatementRunResult, error) {
	return r.storage.previewAnnualStatementRun(r.tenant, year, consumption)
}
func (r *boundAnnualStatementRunRepository) Create(year int, actor string, now time.Time, presentation ...AnnualStatementRunPresentation) (AnnualStatementRun, error) {
	return r.storage.createAnnualStatementRun(r.tenant, year, actor, now, presentation...)
}
func (r *boundAnnualStatementRunRepository) Get(id string) (AnnualStatementRun, bool, error) {
	return r.storage.getAnnualStatementRun(r.tenant, id)
}
func (r *boundAnnualStatementRunRepository) List(year int) ([]AnnualStatementRun, error) {
	return r.storage.listAnnualStatementRuns(r.tenant, year)
}

type AnnualStatementRunSources struct {
	Periods     AnnualStatementPeriodStorage
	Units       UnitStorage
	Receipts    AnnualStatementReceiptStorage
	Prepayments AnnualStatementPrepaymentStorage
	Reserve     AnnualStatementReserveStorage
	Consumption AnnualStatementConsumptionStorage
	Documents   DocumentStorage
}

type MemoryAnnualStatementRunStore struct {
	mu      sync.Mutex
	sources AnnualStatementRunSources
	runs    map[string][]AnnualStatementRun
}

func NewMemoryAnnualStatementRunStore(sources AnnualStatementRunSources) *MemoryAnnualStatementRunStore {
	return &MemoryAnnualStatementRunStore{sources: sources, runs: map[string][]AnnualStatementRun{}}
}
func (*MemoryAnnualStatementRunStore) annualStatementRunStorage() {}

func (s *MemoryAnnualStatementRunStore) previewAnnualStatementRun(tenant TenantRef, year int, consumption map[string]AnnualStatementConsumptionVector) (AnnualStatementRunInput, AnnualStatementRunResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	input, err := s.load(tenant, year, consumption)
	if err != nil {
		return input, AnnualStatementRunResult{}, err
	}
	annualPartyStatementDate(&input, time.Now())
	return evaluateAnnualStatementRun(input)
}
func (s *MemoryAnnualStatementRunStore) createAnnualStatementRun(tenant TenantRef, year int, actor string, now time.Time, presentation ...AnnualStatementRunPresentation) (AnnualStatementRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	input, err := s.load(tenant, year, nil)
	if err != nil {
		return AnnualStatementRun{}, err
	}
	annualPartyStatementDate(&input, now)
	input, result, err := evaluateAnnualStatementRun(input)
	if err != nil {
		return AnnualStatementRun{}, err
	}
	revision := 1
	for _, run := range s.runs[tenant.ID] {
		if run.PeriodYear == year && run.Revision >= revision {
			revision = run.Revision + 1
		}
	}
	if len(presentation) > 0 {
		input.Presentation = presentation[0]
	}
	run, err := newAnnualStatementRun(input, result, revision, actor, now)
	if err != nil {
		return AnnualStatementRun{}, err
	}
	s.runs[tenant.ID] = append(s.runs[tenant.ID], copyAnnualStatementRun(run))
	return run, nil
}
func (s *MemoryAnnualStatementRunStore) listAnnualStatementRuns(tenant TenantRef, year int) ([]AnnualStatementRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []AnnualStatementRun{}
	for _, run := range s.runs[tenant.ID] {
		if run.PeriodYear == year {
			out = append(out, copyAnnualStatementRun(run))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Revision > out[j].Revision })
	return out, nil
}
func (s *MemoryAnnualStatementRunStore) load(tenant TenantRef, year int, vectors map[string]AnnualStatementConsumptionVector) (AnnualStatementRunInput, error) {
	input := AnnualStatementRunInput{Consumption: map[string]AnnualStatementConsumptionVector{}}
	periods, pOK := BindAnnualStatementPeriodRepository(s.sources.Periods, tenant)
	units, uOK := BindUnitRepository(s.sources.Units, tenant)
	receipts, rOK := BindAnnualStatementReceiptRepository(s.sources.Receipts, tenant)
	prepayments, aOK := BindAnnualStatementPrepaymentRepository(s.sources.Prepayments, tenant)
	documents, dOK := BindDocumentRepository(s.sources.Documents, tenant)
	if !pOK || !uOK || !rOK || !aOK || !dOK {
		return input, fmt.Errorf("annual statement sources unavailable")
	}
	for _, period := range periods.List() {
		if period.Year == year {
			input.Period = period
		}
	}
	input.Structure, _ = periods.Structure(year)
	// Both callers hold s.mu; prior-run selection cannot race with creation.
	input.PreviousHeating = previousHeatingSnapshot(input.Period, s.runs[tenant.ID])
	listed, err := units.ListChecked()
	if err != nil {
		return input, annualStatementUnitDataBlock(err)
	}
	for _, unit := range listed {
		input.Units = append(input.Units, AnnualStatementRunUnitIdentity{ID: unit.ID, Label: unit.Label, UnitType: NormalizeUnitType(unit.UnitType)})
		input.Parties = append(input.Parties, annualStatementRunParties(unit, input.Structure.Legal.Regime)...)
	}
	input.Receipts = receipts.ListByPeriod(year)
	input.Prepayments = prepayments.ListByPeriod(year)
	referenced := map[string]bool{}
	for _, receipt := range input.Receipts {
		referenced[receipt.DocumentID] = true
	}
	if reserve, ok := BindAnnualStatementReserveRepository(s.sources.Reserve, tenant); ok {
		input.Reserve = reserve.ListByPeriod(year)
		for _, entry := range input.Reserve {
			if entry.DocumentID != "" {
				referenced[entry.DocumentID] = true
			}
		}
	}
	for _, doc := range documents.List() {
		if referenced[doc.ID] && annualStatementRunDocumentReadable(doc, documents) {
			input.Documents = append(input.Documents, AnnualStatementRunDocument{doc.ID, doc.Title, doc.Filename})
		}
	}
	// The page already loaded these reports for its consumption panel. Creation
	// always passes nil and reads fresh reports as part of its own input load.
	if vectors != nil && !hasDatedAnnualParties(input) {
		input.Consumption = vectors
		return input, nil
	}
	if consumption, ok := BindAnnualStatementConsumptionRepository(s.sources.Consumption, tenant); ok && input.Period.Year != 0 {
		location, err := time.LoadLocation("Europe/Vienna")
		if err != nil {
			return input, err
		}
		ids := []string{}
		for _, unit := range input.Units {
			ids = append(ids, unit.ID)
		}
		for _, cost := range input.Structure.CostTypes {
			if !cost.Allocatable || cost.AllocationKey != AllocationKeyVerbrauch {
				continue
			}
			report, err := consumption.ConsumptionReport(input.Period, cost.Key, ids, location)
			if errors.Is(err, ErrAnnualStatementConsumptionInvalidQuery) {
				continue
			}
			if err != nil {
				return input, err
			}
			input.Consumption[cost.Key] = report.Vector
			input.Evidence = append(input.Evidence, report.BoundaryEvidence...)
			if backend, ok := s.sources.Consumption.(annualStatementConsumptionBackend); ok {
				for id, dates := range annualPartyReadingDates(input) {
					for _, date := range dates {
						readings, err := backend.listAnnualStatementConsumption(tenant, cost.Key, date, date)
						if err != nil {
							return input, err
						}
						for _, reading := range readings {
							if reading.UnitID == id {
								input.PartyEvidence = append(input.PartyEvidence, reading)
							}
						}
					}
				}
			}
		}
	}
	return input, nil
}

func annualStatementRunDocumentReadable(doc DocumentRecord, repository DocumentRepository) bool {
	if !annualStatementReceiptDocumentTypeSupported(doc.ContentType) {
		return false
	}
	path, ok := repository.FilePath(doc)
	if !ok {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	return err == nil && info.Mode().IsRegular() && info.Size() > 0 && info.Size() == doc.Size
}
func annualStatementUnitDataBlock(err error) error {
	var dataErr *UnitDataError
	if !errors.As(err, &dataErr) {
		return err
	}
	return &AnnualStatementRunBlockedError{Issues: []AnnualStatementRunIssue{{Code: "unit-data", UnitID: dataErr.UnitID}}}
}

func evaluateAnnualStatementRun(input AnnualStatementRunInput) (AnnualStatementRunInput, AnnualStatementRunResult, error) {
	result, issues := CalculateAnnualStatementRun(input)
	if len(issues) > 0 {
		return input, AnnualStatementRunResult{}, &AnnualStatementRunBlockedError{issues}
	}
	return input, result, nil
}
func newAnnualStatementRun(input AnnualStatementRunInput, result AnnualStatementRunResult, revision int, actor string, now time.Time) (AnnualStatementRun, error) {
	actor = strings.ToLower(strings.TrimSpace(actor))
	if actor == "" || now.IsZero() {
		return AnnualStatementRun{}, fmt.Errorf("annual statement run requires actor and timestamp")
	}
	id, err := randomToken(16)
	if err != nil {
		return AnnualStatementRun{}, err
	}
	// Own the complete snapshot before canonicalising slices and nested maps.
	// Sorting must neither mutate source data nor leave aliases in a run.
	raw, err := json.Marshal(input)
	if err != nil {
		return AnnualStatementRun{}, err
	}
	var snapshot AnnualStatementRunInput
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return AnnualStatementRun{}, err
	}
	input = snapshot
	sort.Slice(input.Parties, func(i, j int) bool {
		if input.Parties[i].UnitID != input.Parties[j].UnitID {
			return input.Parties[i].UnitID < input.Parties[j].UnitID
		}
		return input.Parties[i].ID < input.Parties[j].ID
	})
	// Stable ordering makes input hashes inspectable and independent of SQL row order.
	sort.Slice(input.Units, func(i, j int) bool { return input.Units[i].ID < input.Units[j].ID })
	sort.Slice(input.Structure.CostTypes, func(i, j int) bool { return input.Structure.CostTypes[i].Key < input.Structure.CostTypes[j].Key })
	sort.Slice(input.Structure.UnitBases, func(i, j int) bool { return input.Structure.UnitBases[i].UnitID < input.Structure.UnitBases[j].UnitID })
	sort.Slice(input.Receipts, func(i, j int) bool { return input.Receipts[i].ID < input.Receipts[j].ID })
	sortAnnualStatementReserveEntries(input.Reserve)
	sort.Slice(input.Prepayments, func(i, j int) bool { return input.Prepayments[i].UnitID < input.Prepayments[j].UnitID })
	sort.Slice(input.Documents, func(i, j int) bool { return input.Documents[i].ID < input.Documents[j].ID })
	sort.Slice(input.PartyEvidence, func(i, j int) bool { return input.PartyEvidence[i].SourceKey < input.PartyEvidence[j].SourceKey })
	sort.Slice(input.Evidence, func(i, j int) bool { return input.Evidence[i].SourceKey < input.Evidence[j].SourceKey })
	for key, vector := range input.Consumption {
		sort.Slice(vector.Units, func(i, j int) bool { return vector.Units[i].UnitID < vector.Units[j].UnitID })
		input.Consumption[key] = vector
	}
	if input.PreviousHeating != nil {
		for key, vector := range input.PreviousHeating.Consumption {
			sort.Slice(vector.Units, func(i, j int) bool { return vector.Units[i].UnitID < vector.Units[j].UnitID })
			input.PreviousHeating.Consumption[key] = vector
		}
	}
	raw, err = json.Marshal(input)
	if err != nil {
		return AnnualStatementRun{}, err
	}
	hash := sha256.Sum256(raw)
	version := AnnualStatementCalculationVersion
	if input.Structure.Legal.ShowVAT {
		version = AnnualStatementCalculationVersionVAT
	}
	if hasDatedAnnualParties(input) {
		version = AnnualStatementCalculationVersionParties
	}
	if annualStatementHasAgreedShares(input) {
		version = AnnualStatementCalculationVersionAgreed
	}
	if result.Reserve != nil {
		version = AnnualStatementCalculationVersionReserveRates
	}
	return AnnualStatementRun{ID: id, PeriodYear: input.Period.Year, Revision: revision, CalculationVersion: version, CreatedAt: now.UTC(), CreatedBy: actor, InputHash: hex.EncodeToString(hash[:]), Input: input, Result: result}, nil
}
func copyAnnualStatementRun(run AnnualStatementRun) AnnualStatementRun {
	raw, _ := json.Marshal(run)
	var copy AnnualStatementRun
	_ = json.Unmarshal(raw, &copy)
	return copy
}

func (s *MemoryAnnualStatementRunStore) getAnnualStatementRun(tenant TenantRef, id string) (AnnualStatementRun, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, run := range s.runs[tenant.ID] {
		if run.ID == id {
			return copyAnnualStatementRun(run), true, nil
		}
	}
	return AnnualStatementRun{}, false, nil
}

func (r *boundAnnualStatementRunRepository) Approve(id, actor, role string, now time.Time, displayName ...string) (AnnualStatementRun, bool, error) {
	return r.storage.approveAnnualStatementRun(r.tenant, id, actor, role, now, displayName...)
}
func (s *MemoryAnnualStatementRunStore) approveAnnualStatementRun(tenant TenantRef, id, actor, role string, now time.Time, displayName ...string) (AnnualStatementRun, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, run := range s.runs[tenant.ID] {
		if run.ID != id {
			continue
		}
		if run.Approval != nil {
			return copyAnnualStatementRun(run), false, nil
		}
		approval, err := newAnnualStatementApproval(run, actor, role, now, displayName...)
		if err != nil {
			return AnnualStatementRun{}, false, err
		}
		if docs, ok := BindDocumentRepository(s.sources.Documents, tenant); ok {
			if err := annualStatementApprovalArchiveCheck(run, docs); err != nil {
				return AnnualStatementRun{}, false, err
			}
		}
		run.Approval = approval
		s.runs[tenant.ID][i] = copyAnnualStatementRun(run)
		return run, true, nil
	}
	return AnnualStatementRun{}, false, fmt.Errorf("annual statement run not found")
}
func newAnnualStatementApproval(run AnnualStatementRun, actor, role string, now time.Time, displayName ...string) (*AnnualStatementRunApproval, error) {
	actor = strings.ToLower(strings.TrimSpace(actor))
	if actor == "" || (role != RoleManager && role != RoleAdmin) || now.IsZero() || now.Before(run.CreatedAt) {
		return nil, fmt.Errorf("invalid annual statement approval")
	}
	name := ""
	if len(displayName) > 0 {
		name = strings.TrimSpace(displayName[0])
	}
	return &AnnualStatementRunApproval{ApprovedAt: now.UTC(), ApprovedBy: actor, Role: role, ApprovedName: name}, nil
}
func annualStatementApprovalArchiveCheck(run AnnualStatementRun, docs DocumentRepository) error {
	for _, doc := range docs.List() {
		if doc.AnnualStatementArchive != nil && doc.AnnualStatementArchive.RunID == run.ID {
			return ErrAnnualStatementArchivedDraft
		}
	}
	return nil
}
