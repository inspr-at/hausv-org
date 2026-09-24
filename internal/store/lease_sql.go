package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

type SQLLeaseStore struct{ db *TenantDB }

func NewSQLLeaseStore(db *TenantDB) *SQLLeaseStore { return &SQLLeaseStore{db: db} }

func (*SQLLeaseStore) leaseStorage() {}

func (s *SQLLeaseStore) createLease(tenant TenantRef, lease Lease) (Lease, error) {
	normalized, err := normalizeLease(lease)
	if err != nil {
		return Lease{}, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return Lease{}, err
	}
	defer tx.Rollback()
	if err := ensureLeaseUnit(tx, tenant, normalized.UnitID); err != nil {
		return Lease{}, err
	}
	if err := ensureNoHauptmieteOverlap(tx, tenant, normalized, ""); err != nil {
		return Lease{}, err
	}
	if err := insertLeaseGraph(tx, tenant, normalized); err != nil {
		return Lease{}, err
	}
	if err := tx.Commit(); err != nil {
		return Lease{}, err
	}
	return normalized, nil
}

func (s *SQLLeaseStore) updateLease(tenant TenantRef, lease Lease) (Lease, error) {
	normalized, err := normalizeLease(lease)
	if err != nil {
		return Lease{}, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return Lease{}, err
	}
	defer tx.Rollback()
	current, found, err := loadLeaseTx(tx, tenant, normalized.ID)
	if err != nil {
		return Lease{}, err
	}
	if !found {
		return Lease{}, ErrLeaseNotFound
	}
	if err := ensureLeaseUnit(tx, tenant, normalized.UnitID); err != nil {
		return Lease{}, err
	}
	if err := ensureNoHauptmieteOverlap(tx, tenant, normalized, normalized.ID); err != nil {
		return Lease{}, err
	}
	if _, err := tx.Exec(`UPDATE leases SET unit_id=$1,status=$2,concluded_on=$3,starts_on=$4,ends_on=$5,lease_kind=$6,use_kind=$7,mrg_scope=$8,rent_regime=$9,price_restricted=$10,max_hmz_cents=$11,landlord_is_business=$12,tenant_is_consumer=$13,zinstermin_day=$14,vat_opted=$15,notes=$16,updated_at=$17,updated_by=$18 WHERE tenant_id=$19 AND id=$20`,
		normalized.UnitID, normalized.Status, normalized.ConcludedOn, normalized.StartsOn, nullString(normalized.EndsOn), normalized.LeaseKind, normalized.UseKind, normalized.MRGScope, normalized.RentRegime, normalized.PriceRestricted, nullInt64(normalized.MaxHMZCents), normalized.LandlordIsBusiness, normalized.TenantIsConsumer, normalized.ZinsterminDay, normalized.VATOpted, normalized.Notes, normalized.UpdatedAt.UTC().Format(time.RFC3339Nano), normalized.UpdatedBy, tenant.ID, normalized.ID); err != nil {
		return Lease{}, err
	}
	if err := tx.Commit(); err != nil {
		return Lease{}, err
	}
	normalized.Parties, normalized.Components, normalized.Clauses = current.Parties, current.Components, current.Clauses
	return normalized, nil
}

func (s *SQLLeaseStore) endLease(tenant TenantRef, id, endsOn, actor string) (Lease, error) {
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return Lease{}, err
	}
	defer tx.Rollback()
	current, found, err := loadLeaseTx(tx, tenant, strings.TrimSpace(id))
	if err != nil || !found {
		if err == nil {
			err = ErrLeaseNotFound
		}
		return Lease{}, err
	}
	current.Status = LeaseStatusEnded
	current.EndsOn = strings.TrimSpace(endsOn)
	current.UpdatedBy = actor
	current.UpdatedAt = time.Now().UTC()
	current.Parties, current.Components, current.Clauses = nil, nil, nil
	normalized, err := normalizeLease(current)
	if err != nil {
		return Lease{}, err
	}
	if err := ensureNoHauptmieteOverlap(tx, tenant, normalized, normalized.ID); err != nil {
		return Lease{}, err
	}
	if _, err := tx.Exec(`UPDATE leases SET status=$1,ends_on=$2,updated_at=$3,updated_by=$4 WHERE tenant_id=$5 AND id=$6`,
		normalized.Status, normalized.EndsOn, normalized.UpdatedAt.Format(time.RFC3339Nano), normalized.UpdatedBy, tenant.ID, normalized.ID); err != nil {
		return Lease{}, err
	}
	if err := tx.Commit(); err != nil {
		return Lease{}, err
	}
	lease, _, err := s.getLease(tenant, normalized.ID)
	return lease, err
}

func (s *SQLLeaseStore) getLease(tenant TenantRef, id string) (Lease, bool, error) {
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return Lease{}, false, err
	}
	defer tx.Rollback()
	return loadLeaseTx(tx, tenant, strings.TrimSpace(id))
}

func (s *SQLLeaseStore) listLeases(tenant TenantRef) ([]Lease, error) {
	return s.leaseHistory(tenant, "")
}

func (s *SQLLeaseStore) leaseHistory(tenant TenantRef, unitID string) ([]Lease, error) {
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return loadLeasesTx(tx, tenant, textutil.UnitID(unitID))
}

func (s *SQLLeaseStore) replaceLeaseParties(tenant TenantRef, leaseID string, parties []LeaseParty) error {
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, found, err := loadLeaseTx(tx, tenant, leaseID); err != nil || !found {
		if err == nil {
			err = ErrLeaseNotFound
		}
		return err
	}
	normalized := make([]LeaseParty, 0, len(parties))
	for _, party := range parties {
		item, err := normalizeParty(leaseID, party)
		if err != nil {
			return err
		}
		normalized = append(normalized, item)
	}
	if _, err := tx.Exec(`DELETE FROM lease_parties WHERE tenant_id=$1 AND lease_id=$2`, tenant.ID, leaseID); err != nil {
		return err
	}
	for _, party := range normalized {
		if err := insertParty(tx, tenant, party); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLLeaseStore) addRentComponent(tenant TenantRef, leaseID string, component RentComponent) (RentComponent, error) {
	component, err := normalizeComponent(leaseID, component)
	if err != nil {
		return RentComponent{}, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return RentComponent{}, err
	}
	defer tx.Rollback()
	if _, found, err := loadLeaseTx(tx, tenant, leaseID); err != nil || !found {
		if err == nil {
			err = ErrLeaseNotFound
		}
		return RentComponent{}, err
	}
	if err := insertComponent(tx, tenant, component); err != nil {
		return RentComponent{}, err
	}
	if err := tx.Commit(); err != nil {
		return RentComponent{}, err
	}
	return component, nil
}

func (s *SQLLeaseStore) setLeaseClause(tenant TenantRef, clause IndexClause) (IndexClause, error) {
	clause, err := normalizeClause(clause.LeaseID, clause)
	if err != nil {
		return IndexClause{}, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return IndexClause{}, err
	}
	defer tx.Rollback()
	if _, found, err := loadLeaseTx(tx, tenant, clause.LeaseID); err != nil || !found {
		if err == nil {
			err = ErrLeaseNotFound
		}
		return IndexClause{}, err
	}
	if err := upsertClause(tx, tenant, clause); err != nil {
		return IndexClause{}, err
	}
	if err := tx.Commit(); err != nil {
		return IndexClause{}, err
	}
	return clause, nil
}

func (s *SQLLeaseStore) reviewLeaseClause(tenant TenantRef, id, status, note string) (IndexClause, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case ReviewUnreviewed, ReviewOK, ReviewDoubtful, ReviewInvalid:
	default:
		return IndexClause{}, fmt.Errorf("%w: review", ErrLeaseInvalid)
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return IndexClause{}, err
	}
	defer tx.Rollback()
	clauses, err := loadClauses(tx, tenant)
	if err != nil {
		return IndexClause{}, err
	}
	var current IndexClause
	found := false
	for _, clause := range clauses[strings.TrimSpace(id)] {
		if clause.ID == strings.TrimSpace(id) {
			current, found = clause, true
		}
	}
	if !found {
		for _, list := range clauses {
			for _, clause := range list {
				if clause.ID == strings.TrimSpace(id) {
					current, found = clause, true
				}
			}
		}
	}
	if !found {
		return IndexClause{}, ErrLeaseNotFound
	}
	current.ReviewStatus = status
	current.ReviewNote = strings.TrimSpace(note)
	if _, err := tx.Exec(`UPDATE index_clauses SET review_status=$1,review_note=$2 WHERE tenant_id=$3 AND id=$4`, current.ReviewStatus, current.ReviewNote, tenant.ID, current.ID); err != nil {
		return IndexClause{}, err
	}
	if err := tx.Commit(); err != nil {
		return IndexClause{}, err
	}
	return current, nil
}

func (s *SQLLeaseStore) setValorisation(tenant TenantRef, state ValorisationState) error {
	if err := normalizeValorisation(&state); err != nil {
		return err
	}
	if strings.TrimSpace(state.ClauseID) == "" {
		return fmt.Errorf("%w: clause", ErrLeaseInvalid)
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := upsertValorisation(tx, tenant, state); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLLeaseStore) importLeases(tenant TenantRef, drafts []LeaseImportDraft, units []Unit, commit bool, actor string) (LeaseImportReport, error) {
	report, leases := prepareLeaseImport(drafts, units)
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return report, err
	}
	defer tx.Rollback()
	existing, err := loadLeasesTx(tx, tenant, "")
	if err != nil {
		return report, err
	}
	clean := make([]Lease, 0, len(leases))
	next := 0
	for i := range report.Rows {
		if len(report.Rows[i].Errors) > 0 {
			continue
		}
		lease := leases[next]
		next++
		lease.UpdatedBy = actor
		if lease.UpdatedAt.IsZero() {
			lease.UpdatedAt = time.Now().UTC()
		}
		blocked := false
		if hauptmieteOccupies(lease) {
			for _, other := range existing {
				if other.UnitID == lease.UnitID && hauptmieteOccupies(other) && rangesOverlap(other.StartsOn, other.EndsOn, lease.StartsOn, lease.EndsOn) {
					report.Rows[i].Errors = append(report.Rows[i].Errors, "overlap")
					blocked = true
					break
				}
			}
		}
		if !blocked {
			clean = append(clean, lease)
			existing = append(existing, lease)
		}
	}
	if !commit || report.HasErrors() {
		return report, nil
	}
	for _, lease := range clean {
		if err := insertLeaseGraph(tx, tenant, lease); err != nil {
			return report, err
		}
	}
	if err := tx.Commit(); err != nil {
		return report, err
	}
	report.Committed = len(clean)
	return report, nil
}

func ensureLeaseUnit(tx *sql.Tx, tenant TenantRef, unitID string) error {
	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM units WHERE tenant_id=$1 AND id=$2`, tenant.ID, unitID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return ErrUnitNotFound
	}
	return nil
}

func ensureNoHauptmieteOverlap(tx *sql.Tx, tenant TenantRef, lease Lease, ignoreID string) error {
	if !hauptmieteOccupies(lease) {
		return nil
	}
	rows, err := tx.Query(`SELECT id,starts_on,COALESCE(ends_on,'') FROM leases WHERE tenant_id=$1 AND unit_id=$2 AND lease_kind=$3 AND id<>$4`, tenant.ID, lease.UnitID, LeaseKindHauptmiete, ignoreID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, start, end string
		if err := rows.Scan(&id, &start, &end); err != nil {
			return err
		}
		if rangesOverlap(lease.StartsOn, lease.EndsOn, start, end) {
			return ErrLeaseOverlap
		}
	}
	return rows.Err()
}

func insertLeaseGraph(tx *sql.Tx, tenant TenantRef, lease Lease) error {
	if _, err := tx.Exec(`INSERT INTO leases(id,tenant_slug,tenant_id,unit_id,status,concluded_on,starts_on,ends_on,lease_kind,use_kind,mrg_scope,rent_regime,price_restricted,max_hmz_cents,landlord_is_business,tenant_is_consumer,zinstermin_day,vat_opted,notes,updated_at,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`,
		lease.ID, tenant.Slug, tenant.ID, lease.UnitID, lease.Status, lease.ConcludedOn, lease.StartsOn, nullString(lease.EndsOn), lease.LeaseKind, lease.UseKind, lease.MRGScope, lease.RentRegime, lease.PriceRestricted, nullInt64(lease.MaxHMZCents), lease.LandlordIsBusiness, lease.TenantIsConsumer, lease.ZinsterminDay, lease.VATOpted, lease.Notes, stamp(lease.UpdatedAt), lease.UpdatedBy); err != nil {
		return err
	}
	for _, party := range lease.Parties {
		if err := insertParty(tx, tenant, party); err != nil {
			return err
		}
	}
	for _, component := range lease.Components {
		if err := insertComponent(tx, tenant, component); err != nil {
			return err
		}
	}
	for _, clause := range lease.Clauses {
		if err := upsertClause(tx, tenant, clause); err != nil {
			return err
		}
	}
	return nil
}

func stamp(at time.Time) string {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return at.UTC().Format(time.RFC3339Nano)
}

func nullString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func insertParty(tx *sql.Tx, tenant TenantRef, party LeaseParty) error {
	_, err := tx.Exec(`INSERT INTO lease_parties(id,lease_id,tenant_slug,tenant_id,name,address,email,role,valid_from,valid_to) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		party.ID, party.LeaseID, tenant.Slug, tenant.ID, party.Name, party.Address, party.Email, party.Role, party.ValidFrom, nullString(party.ValidTo))
	return err
}

func insertComponent(tx *sql.Tx, tenant TenantRef, component RentComponent) error {
	_, err := tx.Exec(`INSERT INTO rent_components(id,lease_id,tenant_slug,tenant_id,kind,net_cents,vat_rate_bp,valid_from,origin,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		component.ID, component.LeaseID, tenant.Slug, tenant.ID, component.Kind, component.NetCents, component.VATRateBP, component.ValidFrom, component.Origin, stamp(component.CreatedAt))
	return err
}

func upsertClause(tx *sql.Tx, tenant TenantRef, clause IndexClause) error {
	steps := "[]"
	if len(clause.StaffelSteps) > 0 {
		raw, err := json.Marshal(clause.StaffelSteps)
		if err != nil {
			return err
		}
		steps = string(raw)
	}
	result, err := tx.Exec(`UPDATE index_clauses SET lease_id=$1,component_kind=$2,clause_type=$3,series=$4,base_period=$5,base_value=$6,threshold_kind=$7,threshold_value=$8,threshold_inclusive=$9,full_change_on_trigger=$10,two_way=$11,pct_rounding=$12,periodic_month=$13,reference_month_offset=$14,clause_text=$15,review_status=$16,review_note=$17,valid_from=$18,staffel_steps=$21 WHERE tenant_id=$19 AND id=$20`,
		clause.LeaseID, clause.ComponentKind, clause.ClauseType, clause.Series, clause.BasePeriod, clause.BaseValue, nullString(clause.ThresholdKind), nullString(clause.ThresholdValue), clause.ThresholdInclusive, clause.FullChangeOnTrigger, clause.TwoWay, clause.PctRounding, nullInt(clause.PeriodicMonth), nullOffset(clause.ReferenceMonthOffset), clause.ClauseText, clause.ReviewStatus, clause.ReviewNote, clause.ValidFrom, tenant.ID, clause.ID, steps)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		if _, err := tx.Exec(`INSERT INTO index_clauses(id,lease_id,tenant_slug,tenant_id,component_kind,clause_type,series,base_period,base_value,threshold_kind,threshold_value,threshold_inclusive,full_change_on_trigger,two_way,pct_rounding,periodic_month,reference_month_offset,clause_text,review_status,review_note,valid_from,staffel_steps) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`,
			clause.ID, clause.LeaseID, tenant.Slug, tenant.ID, clause.ComponentKind, clause.ClauseType, clause.Series, clause.BasePeriod, clause.BaseValue, nullString(clause.ThresholdKind), nullString(clause.ThresholdValue), clause.ThresholdInclusive, clause.FullChangeOnTrigger, clause.TwoWay, clause.PctRounding, nullInt(clause.PeriodicMonth), nullOffset(clause.ReferenceMonthOffset), clause.ClauseText, clause.ReviewStatus, clause.ReviewNote, clause.ValidFrom, steps); err != nil {
			return err
		}
	}
	if clause.State != nil {
		clause.State.ClauseID = clause.ID
		return upsertValorisation(tx, tenant, *clause.State)
	}
	return nil
}

func nullOffset(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func upsertValorisation(tx *sql.Tx, tenant TenantRef, state ValorisationState) error {
	result, err := tx.Exec(`UPDATE valorisation_state SET contract_value=$1,contract_base_period=$2,contract_base_value=$3,cap_value=$4,cap_anchor_period=$5,cap_last_year=$6,last_effective_on=$7,last_run_item_id=$8 WHERE tenant_id=$9 AND clause_id=$10`,
		state.ContractValue, state.ContractBasePeriod, state.ContractBaseValue, state.CapValue, state.CapAnchorPeriod, nullInt(state.CapLastYear), nullString(state.LastEffectiveOn), nullString(state.LastRunItemID), tenant.ID, state.ClauseID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}
	_, err = tx.Exec(`INSERT INTO valorisation_state(clause_id,tenant_slug,tenant_id,contract_value,contract_base_period,contract_base_value,cap_value,cap_anchor_period,cap_last_year,last_effective_on,last_run_item_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		state.ClauseID, tenant.Slug, tenant.ID, state.ContractValue, state.ContractBasePeriod, state.ContractBaseValue, state.CapValue, state.CapAnchorPeriod, nullInt(state.CapLastYear), nullString(state.LastEffectiveOn), nullString(state.LastRunItemID))
	return err
}

func loadLeaseTx(tx *sql.Tx, tenant TenantRef, id string) (Lease, bool, error) {
	leases, err := loadLeasesTx(tx, tenant, "")
	if err != nil {
		return Lease{}, false, err
	}
	for _, lease := range leases {
		if lease.ID == id {
			return lease, true, nil
		}
	}
	return Lease{}, false, nil
}

func loadLeasesTx(tx *sql.Tx, tenant TenantRef, unitID string) ([]Lease, error) {
	query := `SELECT id,unit_id,status,concluded_on,starts_on,COALESCE(ends_on,''),lease_kind,use_kind,mrg_scope,rent_regime,price_restricted,max_hmz_cents,landlord_is_business,tenant_is_consumer,zinstermin_day,vat_opted,notes,updated_at,updated_by FROM leases WHERE tenant_id=$1`
	args := []any{tenant.ID}
	if unitID != "" {
		query += ` AND unit_id=$2`
		args = append(args, unitID)
	}
	query += ` ORDER BY unit_id,starts_on,id`
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	leases := []Lease{}
	for rows.Next() {
		var lease Lease
		var maxHMZ sql.NullInt64
		var updated string
		if err := rows.Scan(&lease.ID, &lease.UnitID, &lease.Status, &lease.ConcludedOn, &lease.StartsOn, &lease.EndsOn, &lease.LeaseKind, &lease.UseKind, &lease.MRGScope, &lease.RentRegime, &lease.PriceRestricted, &maxHMZ, &lease.LandlordIsBusiness, &lease.TenantIsConsumer, &lease.ZinsterminDay, &lease.VATOpted, &lease.Notes, &updated, &lease.UpdatedBy); err != nil {
			return nil, err
		}
		if maxHMZ.Valid {
			lease.MaxHMZCents = &maxHMZ.Int64
		}
		lease.PriceRestrictedSet = true
		lease.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		leases = append(leases, lease)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	parties, err := loadParties(tx, tenant)
	if err != nil {
		return nil, err
	}
	components, err := loadComponents(tx, tenant)
	if err != nil {
		return nil, err
	}
	clauses, err := loadClauses(tx, tenant)
	if err != nil {
		return nil, err
	}
	for i := range leases {
		leases[i].Parties = parties[leases[i].ID]
		leases[i].Components = components[leases[i].ID]
		leases[i].Clauses = clauses[leases[i].ID]
	}
	return leases, nil
}

func loadParties(tx *sql.Tx, tenant TenantRef) (map[string][]LeaseParty, error) {
	rows, err := tx.Query(`SELECT id,lease_id,name,address,email,role,valid_from,COALESCE(valid_to,'') FROM lease_parties WHERE tenant_id=$1 ORDER BY valid_from,id`, tenant.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]LeaseParty{}
	for rows.Next() {
		var party LeaseParty
		if err := rows.Scan(&party.ID, &party.LeaseID, &party.Name, &party.Address, &party.Email, &party.Role, &party.ValidFrom, &party.ValidTo); err != nil {
			return nil, err
		}
		out[party.LeaseID] = append(out[party.LeaseID], party)
	}
	return out, rows.Err()
}

func loadComponents(tx *sql.Tx, tenant TenantRef) (map[string][]RentComponent, error) {
	rows, err := tx.Query(`SELECT id,lease_id,kind,net_cents,vat_rate_bp,valid_from,origin,created_at FROM rent_components WHERE tenant_id=$1 ORDER BY valid_from,id`, tenant.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]RentComponent{}
	for rows.Next() {
		var component RentComponent
		var created string
		if err := rows.Scan(&component.ID, &component.LeaseID, &component.Kind, &component.NetCents, &component.VATRateBP, &component.ValidFrom, &component.Origin, &created); err != nil {
			return nil, err
		}
		component.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out[component.LeaseID] = append(out[component.LeaseID], component)
	}
	return out, rows.Err()
}

func loadClauses(tx *sql.Tx, tenant TenantRef) (map[string][]IndexClause, error) {
	rows, err := tx.Query(`SELECT id,lease_id,component_kind,clause_type,series,base_period,base_value,COALESCE(threshold_kind,''),COALESCE(threshold_value,''),threshold_inclusive,full_change_on_trigger,two_way,pct_rounding,periodic_month,reference_month_offset,clause_text,review_status,review_note,valid_from,staffel_steps FROM index_clauses WHERE tenant_id=$1 ORDER BY valid_from,id`, tenant.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]IndexClause{}
	for rows.Next() {
		var clause IndexClause
		var steps string
		var month, offset sql.NullInt64
		if err := rows.Scan(&clause.ID, &clause.LeaseID, &clause.ComponentKind, &clause.ClauseType, &clause.Series, &clause.BasePeriod, &clause.BaseValue, &clause.ThresholdKind, &clause.ThresholdValue, &clause.ThresholdInclusive, &clause.FullChangeOnTrigger, &clause.TwoWay, &clause.PctRounding, &month, &offset, &clause.ClauseText, &clause.ReviewStatus, &clause.ReviewNote, &clause.ValidFrom, &steps); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(steps), &clause.StaffelSteps); err != nil {
			return nil, fmt.Errorf("Staffeldaten: %w", err)
		}
		if month.Valid {
			clause.PeriodicMonth = int(month.Int64)
		}
		if offset.Valid {
			value := int(offset.Int64)
			clause.ReferenceMonthOffset = &value
		}
		out[clause.LeaseID] = append(out[clause.LeaseID], clause)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	states, err := loadStates(tx, tenant)
	if err != nil {
		return nil, err
	}
	for leaseID, list := range out {
		for i := range list {
			if state, ok := states[list[i].ID]; ok {
				copied := state
				list[i].State = &copied
			}
		}
		out[leaseID] = list
	}
	return out, nil
}

func loadStates(tx *sql.Tx, tenant TenantRef) (map[string]ValorisationState, error) {
	rows, err := tx.Query(`SELECT clause_id,contract_value,contract_base_period,contract_base_value,cap_value,cap_anchor_period,cap_last_year,COALESCE(last_effective_on,''),COALESCE(last_run_item_id,'') FROM valorisation_state WHERE tenant_id=$1`, tenant.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]ValorisationState{}
	for rows.Next() {
		var state ValorisationState
		var year sql.NullInt64
		if err := rows.Scan(&state.ClauseID, &state.ContractValue, &state.ContractBasePeriod, &state.ContractBaseValue, &state.CapValue, &state.CapAnchorPeriod, &year, &state.LastEffectiveOn, &state.LastRunItemID); err != nil {
			return nil, err
		}
		if year.Valid {
			state.CapLastYear = int(year.Int64)
		}
		out[state.ClauseID] = state
	}
	return out, rows.Err()
}

// ReplaceLeaseGraph writes one lease and its children, replacing any previous
// row with the same id. Demo seeding uses it inside an already-open transaction.
func ReplaceLeaseGraph(tx *sql.Tx, tenant TenantRef, lease Lease) error {
	normalized, err := normalizeLease(lease)
	if err != nil {
		return err
	}
	if err := ensureLeaseUnit(tx, tenant, normalized.UnitID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM valorisation_state WHERE tenant_id=$1 AND clause_id IN (SELECT id FROM index_clauses WHERE lease_id=$2)`, tenant.ID, normalized.ID); err != nil {
		return err
	}
	for _, query := range []string{
		`DELETE FROM index_clauses WHERE tenant_id=$1 AND lease_id=$2`,
		`DELETE FROM rent_components WHERE tenant_id=$1 AND lease_id=$2`,
		`DELETE FROM lease_parties WHERE tenant_id=$1 AND lease_id=$2`,
		`DELETE FROM leases WHERE tenant_id=$1 AND id=$2`,
	} {
		if _, err := tx.Exec(query, tenant.ID, normalized.ID); err != nil {
			return err
		}
	}
	if err := ensureNoHauptmieteOverlap(tx, tenant, normalized, ""); err != nil {
		return err
	}
	return insertLeaseGraph(tx, tenant, normalized)
}
