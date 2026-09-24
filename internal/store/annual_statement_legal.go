package store

import (
	"encoding/json"
	"fmt"
	"time"
)

// Legal settings are period-scoped and copied into every immutable run.
type AnnualStatementLegalSettings struct {
	AgreedShares              map[string]map[string]int          `json:"agreed_shares_ppm,omitempty"`
	HeatingInformation        *AnnualStatementHeatingInformation `json:"heating_information,omitempty"`
	MonthlyProposals          map[string]map[string]int64        `json:"monthly_proposals_cents,omitempty"`
	NextPrepaymentOn          string                             `json:"next_prepayment_on,omitempty"`
	HeatingPrepayments        map[string]map[string]int64        `json:"heating_prepayments_cents,omitempty"`
	HeatingConsumptionPercent int                                `json:"heating_consumption_percent"`
	HeatableAreas             map[string]int                     `json:"heatable_areas_m2_hundredths,omitempty"`
	InspectionPlace           string                             `json:"inspection_place"`
	InspectionPeriod          string                             `json:"inspection_period"`
	InspectionContact         string                             `json:"inspection_contact"`
	Regime                    string                             `json:"regime"`
	HeizKGApplies             bool                               `json:"heizkg_applies"`
	// ShowVAT prints net, rate and VAT. Off leaves the gross statement unchanged.
	ShowVAT bool `json:"show_vat,omitempty"`
}

func DefaultAnnualStatementLegalSettings() AnnualStatementLegalSettings {
	return AnnualStatementLegalSettings{Regime: "weg", HeatingConsumptionPercent: 70}
}
func (s AnnualStatementLegalSettings) Validate() error {
	for cost, shares := range s.AgreedShares {
		if !annualStatementCostTypeKeyPattern.MatchString(cost) || !validAgreedShareTotal(shares) {
			return fmt.Errorf("agreed shares must total 1000000 ppm")
		}
	}
	if s.HeatingInformation != nil {
		if err := s.HeatingInformation.Validate(); err != nil {
			return err
		}
	}
	switch s.Regime {
	case "weg", "mrg_voll", "mrg_teil", "ausnahme":
	default:
		return fmt.Errorf("invalid legal regime")
	}
	if s.HeizKGApplies && (s.HeatingConsumptionPercent < 55 || s.HeatingConsumptionPercent > 85) {
		return fmt.Errorf("heating consumption share outside 55–85 percent")
	}
	for id, area := range s.HeatableAreas {
		if NormalizeUnitID(id) != id || area < 0 || area > 9999999 {
			return fmt.Errorf("invalid heatable area")
		}
	}
	if s.NextPrepaymentOn != "" {
		if _, err := time.Parse("2006-01-02", s.NextPrepaymentOn); err != nil {
			return fmt.Errorf("invalid prepayment start")
		}
	}
	for _, components := range s.MonthlyProposals {
		for _, cents := range components {
			if cents < 0 || cents > 9223372036854775807/12 {
				return fmt.Errorf("invalid monthly proposal")
			}
		}
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
	structure.Legal = cloneAnnualStatementLegal(settings)
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
	if legal.HeatingConsumptionPercent == 0 {
		legal.HeatingConsumptionPercent = 70
	}
	return legal, legal.Validate()
}

func cloneAnnualStatementLegal(legal AnnualStatementLegalSettings) AnnualStatementLegalSettings {
	raw, _ := json.Marshal(legal)
	var copy AnnualStatementLegalSettings
	_ = json.Unmarshal(raw, &copy)
	return copy
}
