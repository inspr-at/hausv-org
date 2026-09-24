package store

import (
	"encoding/json"
	"fmt"
	"time"
)

// Legal settings are period-scoped and copied into every immutable run.
type AnnualStatementLegalSettings struct {
	InspectionPlace   string `json:"inspection_place"`
	InspectionPeriod  string `json:"inspection_period"`
	InspectionContact string `json:"inspection_contact"`
	Regime            string `json:"regime"`
	HeizKGApplies     bool   `json:"heizkg_applies"`
}

func DefaultAnnualStatementLegalSettings() AnnualStatementLegalSettings {
	return AnnualStatementLegalSettings{Regime: "weg"}
}
func (s AnnualStatementLegalSettings) Validate() error {
	switch s.Regime {
	case "weg", "mrg_voll", "mrg_teil", "ausnahme":
	default:
		return fmt.Errorf("invalid legal regime")
	}
	return nil
}
func (s AnnualStatementLegalSettings) Basis() string {
	var basis string
	switch s.Regime {
	case "weg":
		basis = "WEG · §§ 32, 34 WEG 2002"
	case "mrg_voll":
		basis = "MRG Vollanwendung · §§ 17, 21 MRG"
	case "mrg_teil":
		basis = "MRG Teilanwendung · Vertrag / ABGB"
	case "ausnahme":
		basis = "MRG Ausnahme · Vertrag / ABGB"
	default:
		return "Rechtsgrundlage nicht hinterlegt"
	}
	if s.HeizKGApplies {
		basis += " · HeizKG §§ 10, 12, 17, 18"
	}
	return basis
}

// ShiftStatementDate clamps to the last day of the destination month.
func ShiftStatementDate(at time.Time, months int) time.Time {
	first := time.Date(at.Year(), at.Month()+time.Month(months), 1, 0, 0, 0, 0, at.Location())
	last := first.AddDate(0, 1, -1).Day()
	return time.Date(first.Year(), first.Month(), min(at.Day(), last), 0, 0, 0, 0, at.Location())
}
func (s AnnualStatementLegalSettings) Deadline(period AnnualStatementPeriod) string {
	end, err := time.Parse("2006-01-02", period.EndsOn)
	if err != nil {
		return ""
	}
	var deadline time.Time
	if s.Regime == "mrg_voll" {
		deadline = time.Date(end.Year()+1, 6, 30, 0, 0, 0, 0, time.UTC)
	}
	if s.Regime == "weg" || s.HeizKGApplies {
		heat := ShiftStatementDate(end, 6)
		if deadline.IsZero() || heat.Before(deadline) {
			deadline = heat
		}
	}
	if deadline.IsZero() {
		return ""
	}
	return deadline.Format("2006-01-02")
}
func (r *boundAnnualStatementPeriodRepository) SaveLegal(year int, settings AnnualStatementLegalSettings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	return r.storage.saveAnnualStatementLegal(r.tenant, year, settings)
}
func (s *MemoryAnnualStatementPeriodStore) saveAnnualStatementLegal(tenant TenantRef, year int, settings AnnualStatementLegalSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	structure, found := s.structures[tenant.ID][year]
	if !found {
		return fmt.Errorf("annual statement period not found")
	}
	structure.Legal = settings
	s.structures[tenant.ID][year] = structure
	return nil
}
func (s *SQLAnnualStatementPeriodStore) saveAnnualStatementLegal(tenant TenantRef, year int, settings AnnualStatementLegalSettings) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	result, err := s.db.For(tenant).Exec(`UPDATE annual_statement_periods SET legal_settings=$1 WHERE tenant_id=$2 AND year=$3`, string(raw), tenant.ID, year)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("annual statement period not found")
	}
	return nil
}
func loadAnnualStatementLegal(q annualStatementRunQueryer, tenant TenantRef, year int) (AnnualStatementLegalSettings, error) {
	var raw string
	var legal AnnualStatementLegalSettings
	if err := q.QueryRow(`SELECT legal_settings FROM annual_statement_periods WHERE tenant_id=$1 AND year=$2`, tenant.ID, year).Scan(&raw); err != nil {
		return legal, err
	}
	if err := json.Unmarshal([]byte(raw), &legal); err != nil {
		return legal, err
	}
	return legal, legal.Validate()
}
