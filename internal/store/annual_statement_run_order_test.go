package store

import (
	"strings"
	"testing"
)

// The demo register has stellplatz-1 … stellplatz-6 and top-1 … top-18; sorted
// as ID strings a Verwaltung sees Stellplätze first and Top 10 before Top 2.
func TestAnnualStatementRunDisplayOrderReadsLikeTheRegister(t *testing.T) {
	run := AnnualStatementRun{
		Input: AnnualStatementRunInput{Units: []AnnualStatementRunUnitIdentity{
			{ID: "stellplatz-1", Label: "Stellplatz 1", UnitType: UnitTypeParking},
			{ID: "top-10", Label: "Top 10", UnitType: UnitTypeResidential},
			{ID: "top-2", Label: "Top 2", UnitType: UnitTypeResidential},
			{ID: "top-1", Label: "Top 1"},
			{ID: "stellplatz-10", Label: "Stellplatz 10", UnitType: "stellplatz"},
		}},
		Result: AnnualStatementRunResult{Units: []AnnualStatementRunUnit{
			{UnitID: "stellplatz-1", Label: "Stellplatz 1"},
			{UnitID: "stellplatz-10", Label: "Stellplatz 10"},
			{UnitID: "top-1", Label: "Top 1"},
			{UnitID: "top-10", Label: "Top 10"},
			{UnitID: "top-2", Label: "Top 2"},
		}},
	}
	got := make([]string, 0, 5)
	for _, unit := range AnnualStatementRunDisplayOrder(run) {
		got = append(got, unit.Label)
	}
	if want := "Top 1, Top 2, Top 10, Stellplatz 1, Stellplatz 10"; strings.Join(got, ", ") != want {
		t.Fatalf("display order = %q, want %q", strings.Join(got, ", "), want)
	}
	if run.Result.Units[0].UnitID != "stellplatz-1" {
		t.Fatal("display ordering must not reorder the stored result")
	}
}

// A run stored before the unit type existed still sorts naturally by label.
func TestAnnualStatementRunDisplayOrderWithoutSnapshotTypes(t *testing.T) {
	run := AnnualStatementRun{Result: AnnualStatementRunResult{Units: []AnnualStatementRunUnit{
		{UnitID: "b", Label: "Top 10"}, {UnitID: "a", Label: "Top 9"}, {UnitID: "c", Label: "Top 9"},
	}}}
	got := AnnualStatementRunDisplayOrder(run)
	if got[0].UnitID != "a" || got[1].UnitID != "c" || got[2].UnitID != "b" {
		t.Fatalf("fallback order = %v", got)
	}
}
