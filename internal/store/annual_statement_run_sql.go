package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type SQLAnnualStatementRunStore struct {
	db        *TenantDB
	documents DocumentStorage
}

func NewSQLAnnualStatementRunStore(db *TenantDB, documents DocumentStorage) *SQLAnnualStatementRunStore {
	return &SQLAnnualStatementRunStore{db, documents}
}
func (*SQLAnnualStatementRunStore) annualStatementRunStorage() {}

// The lane establishes tenant scope before BeginTx. Serializable isolation
// gives the calculation one coherent input snapshot, including empty sets.
func (s *SQLAnnualStatementRunStore) begin(tenant TenantRef, readOnly bool) (*sql.Tx, error) {
	handle, ok := s.db.For(tenant).(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return nil, fmt.Errorf("annual statement snapshot transactions unavailable")
	}
	level := sql.LevelSerializable
	if readOnly {
		level = sql.LevelRepeatableRead
	}
	return handle.BeginTx(context.Background(), &sql.TxOptions{Isolation: level, ReadOnly: readOnly})
}
func (s *SQLAnnualStatementRunStore) previewAnnualStatementRun(tenant TenantRef, year int, consumption map[string]AnnualStatementConsumptionVector) (AnnualStatementRunInput, AnnualStatementRunResult, error) {
	tx, err := s.begin(tenant, true)
	if err != nil {
		return AnnualStatementRunInput{}, AnnualStatementRunResult{}, err
	}
	defer tx.Rollback()
	input, err := s.load(tx, tenant, year, consumption)
	if err != nil {
		return input, AnnualStatementRunResult{}, err
	}
	return evaluateAnnualStatementRun(input)
}
func (s *SQLAnnualStatementRunStore) createAnnualStatementRun(tenant TenantRef, year int, actor string, now time.Time, presentation ...AnnualStatementRunPresentation) (AnnualStatementRun, error) {
	tx, err := s.begin(tenant, false)
	if err != nil {
		return AnnualStatementRun{}, err
	}
	defer tx.Rollback()
	input, err := s.load(tx, tenant, year, nil)
	if err != nil {
		return AnnualStatementRun{}, err
	}
	input, result, err := evaluateAnnualStatementRun(input)
	if err != nil {
		return AnnualStatementRun{}, err
	}
	var revision int
	if err := tx.QueryRow(`SELECT COALESCE(MAX(revision),0)+1 FROM annual_statement_runs WHERE tenant_id=$1 AND period_year=$2`, tenant.ID, year).Scan(&revision); err != nil {
		return AnnualStatementRun{}, err
	}
	if len(presentation) > 0 {
		input.Presentation = presentation[0]
	}
	run, err := newAnnualStatementRun(input, result, revision, actor, now)
	if err != nil {
		return AnnualStatementRun{}, err
	}
	raw, err := json.Marshal(run)
	if err != nil {
		return AnnualStatementRun{}, err
	}
	if _, err := tx.Exec(`INSERT INTO annual_statement_runs(tenant_id,tenant_slug,id,period_year,revision,data) VALUES($1,$2,$3,$4,$5,$6)`, tenant.ID, tenant.Slug, run.ID, year, revision, string(raw)); err != nil {
		return AnnualStatementRun{}, err
	}
	if err := tx.Commit(); err != nil {
		return AnnualStatementRun{}, err
	}
	return run, nil
}
func (s *SQLAnnualStatementRunStore) listAnnualStatementRuns(tenant TenantRef, year int) ([]AnnualStatementRun, error) {
	rows, err := s.db.For(tenant).Query(`SELECT data FROM annual_statement_runs WHERE tenant_id=$1 AND period_year=$2 ORDER BY revision DESC`, tenant.ID, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AnnualStatementRun{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var run AnnualStatementRun
		if err := json.Unmarshal([]byte(raw), &run); err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

type annualStatementRunQueryer interface {
	Query(string, ...any) (*sql.Rows, error)
	QueryRow(string, ...any) *sql.Row
}

func (s *SQLAnnualStatementRunStore) load(tx annualStatementRunQueryer, tenant TenantRef, year int, vectors map[string]AnnualStatementConsumptionVector) (AnnualStatementRunInput, error) {
	input := AnnualStatementRunInput{Consumption: map[string]AnnualStatementConsumptionVector{}}
	var updatedAt string
	err := tx.QueryRow(`SELECT year,starts_on,ends_on,updated_at,updated_by FROM annual_statement_periods WHERE tenant_id=$1 AND year=$2`, tenant.ID, year).Scan(&input.Period.Year, &input.Period.StartsOn, &input.Period.EndsOn, &updatedAt, &input.Period.UpdatedBy)
	if errors.Is(err, sql.ErrNoRows) {
		return input, nil
	}
	if err != nil {
		return input, err
	}
	input.Period.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return input, err
	}
	read := func(query string, scan func(*sql.Rows) error, args ...any) error {
		rows, err := tx.Query(query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			if err := scan(rows); err != nil {
				return err
			}
		}
		return rows.Err()
	}
	err = read(`SELECT key,name,allocatable,allocation_key,updated_at,updated_by FROM annual_statement_period_cost_types WHERE tenant_id=$1 AND period_year=$2 ORDER BY key`, func(rows *sql.Rows) error {
		var item AnnualStatementCostType
		var updated string
		if err := rows.Scan(&item.Key, &item.Name, &item.Allocatable, &item.AllocationKey, &updated, &item.UpdatedBy); err != nil {
			return err
		}
		at, err := time.Parse(time.RFC3339Nano, updated)
		if err != nil {
			return err
		}
		item.UpdatedAt = at
		input.Structure.CostTypes = append(input.Structure.CostTypes, item)
		return nil
	}, tenant.ID, year)
	if err != nil {
		return input, err
	}
	err = read(`SELECT unit_id,miteigentumsanteil_ppm,usable_area_m2_hundredths,usable_area_recorded,persons,persons_recorded FROM annual_statement_period_unit_bases WHERE tenant_id=$1 AND period_year=$2 ORDER BY unit_id`, func(rows *sql.Rows) error {
		var item AnnualStatementPeriodUnitBasis
		if err := rows.Scan(&item.UnitID, &item.MiteigentumsanteilPPM, &item.UsableAreaM2Hundredths, &item.UsableAreaRecorded, &item.Persons, &item.PersonsRecorded); err != nil {
			return err
		}
		input.Structure.UnitBases = append(input.Structure.UnitBases, item)
		return nil
	}, tenant.ID, year)
	if err != nil {
		return input, err
	}
	err = read(`SELECT id,data FROM units WHERE tenant_id=$1 ORDER BY id`, func(rows *sql.Rows) error {
		var id, raw string
		if err := rows.Scan(&id, &raw); err != nil {
			return err
		}
		var unit Unit
		if err := json.Unmarshal([]byte(raw), &unit); err != nil {
			return err
		}
		if unit.ID != id {
			return fmt.Errorf("annual statement unit identity mismatch")
		}
		input.Units = append(input.Units, AnnualStatementRunUnitIdentity{ID: unit.ID, Label: unit.Label, UnitType: NormalizeUnitType(unit.UnitType)})
		input.Parties = append(input.Parties, annualStatementRunParties(unit)...)
		return nil
	}, tenant.ID)
	if err != nil {
		return input, err
	}
	err = read(`SELECT id,document_id,period_year,cost_type_key,amount_cents,invoice_date,created_at,created_by,updated_at,updated_by FROM annual_statement_receipts WHERE tenant_id=$1 AND period_year=$2 ORDER BY id`, func(rows *sql.Rows) error {
		item, err := scanAnnualStatementReceipt(rows)
		if err != nil {
			return err
		}
		input.Receipts = append(input.Receipts, item)
		return nil
	}, tenant.ID, year)
	if err != nil {
		return input, err
	}
	err = read(`SELECT period_year,unit_id,amount_cents,updated_at,updated_by FROM annual_statement_prepayments WHERE tenant_id=$1 AND period_year=$2 ORDER BY unit_id`, func(rows *sql.Rows) error {
		item, _, err := getAnnualStatementPrepaymentQuery(rows)
		if err != nil {
			return err
		}
		input.Prepayments = append(input.Prepayments, item)
		return nil
	}, tenant.ID, year)
	if err != nil {
		return input, err
	}
	docs, ok := BindDocumentRepository(s.documents, tenant)
	if !ok {
		return input, fmt.Errorf("annual statement documents unavailable")
	}
	err = read(`SELECT id,data FROM documents WHERE tenant_id=$1 AND id IN (
		SELECT document_id FROM annual_statement_receipts WHERE tenant_id=$1 AND period_year=$2
	) ORDER BY id`, func(rows *sql.Rows) error {
		var id, raw string
		if err := rows.Scan(&id, &raw); err != nil {
			return err
		}
		var doc DocumentRecord
		if err := json.Unmarshal([]byte(raw), &doc); err != nil {
			return err
		}
		if doc.ID == id && annualStatementRunDocumentReadable(doc, docs) {
			input.Documents = append(input.Documents, AnnualStatementRunDocument{doc.ID, doc.Title, doc.Filename})
		}
		return nil
	}, tenant.ID, year)
	if err != nil {
		return input, err
	}
	// Only a preview can reuse vectors loaded for the page. Create passes nil
	// so every report is read within the serializable transaction below.
	if vectors != nil {
		input.Consumption = vectors
		return input, nil
	}
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
		start, end, expected, err := annualStatementConsumptionQuery(input.Period, cost.Key, ids, location)
		if err != nil {
			continue
		}
		report, err := queryAnnualStatementConsumptionReport(tx, tenant, input.Period.Year, cost.Key, expected, start, end)
		if err != nil {
			return input, err
		}
		input.Consumption[cost.Key] = report.Vector
		input.Evidence = append(input.Evidence, report.BoundaryEvidence...)
	}
	return input, nil
}

func (s *SQLAnnualStatementRunStore) getAnnualStatementRun(tenant TenantRef, id string) (AnnualStatementRun, bool, error) {
	var raw string
	err := s.db.For(tenant).QueryRow(`SELECT data FROM annual_statement_runs WHERE tenant_id=$1 AND id=$2`, tenant.ID, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return AnnualStatementRun{}, false, nil
	}
	if err != nil {
		return AnnualStatementRun{}, false, err
	}
	var run AnnualStatementRun
	if err := json.Unmarshal([]byte(raw), &run); err != nil {
		return run, false, err
	}
	return run, true, nil
}
