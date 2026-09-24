package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	ReserveKindOpening      = "opening"
	ReserveKindContribution = "contribution"
	ReserveKindWithdrawal   = "withdrawal"
	ReserveKindInterest     = "interest"
	ReserveKindClosingCheck = "closing_check"
)

// WEGMinimumReserveCentsPerSquareMetreMonth is the statutory floor from
// 1 January 2026: 1,12 € per m² of usable area per month.
// WEG 2002 § 31 Abs 5 valorises 0,90 € by VPI 2020:
// 0,90 × 128,1 (June 2025) / 102,6 = 1,1237, stated as 1,12.
// Source: WKO/ÖVI, law-report example E6.
const WEGMinimumReserveCentsPerSquareMetreMonth = 112

// AnnualStatementReserveEntry is an insert-only booking on the WEG Rücklage.
// A correction is a later entry; stored rows are not updated.
type AnnualStatementReserveEntry struct {
	ID          string    `json:"id"`
	PeriodYear  int       `json:"period_year"`
	Kind        string    `json:"kind"`
	EntryDate   string    `json:"entry_date"`
	AmountCents int64     `json:"amount_cents"`
	DocumentID  string    `json:"document_id,omitempty"`
	Note        string    `json:"note,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
}

type AnnualStatementReserveShare struct {
	UnitID      string `json:"unit_id"`
	SharePPM    int    `json:"share_ppm"`
	AmountCents int64  `json:"amount_cents"`
}

// AnnualStatementReserveResult is the cent-exact Rücklage snapshot of one run.
type AnnualStatementReserveResult struct {
	OpeningCents        int64                         `json:"opening_cents"`
	ContributionCents   int64                         `json:"contribution_cents"`
	WithdrawalCents     int64                         `json:"withdrawal_cents"`
	InterestCents       int64                         `json:"interest_cents"`
	ClosingCents        int64                         `json:"closing_cents"`
	Shares              []AnnualStatementReserveShare `json:"shares,omitempty"`
	MinimumMonthlyCents int64                         `json:"minimum_monthly_cents,omitempty"`
	MinimumPeriodCents  int64                         `json:"minimum_period_cents,omitempty"`
	// MinimumWarning is set when recorded contributions are below the statutory floor.
	// AreaIncomplete means the floor could not be computed.
	MinimumWarning bool `json:"minimum_warning,omitempty"`
	AreaIncomplete bool `json:"area_incomplete,omitempty"`
	// ClosingMismatch is set when a closing_check entry differs from the computed closing.
	ClosingMismatch bool `json:"closing_mismatch,omitempty"`
}

type AnnualStatementReserveRepository interface {
	Add(entry AnnualStatementReserveEntry) (AnnualStatementReserveEntry, error)
	ListByPeriod(year int) []AnnualStatementReserveEntry
}

type AnnualStatementReserveStorage interface{ annualStatementReserveStorage() }

type annualStatementReserveBackend interface {
	addAnnualStatementReserveEntry(TenantRef, AnnualStatementReserveEntry) (AnnualStatementReserveEntry, error)
	listAnnualStatementReserveEntries(TenantRef, int) []AnnualStatementReserveEntry
}

type boundAnnualStatementReserveRepository struct {
	storage annualStatementReserveBackend
	tenant  TenantRef
}

func BindAnnualStatementReserveRepository(storage AnnualStatementReserveStorage, tenant TenantRef) (AnnualStatementReserveRepository, bool) {
	backend, ok := storage.(annualStatementReserveBackend)
	resolved, tenantOK := validTenantRef(tenant)
	if !ok || !tenantOK {
		return nil, false
	}
	return &boundAnnualStatementReserveRepository{storage: backend, tenant: resolved}, true
}

func (r *boundAnnualStatementReserveRepository) Add(entry AnnualStatementReserveEntry) (AnnualStatementReserveEntry, error) {
	return r.storage.addAnnualStatementReserveEntry(r.tenant, entry)
}

func (r *boundAnnualStatementReserveRepository) ListByPeriod(year int) []AnnualStatementReserveEntry {
	return r.storage.listAnnualStatementReserveEntries(r.tenant, year)
}

type MemoryAnnualStatementReserveStore struct {
	mu        sync.Mutex
	byHome    map[string]map[string]AnnualStatementReserveEntry
	periods   AnnualStatementPeriodStorage
	documents DocumentStorage
}

func NewMemoryAnnualStatementReserveStore(periods AnnualStatementPeriodStorage, documents DocumentStorage) *MemoryAnnualStatementReserveStore {
	return &MemoryAnnualStatementReserveStore{byHome: map[string]map[string]AnnualStatementReserveEntry{}, periods: periods, documents: documents}
}

func (*MemoryAnnualStatementReserveStore) annualStatementReserveStorage() {}

func (s *MemoryAnnualStatementReserveStore) addAnnualStatementReserveEntry(tenant TenantRef, entry AnnualStatementReserveEntry) (AnnualStatementReserveEntry, error) {
	periods, periodsOK := BindAnnualStatementPeriodRepository(s.periods, tenant)
	documents, documentsOK := BindDocumentRepository(s.documents, tenant)
	if !periodsOK || !documentsOK {
		return AnnualStatementReserveEntry{}, fmt.Errorf("annual statement reserve sources unavailable")
	}
	entry, err := normalizeAnnualStatementReserveEntry(entry, periods, documents)
	if err != nil {
		return AnnualStatementReserveEntry{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byHome[tenant.ID] == nil {
		s.byHome[tenant.ID] = map[string]AnnualStatementReserveEntry{}
	}
	if _, exists := s.byHome[tenant.ID][entry.ID]; exists {
		return AnnualStatementReserveEntry{}, fmt.Errorf("annual statement reserve entry exists")
	}
	s.byHome[tenant.ID][entry.ID] = entry
	return entry, nil
}

func (s *MemoryAnnualStatementReserveStore) listAnnualStatementReserveEntries(tenant TenantRef, year int) []AnnualStatementReserveEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []AnnualStatementReserveEntry{}
	for _, entry := range s.byHome[tenant.ID] {
		if year == 0 || entry.PeriodYear == year {
			out = append(out, entry)
		}
	}
	sortAnnualStatementReserveEntries(out)
	return out
}

type SQLAnnualStatementReserveStore struct{ db *TenantDB }

func NewSQLAnnualStatementReserveStore(db *TenantDB) *SQLAnnualStatementReserveStore {
	return &SQLAnnualStatementReserveStore{db: db}
}

func (*SQLAnnualStatementReserveStore) annualStatementReserveStorage() {}

func (s *SQLAnnualStatementReserveStore) addAnnualStatementReserveEntry(tenant TenantRef, entry AnnualStatementReserveEntry) (AnnualStatementReserveEntry, error) {
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return AnnualStatementReserveEntry{}, err
	}
	defer tx.Rollback()
	var starts, ends, legalRaw string
	err = tx.QueryRow(`SELECT starts_on, ends_on, legal_settings FROM annual_statement_periods WHERE tenant_id=$1 AND year=$2`, tenant.ID, entry.PeriodYear).Scan(&starts, &ends, &legalRaw)
	if err != nil {
		return AnnualStatementReserveEntry{}, fmt.Errorf("annual statement reserve period: %w", err)
	}
	var legal AnnualStatementLegalSettings
	if err := json.Unmarshal([]byte(legalRaw), &legal); err != nil || legal.Validate() != nil {
		return AnnualStatementReserveEntry{}, fmt.Errorf("annual statement reserve period is invalid")
	}
	if legal.Regime != "weg" {
		return AnnualStatementReserveEntry{}, fmt.Errorf("annual statement reserve is only recorded for WEG")
	}
	entry, err = prepareAnnualStatementReserveEntry(entry, starts, ends)
	if err != nil {
		return AnnualStatementReserveEntry{}, err
	}
	if entry.Kind == ReserveKindWithdrawal {
		var documentID string
		if err := tx.QueryRow(`SELECT id FROM documents WHERE tenant_id=$1 AND id=$2`, tenant.ID, entry.DocumentID).Scan(&documentID); err != nil {
			return AnnualStatementReserveEntry{}, fmt.Errorf("annual statement reserve withdrawal requires a document")
		}
	}
	_, err = tx.Exec(`INSERT INTO annual_statement_reserve_entries(
		tenant_id, tenant_slug, id, period_year, kind, entry_date, amount_cents, document_id, note, created_at, created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		tenant.ID, tenant.Slug, entry.ID, entry.PeriodYear, entry.Kind, entry.EntryDate, entry.AmountCents, entry.DocumentID, entry.Note, entry.CreatedAt.Format(time.RFC3339Nano), entry.CreatedBy)
	if err != nil {
		return AnnualStatementReserveEntry{}, err
	}
	return entry, tx.Commit()
}

func (s *SQLAnnualStatementReserveStore) listAnnualStatementReserveEntries(tenant TenantRef, year int) []AnnualStatementReserveEntry {
	query := `SELECT id, period_year, kind, entry_date, amount_cents, document_id, note, created_at, created_by
		FROM annual_statement_reserve_entries WHERE tenant_id=$1`
	args := []any{tenant.ID}
	if year != 0 {
		query += ` AND period_year=$2`
		args = append(args, year)
	}
	query += ` ORDER BY entry_date, id`
	rows, err := s.db.For(tenant).Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []AnnualStatementReserveEntry{}
	for rows.Next() {
		entry, err := scanAnnualStatementReserveEntry(rows)
		if err != nil {
			return nil
		}
		out = append(out, entry)
	}
	if rows.Err() != nil {
		return nil
	}
	return out
}

func scanAnnualStatementReserveEntry(rows *sql.Rows) (AnnualStatementReserveEntry, error) {
	var entry AnnualStatementReserveEntry
	var created string
	if err := rows.Scan(&entry.ID, &entry.PeriodYear, &entry.Kind, &entry.EntryDate, &entry.AmountCents, &entry.DocumentID, &entry.Note, &created, &entry.CreatedBy); err != nil {
		return AnnualStatementReserveEntry{}, err
	}
	at, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return AnnualStatementReserveEntry{}, err
	}
	entry.CreatedAt = at
	return entry, nil
}

func normalizeAnnualStatementReserveEntry(entry AnnualStatementReserveEntry, periods AnnualStatementPeriodRepository, documents DocumentRepository) (AnnualStatementReserveEntry, error) {
	var period AnnualStatementPeriod
	found := false
	for _, item := range periods.List() {
		if item.Year == entry.PeriodYear {
			period, found = item, true
		}
	}
	if !found {
		return AnnualStatementReserveEntry{}, fmt.Errorf("annual statement reserve period is missing")
	}
	structure, ok := periods.Structure(entry.PeriodYear)
	if !ok || structure.Legal.Regime != "weg" {
		return AnnualStatementReserveEntry{}, fmt.Errorf("annual statement reserve is only recorded for WEG")
	}
	entry, err := prepareAnnualStatementReserveEntry(entry, period.StartsOn, period.EndsOn)
	if err != nil {
		return AnnualStatementReserveEntry{}, err
	}
	if entry.Kind == ReserveKindWithdrawal {
		if _, ok := documents.Get(entry.DocumentID); !ok {
			return AnnualStatementReserveEntry{}, fmt.Errorf("annual statement reserve withdrawal requires a document")
		}
	}
	return entry, nil
}

func prepareAnnualStatementReserveEntry(entry AnnualStatementReserveEntry, startsOn, endsOn string) (AnnualStatementReserveEntry, error) {
	switch entry.Kind {
	case ReserveKindOpening, ReserveKindContribution, ReserveKindWithdrawal, ReserveKindInterest, ReserveKindClosingCheck:
	default:
		return AnnualStatementReserveEntry{}, fmt.Errorf("invalid annual statement reserve kind")
	}
	entry.EntryDate = strings.TrimSpace(entry.EntryDate)
	if _, err := time.Parse("2006-01-02", entry.EntryDate); err != nil || entry.EntryDate < startsOn || entry.EntryDate > endsOn {
		return AnnualStatementReserveEntry{}, fmt.Errorf("invalid annual statement reserve date")
	}
	if entry.AmountCents == 0 {
		return AnnualStatementReserveEntry{}, fmt.Errorf("invalid annual statement reserve amount")
	}
	entry.Note = strings.TrimSpace(entry.Note)
	if len(entry.Note) > 500 {
		return AnnualStatementReserveEntry{}, fmt.Errorf("annual statement reserve note is too long")
	}
	entry.DocumentID = strings.TrimSpace(entry.DocumentID)
	if entry.Kind == ReserveKindWithdrawal {
		if entry.DocumentID == "" {
			return AnnualStatementReserveEntry{}, fmt.Errorf("annual statement reserve withdrawal requires a document")
		}
	} else {
		entry.DocumentID = ""
	}
	entry.CreatedBy = strings.ToLower(strings.TrimSpace(entry.CreatedBy))
	if entry.CreatedBy == "" {
		return AnnualStatementReserveEntry{}, fmt.Errorf("annual statement reserve requires an actor")
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	entry.CreatedAt = entry.CreatedAt.UTC()
	if strings.TrimSpace(entry.ID) == "" {
		id, err := randomToken(16)
		if err != nil {
			return AnnualStatementReserveEntry{}, err
		}
		entry.ID = id
	}
	return entry, nil
}

func sortAnnualStatementReserveEntries(entries []AnnualStatementReserveEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].EntryDate != entries[j].EntryDate {
			return entries[i].EntryDate < entries[j].EntryDate
		}
		return entries[i].ID < entries[j].ID
	})
}

func reserveAdd(sum *int64, value int64) bool {
	if value > 0 && *sum > math.MaxInt64-value {
		return false
	}
	if value < 0 && *sum < math.MinInt64-value {
		return false
	}
	*sum += value
	return true
}

// AnnualStatementReserveBalance closes the Rücklage as
// opening + contributions − withdrawals + interest.
// Owner shares reuse the Nutzwert basis and the cent allocator.
// The minimum check warns only; it never blocks the run.
func AnnualStatementReserveBalance(entries []AnnualStatementReserveEntry, period AnnualStatementPeriod, units []Unit) (AnnualStatementReserveResult, bool) {
	var result AnnualStatementReserveResult
	var closingCheck int64
	hasCheck := false
	for _, entry := range entries {
		switch entry.Kind {
		case ReserveKindOpening:
			if !reserveAdd(&result.OpeningCents, entry.AmountCents) {
				return AnnualStatementReserveResult{}, false
			}
		case ReserveKindContribution:
			if !reserveAdd(&result.ContributionCents, entry.AmountCents) {
				return AnnualStatementReserveResult{}, false
			}
		case ReserveKindWithdrawal:
			if !reserveAdd(&result.WithdrawalCents, entry.AmountCents) {
				return AnnualStatementReserveResult{}, false
			}
		case ReserveKindInterest:
			if !reserveAdd(&result.InterestCents, entry.AmountCents) {
				return AnnualStatementReserveResult{}, false
			}
		case ReserveKindClosingCheck:
			if !reserveAdd(&closingCheck, entry.AmountCents) {
				return AnnualStatementReserveResult{}, false
			}
			hasCheck = true
		default:
			return AnnualStatementReserveResult{}, false
		}
	}
	result.ClosingCents = result.OpeningCents
	if !reserveAdd(&result.ClosingCents, result.ContributionCents) || !reserveAdd(&result.ClosingCents, -result.WithdrawalCents) || !reserveAdd(&result.ClosingCents, result.InterestCents) {
		return AnnualStatementReserveResult{}, false
	}
	if hasCheck && closingCheck != result.ClosingCents {
		result.ClosingMismatch = true
	}
	applyReserveMinimum(&result, period, units)
	applyReserveShares(&result, units)
	return result, true
}

func applyReserveMinimum(result *AnnualStatementReserveResult, period AnnualStatementPeriod, units []Unit) {
	area, complete := 0, len(units) > 0
	for _, unit := range units {
		if !unit.UsableAreaRecorded || unit.UsableAreaM2Hundredths < 0 {
			complete = false
			continue
		}
		if unit.UsableAreaM2Hundredths > math.MaxInt-area {
			complete = false
			continue
		}
		area += unit.UsableAreaM2Hundredths
	}
	months, monthsOK := reservePeriodMonths(period.StartsOn, period.EndsOn)
	if !complete || !monthsOK {
		result.AreaIncomplete = true
		result.MinimumWarning = true
		return
	}
	monthly := reserveMinimumMonthlyCents(area)
	result.MinimumMonthlyCents = monthly
	periodMinimum := monthly * int64(months)
	result.MinimumPeriodCents = periodMinimum
	result.MinimumWarning = result.ContributionCents < periodMinimum
}

// reserveMinimumMonthlyCents applies 1,12 €/m² to an area stored in hundredths of a square metre.
// A remainder of 0,50 cent rounds down, matching the statement's half-down money rule.
func reserveMinimumMonthlyCents(areaHundredths int) int64 {
	product := int64(WEGMinimumReserveCentsPerSquareMetreMonth) * int64(areaHundredths)
	cents := product / 100
	if product%100 > 50 {
		cents++
	}
	return cents
}

func reservePeriodMonths(startsOn, endsOn string) (int, bool) {
	start, errStart := time.Parse("2006-01-02", startsOn)
	end, errEnd := time.Parse("2006-01-02", endsOn)
	if errStart != nil || errEnd != nil || end.Before(start) {
		return 0, false
	}
	months := (end.Year()-start.Year())*12 + int(end.Month()-start.Month()) + 1
	return months, months > 0
}

func applyReserveShares(result *AnnualStatementReserveResult, units []Unit) {
	if result.ClosingCents < 0 {
		return
	}
	previews := AnnualStatementAllocationPreviews([]AnnualStatementCostType{{Key: "ruecklage", Name: "Rücklage", Allocatable: true, AllocationKey: AllocationKeyNutzwert}}, units)
	if len(previews) != 1 || previews[0].Blocked || previews[0].BasisTotal != MiteigentumsanteilTotalPPM {
		return
	}
	cents := annualStatementRunCents(previews[0].Shares, result.ClosingCents)
	result.Shares = make([]AnnualStatementReserveShare, 0, len(previews[0].Shares))
	for i, share := range previews[0].Shares {
		result.Shares = append(result.Shares, AnnualStatementReserveShare{UnitID: share.UnitID, SharePPM: share.SharePPM, AmountCents: cents[i]})
	}
}
