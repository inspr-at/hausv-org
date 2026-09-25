package store

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Informational inputs do not alter the heating cost pools. Amounts and prices
// remain exact integers; the purchase list is not a second set of receipts.
type AnnualStatementEnergyPurchase struct {
	CostTypeKey    string `json:"cost_type_key"`
	Supplier       string `json:"supplier"`
	Carrier        string `json:"carrier"`
	QuantityMicros int64  `json:"quantity_micros"`
	Unit           string `json:"unit"`
	PriceMicros    int64  `json:"price_eur_micros"`
	PriceNote      string `json:"price_note"`
}

type AnnualStatementHeatingInformation struct {
	Purchases               []AnnualStatementEnergyPurchase `json:"purchases,omitempty"`
	TaxesNote               string                          `json:"taxes_note"`
	DistrictHeatingOver20MW bool                            `json:"district_heating_over_20mw"`
	FuelMix                 string                          `json:"fuel_mix"`
	Emissions               string                          `json:"emissions"`
	MeteringCostsNote       string                          `json:"metering_costs_note"`
	OperatingCostsNote      string                          `json:"operating_costs_note"`
	RemoteMeters            string                          `json:"remote_meters"`
	MonthlyInformation      string                          `json:"monthly_information"`
	ClimateCurrentPPM       int                             `json:"climate_current_ppm"`
	ClimatePreviousPPM      int                             `json:"climate_previous_ppm"`
	ClimateSource           string                          `json:"climate_source"`
	ComplaintContact        string                          `json:"complaint_contact"`
}

func (info AnnualStatementHeatingInformation) Validate() error {
	if len(info.Purchases) > 100 {
		return fmt.Errorf("too many energy purchases")
	}
	for _, p := range info.Purchases {
		if !IsAnnualHeatingCost(p.CostTypeKey) || strings.TrimSpace(p.Supplier) == "" || strings.TrimSpace(p.Carrier) == "" || p.QuantityMicros <= 0 || p.PriceMicros < 0 || strings.TrimSpace(p.PriceNote) == "" {
			return fmt.Errorf("invalid energy purchase")
		}
		switch p.Unit {
		case "kWh", "MWh", "m³", "l", "kg", "t":
		default:
			return fmt.Errorf("invalid energy quantity unit")
		}
		if len(p.Supplier) > 300 || len(p.Carrier) > 120 || len(p.PriceNote) > 1000 {
			return fmt.Errorf("energy purchase text too long")
		}
	}
	switch info.RemoteMeters {
	case "", "yes", "no":
	default:
		return fmt.Errorf("invalid remote meter status")
	}
	for _, text := range []string{info.TaxesNote, info.FuelMix, info.Emissions, info.MeteringCostsNote, info.OperatingCostsNote, info.MonthlyInformation, info.ClimateSource, info.ComplaintContact} {
		if len(text) > 2000 {
			return fmt.Errorf("heating information text too long")
		}
	}
	if info.ClimateCurrentPPM < 0 || info.ClimateCurrentPPM > 10_000_000 || info.ClimatePreviousPPM < 0 || info.ClimatePreviousPPM > 10_000_000 {
		return fmt.Errorf("invalid climate factor")
	}
	if (info.ClimateCurrentPPM > 0 || info.ClimatePreviousPPM > 0) && (info.ClimateCurrentPPM == 0 || info.ClimatePreviousPPM == 0 || strings.TrimSpace(info.ClimateSource) == "") {
		return fmt.Errorf("climate correction requires both factors and source")
	}
	return nil
}

// This is a bounded snapshot, never a recursive copy of previous runs. The
// selected run and its measurements remain stable when a later revision exists.
type AnnualStatementPreviousHeating struct {
	RunID       string                                      `json:"run_id"`
	Revision    int                                         `json:"revision"`
	Period      AnnualStatementPeriod                       `json:"period"`
	Consumption map[string]AnnualStatementConsumptionVector `json:"consumption"`
}

func previousHeatingSnapshot(period AnnualStatementPeriod, runs []AnnualStatementRun) *AnnualStatementPreviousHeating {
	start, e1 := time.Parse("2006-01-02", period.StartsOn)
	end, e2 := time.Parse("2006-01-02", period.EndsOn)
	if e1 != nil || e2 != nil {
		return nil
	}
	var selected *AnnualStatementRun
	for i := range runs {
		run := &runs[i]
		if run.Input.Period.Year != period.Year-1 {
			continue
		}
		if run.Input.Period.StartsOn != ShiftStatementDate(start, -12).Format("2006-01-02") || run.Input.Period.EndsOn != ShiftStatementDate(end, -12).Format("2006-01-02") {
			continue
		}
		if selected == nil || run.Revision > selected.Revision {
			selected = run
		}
	}
	if selected == nil {
		return nil
	}
	snapshot := AnnualStatementPreviousHeating{RunID: selected.ID, Revision: selected.Revision, Period: selected.Input.Period, Consumption: selected.Input.Consumption}
	raw, _ := json.Marshal(snapshot)
	var copy AnnualStatementPreviousHeating
	_ = json.Unmarshal(raw, &copy)
	return &copy
}
