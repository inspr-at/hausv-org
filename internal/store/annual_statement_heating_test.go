package store

import (
	"reflect"
	"testing"
)

func heatingFixture() AnnualStatementRunInput {
	in := annualRunFixture()
	in.Structure.Legal = AnnualStatementLegalSettings{Regime: "weg", HeizKGApplies: true, HeatingConsumptionPercent: 70, HeatableAreas: map[string]int{"a": 5000, "b": 5000}}
	in.Structure.Legal.HeatingPrepayments = map[string]map[string]int64{"a": {"heizung": 1000}, "b": {"heizung": 0}}
	in.Structure.CostTypes = []AnnualStatementCostType{{Key: "heizung", Name: "Heizung", Allocatable: true, AllocationKey: AllocationKeyVerbrauch}}
	in.Receipts = in.Receipts[:2]
	for i := range in.Receipts {
		in.Receipts[i].CostTypeKey = "heizung"
	}
	in.Receipts[0].HeatingCategory = "energie"
	in.Receipts[0].AmountCents = 100000
	in.Receipts[1].HeatingCategory = "sonstige_betriebskosten"
	in.Receipts[1].AmountCents = 20000
	in.Consumption = map[string]AnnualStatementConsumptionVector{"heizung": {PeriodYear: 2025, CostTypeKey: "heizung", Units: []AnnualStatementUnitConsumption{{UnitID: "a", ValueMicros: 1000000, MeasurementUnit: "kWh"}, {UnitID: "b", ValueMicros: 3000000, MeasurementUnit: "kWh"}}}}
	return in
}
func TestHeizKGHandCalculationAndLegacyReplay(t *testing.T) {
	// House: energy EUR 1,000; other operating costs EUR 200. Energy 70% consumption.
	// A: 25% consumption, 50% area => 700*0.25 + (300+200)*0.50 = EUR 425.
	// B: 75% consumption, 50% area => 700*0.75 + (300+200)*0.50 = EUR 775.
	in := heatingFixture()
	result, issues := CalculateAnnualStatementRun(in)
	if len(issues) > 0 || result.Units[0].AllocatedCents != 42500 || result.Units[1].AllocatedCents != 77500 || result.TotalCents != 120000 {
		t.Fatal(result, issues)
	}
	old, issues := ReplayAnnualStatementRun(AnnualStatementRun{CalculationVersion: 1, Input: in})
	if len(issues) > 0 || old.Units[0].AllocatedCents != 30000 || old.Units[1].AllocatedCents != 90000 {
		t.Fatal(old, issues)
	}
	replay, issues := ReplayAnnualStatementRun(AnnualStatementRun{CalculationVersion: 2, Input: in})
	if len(issues) > 0 || !reflect.DeepEqual(result, replay) {
		t.Fatal(replay, issues)
	}
	for _, share := range []int{55, 85} {
		in.Structure.Legal.HeatingConsumptionPercent = share
		r, issues := CalculateAnnualStatementRun(in)
		if len(issues) > 0 || r.Units[0].AllocatedCents+r.Units[1].AllocatedCents != 120000 {
			t.Fatal(r, issues)
		}
	}
}
func TestHeizKGBlocksIncompleteLegalBasis(t *testing.T) {
	for name, change := range map[string]func(*AnnualStatementRunInput){
		"low":              func(in *AnnualStatementRunInput) { in.Structure.Legal.HeatingConsumptionPercent = 54 },
		"high":             func(in *AnnualStatementRunInput) { in.Structure.Legal.HeatingConsumptionPercent = 86 },
		"missing area":     func(in *AnnualStatementRunInput) { delete(in.Structure.Legal.HeatableAreas, "a") },
		"missing category": func(in *AnnualStatementRunInput) { in.Receipts[0].HeatingCategory = "" },
		"wrong key":        func(in *AnnualStatementRunInput) { in.Structure.CostTypes[0].AllocationKey = AllocationKeyNutzwert },
	} {
		t.Run(name, func(t *testing.T) {
			in := heatingFixture()
			change(&in)
			r, issues := CalculateAnnualStatementRun(in)
			if len(issues) == 0 || len(r.Units) != 0 {
				t.Fatal(r, issues)
			}
		})
	}
}
