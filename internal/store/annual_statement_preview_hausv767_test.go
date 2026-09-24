package store

import (
	"math"
	"testing"
)

func TestAnnualStatementPreviewRoundsPerCostInStableUnitOrder(t *testing.T) {
	input := annualRunFixture()
	for i := range input.Structure.CostTypes[:2] {
		input.Structure.CostTypes[i].AllocationKey = AllocationKeyPersonen
		input.Receipts[i].AmountCents = 1
	}
	run, issues := CalculateAnnualStatementRun(input)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	units := []Unit{{ID: "b", Persons: 1, PersonsRecorded: true}, {ID: "a", Persons: 1, PersonsRecorded: true}}
	preview, ready := AnnualStatementSettlementPreview(input.Structure.CostTypes, input.Receipts, units)
	if !ready {
		t.Fatal("preview blocked")
	}
	for _, row := range preview {
		for _, result := range run.Units {
			if row.UnitID == result.UnitID && row.AllocatedCents != result.AllocatedCents {
				t.Errorf("unit %s: preview %d != run %d", row.UnitID, row.AllocatedCents, result.AllocatedCents)
			}
		}
	}
}

func TestAnnualStatementPreviewBlocksInvalidNutzwertTotalAndHouseOverflow(t *testing.T) {
	for _, tc := range []struct {
		name   string
		key    string
		amount int64
		basis  int
	}{
		{"incomplete shares", AllocationKeyNutzwert, 1, 1},
		{"house total overflow", AllocationKeyPersonen, math.MaxInt64, 500000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			costs := []AnnualStatementCostType{{Key: "a", Allocatable: true, AllocationKey: tc.key}, {Key: "b", Allocatable: true, AllocationKey: AllocationKeyFlaeche}}
			units := []Unit{{ID: "a", MiteigentumsanteilPPM: tc.basis, Persons: 1, PersonsRecorded: true, UsableAreaRecorded: true}, {ID: "b", MiteigentumsanteilPPM: tc.basis, PersonsRecorded: true, UsableAreaRecorded: true, UsableAreaM2Hundredths: 1}}
			receipts := []AnnualStatementReceipt{{CostTypeKey: "a", AmountCents: tc.amount}, {CostTypeKey: "b", AmountCents: tc.amount}}
			if got, ready := AnnualStatementSettlementPreview(costs, receipts, units); ready || len(got) != 0 {
				t.Fatalf("published invalid preview: %+v", got)
			}
		})
	}
}
