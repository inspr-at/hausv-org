package store

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"
)

func TestAnnualStatementSnapshotCanonicalConsumptionAndOwnedInput(t *testing.T) {
	input := annualRunFixture()
	input.Structure.CostTypes[0].Key = "heizung"
	input.Structure.CostTypes[0].AllocationKey = AllocationKeyVerbrauch
	input.Receipts[0].CostTypeKey = "heizung"
	input.Consumption = map[string]AnnualStatementConsumptionVector{"heizung": {PeriodYear: 2025, CostTypeKey: "heizung", Units: []AnnualStatementUnitConsumption{
		{UnitID: "b", ValueMicros: 3, MeasurementUnit: "kWh"}, {UnitID: "a", ValueMicros: 1, MeasurementUnit: "kWh"},
	}}}
	result, issues := CalculateAnnualStatementRun(input)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	before, _ := json.Marshal(input)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	first, err := newAnnualStatementRun(input, result, 1, "manager@example.com", now)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(input)
	if string(before) != string(after) {
		t.Error("snapshot construction mutated caller inputs")
	}
	slices.Reverse(input.Consumption["heizung"].Units)
	second, err := newAnnualStatementRun(input, result, 2, "manager@example.com", now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if first.InputHash != second.InputHash {
		t.Error("same measurements in a different row order changed input hash")
	}
	if !reflect.DeepEqual(first.Result, second.Result) {
		t.Error("identical facts changed calculation")
	}
	input.Consumption["heizung"].Units[0].ValueMicros = 999
	replayed, issues := CalculateAnnualStatementRun(first.Input)
	if len(issues) > 0 || !reflect.DeepEqual(replayed, result) {
		t.Error("caller mutation changed frozen snapshot")
	}
}
