package server

import (
	"testing"

	storepkg "github.com/inspr-at/hausv-org/internal/store"
)

func TestFormatAnnualStatementShareRoundsHalfUpWithCarry(t *testing.T) {
	tests := []struct {
		ppm  int
		want string
	}{
		{0, "0,00 %"},
		{1, "0,00 %"},
		{49, "0,00 %"},
		{50, "0,01 %"},
		{99, "0,01 %"},
		{100, "0,01 %"},
		{333_333, "33,33 %"},
		{333_334, "33,33 %"},
		{547_149, "54,71 %"},
		{547_150, "54,72 %"},
		{547_170, "54,72 %"},
		{666_666, "66,67 %"},
		{666_667, "66,67 %"},
		{999_949, "99,99 %"},
		{999_950, "100,00 %"},
		{999_999, "100,00 %"},
		{1_000_000, "100,00 %"},
	}
	for _, test := range tests {
		if got := formatAnnualStatementShare(test.ppm, true); got != test.want {
			t.Errorf("formatAnnualStatementShare(%d) = %q, want %q", test.ppm, got, test.want)
		}
	}
	if got := formatAnnualStatementShare(500_000, false); got != "–" {
		t.Errorf("unmapped share must render no percentage, got %q", got)
	}

	// A 2:1 allocation proves that display rounding changes without touching
	// the exact largest-remainder allocation underneath.
	costTypes := []storepkg.AnnualStatementCostType{{
		Key: "grundsteuer", Allocatable: true, AllocationKey: storepkg.AllocationKeyNutzwert,
	}}
	units := []unit{
		{ID: "a", Label: "A", MiteigentumsanteilPPM: 2},
		{ID: "b", Label: "B", MiteigentumsanteilPPM: 1},
	}
	preview := storepkg.AnnualStatementAllocationPreviews(costTypes, units)[0]
	if preview.Shares[0].SharePPM+preview.Shares[1].SharePPM != 1_000_000 || preview.Shares[0].SharePPM != 666_667 {
		t.Fatalf("ppm allocation must stay exact: %+v", preview.Shares)
	}
	view := annualStatementAllocationView(costTypes, units, annualStatementConsumptionData{})
	if view.Previews[0].Shares[0].Share != "66,67 %" || view.Previews[0].Shares[1].Share != "33,33 %" {
		t.Fatalf("rendered shares = %q / %q", view.Previews[0].Shares[0].Share, view.Previews[0].Shares[1].Share)
	}
}
