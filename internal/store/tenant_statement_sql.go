package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type TenantStatementRepository struct {
	db     *TenantDB
	tenant TenantRef
}

func BindTenantStatementRepository(database *TenantDB, tenant TenantRef) (*TenantStatementRepository, bool) {
	ref, ok := validTenantRef(tenant)
	if !ok || database == nil {
		return nil, false
	}
	return &TenantStatementRepository{database, ref}, true
}

func (r *TenantStatementRepository) Management(unit string) (RentalManagement, error) {
	var raw string
	err := r.db.For(r.tenant).QueryRow(`SELECT data FROM rental_management WHERE tenant_id=$1 AND unit_id=$2`, r.tenant.ID, unit).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return RentalManagement{UnitID: unit, Passable: map[string]bool{}, ContractCosts: map[string][]string{}}, nil
	}
	var m RentalManagement
	if err == nil {
		err = json.Unmarshal([]byte(raw), &m)
	}
	return m, err
}

func (r *TenantStatementRepository) SaveManagement(m RentalManagement) error {
	if NormalizeUnitID(m.UnitID) != m.UnitID || m.UnitID == "" || m.OwnerEmail == "" {
		return fmt.Errorf("Einheit und Eigentümer fehlen.")
	}
	// A mandate can only name a currently registered owner in this tenant.
	units, _ := BindUnitRepository(NewSQLUnitStore(r.db), r.tenant)
	listed, err := units.ListChecked()
	if err != nil {
		return err
	}
	found := false
	for _, unit := range listed {
		if unit.ID == m.UnitID && EmailListContains(unit.OwnerEmails, m.OwnerEmail) {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("Eigentümer gehört nicht zur Einheit.")
	}
	m.UpdatedAt = time.Now().UTC()
	raw, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = r.db.For(r.tenant).Exec(`INSERT INTO rental_management(tenant_id,tenant_slug,unit_id,data) VALUES($1,$2,$3,$4) ON CONFLICT(tenant_id,unit_id) DO UPDATE SET data=excluded.data,tenant_id=coalesce(rental_management.tenant_id,excluded.tenant_id)`, r.tenant.ID, r.tenant.Slug, m.UnitID, string(raw))
	return err
}

func (r *TenantStatementRepository) Create(run AnnualStatementRun, unit, statementOn, contractDueOn, actor string) (TenantStatement, error) {
	tx, err := (&SQLAnnualStatementRunStore{db: r.db}).begin(r.tenant, false)
	if err != nil {
		return TenantStatement{}, err
	}
	defer tx.Rollback()
	var runRaw, approvalRaw, managementRaw string
	if err = tx.QueryRow(`SELECT r.data,a.data FROM annual_statement_runs r JOIN annual_statement_run_approvals a ON a.tenant_id=r.tenant_id AND a.run_id=r.id WHERE r.tenant_id=$1 AND r.id=$2`, r.tenant.ID, run.ID).Scan(&runRaw, &approvalRaw); err != nil {
		return TenantStatement{}, fmt.Errorf("Freigegebener WEG-Lauf nicht verfügbar.")
	}
	run = AnnualStatementRun{}
	if err = json.Unmarshal([]byte(runRaw), &run); err != nil {
		return TenantStatement{}, err
	}
	if err = json.Unmarshal([]byte(approvalRaw), &run.Approval); err != nil {
		return TenantStatement{}, err
	}
	if err = tx.QueryRow(`SELECT data FROM rental_management WHERE tenant_id=$1 AND unit_id=$2`, r.tenant.ID, unit).Scan(&managementRaw); err != nil {
		return TenantStatement{}, fmt.Errorf("Mietverwaltung zuerst einrichten.")
	}
	var m RentalManagement
	if err = json.Unmarshal([]byte(managementRaw), &m); err != nil {
		return TenantStatement{}, err
	}
	history, err := loadLeasesTx(tx, r.tenant, unit)
	if err != nil {
		return TenantStatement{}, err
	}
	s, err := DeriveTenantStatement(run, m, history, statementOn, contractDueOn)
	if err != nil {
		return s, err
	}
	id, err := newLeaseID()
	if err != nil {
		return s, err
	}
	s.ID = "mieter-" + id
	s.CreatedBy = actor
	s.CreatedAt = time.Now().UTC()
	raw, err := json.Marshal(s)
	if err != nil {
		return s, err
	}
	_, err = tx.Exec(`INSERT INTO tenant_statements(tenant_id,tenant_slug,id,run_id,unit_id,data) VALUES($1,$2,$3,$4,$5,$6)`, r.tenant.ID, r.tenant.Slug, s.ID, run.ID, unit, string(raw))
	if err == nil {
		err = tx.Commit()
	}
	return s, err
}

func (r *TenantStatementRepository) Get(id string) (TenantStatement, bool, error) {
	var raw, approval string
	err := r.db.For(r.tenant).QueryRow(`SELECT s.data,COALESCE(a.data,'null') FROM tenant_statements s LEFT JOIN tenant_statement_approvals a ON a.tenant_id=s.tenant_id AND a.statement_id=s.id WHERE s.tenant_id=$1 AND s.id=$2`, r.tenant.ID, id).Scan(&raw, &approval)
	if errors.Is(err, sql.ErrNoRows) {
		return TenantStatement{}, false, nil
	}
	if err != nil {
		return TenantStatement{}, false, err
	}
	var s TenantStatement
	if err = json.Unmarshal([]byte(raw), &s); err == nil {
		err = json.Unmarshal([]byte(approval), &s.Approval)
	}
	return s, err == nil, err
}

func (r *TenantStatementRepository) List(runID string) ([]TenantStatement, error) {
	rows, err := r.db.For(r.tenant).Query(`SELECT s.data,COALESCE(a.data,'null') FROM tenant_statements s LEFT JOIN tenant_statement_approvals a ON a.tenant_id=s.tenant_id AND a.statement_id=s.id WHERE s.tenant_id=$1 AND s.run_id=$2 ORDER BY s.id DESC`, r.tenant.ID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TenantStatement{}
	for rows.Next() {
		var raw, approval string
		if err = rows.Scan(&raw, &approval); err != nil {
			return nil, err
		}
		var s TenantStatement
		if err = json.Unmarshal([]byte(raw), &s); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(approval), &s.Approval); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *TenantStatementRepository) Approve(id, actor, role string) (TenantStatement, error) {
	s, found, err := r.Get(id)
	if err != nil {
		return s, err
	}
	if !found {
		return s, fmt.Errorf("Mieterabrechnung fehlt.")
	}
	if s.Approval != nil {
		return s, nil
	}
	if role != RoleAdmin && role != RoleManager {
		return s, fmt.Errorf("Freigabe nur durch Verwaltung.")
	}
	approval := AnnualStatementRunApproval{ApprovedAt: time.Now().UTC(), ApprovedBy: actor, Role: role}
	raw, err := json.Marshal(approval)
	if err != nil {
		return s, err
	}
	_, err = r.db.For(r.tenant).Exec(`INSERT INTO tenant_statement_approvals(tenant_id,tenant_slug,statement_id,data) VALUES($1,$2,$3,$4) ON CONFLICT(tenant_id,statement_id) DO NOTHING`, r.tenant.ID, r.tenant.Slug, id, string(raw))
	if err != nil {
		return s, err
	}
	s, _, err = r.Get(id)
	return s, err
}
