package server

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

type savedMRGRunRepository struct {
	store.AnnualStatementRunRepository
	run store.AnnualStatementRun
}

func (r savedMRGRunRepository) Preview(int, map[string]store.AnnualStatementConsumptionVector) (store.AnnualStatementRunInput, store.AnnualStatementRunResult, error) {
	return r.run.Input, r.run.Result, nil
}
func (r savedMRGRunRepository) List(int) ([]store.AnnualStatementRun, error) {
	return []store.AnnualStatementRun{r.run}, nil
}

func TestMRGResultSeparatesCopiesAndVacancyWithoutChangingSnapshot(t *testing.T) {
	run := store.AnnualStatementRun{ID: "saved", PeriodYear: 2025, CalculationVersion: 4,
		Input: store.AnnualStatementRunInput{
			Period:    store.AnnualStatementPeriod{StartsOn: "2025-01-01", EndsOn: "2025-12-31"},
			Structure: store.AnnualStatementPeriodStructure{Legal: store.AnnualStatementLegalSettings{Regime: "mrg_voll"}},
			Parties: []store.AnnualStatementRunParty{
				{UnitID: "top-1", ID: "landlord", Owner: true}, {UnitID: "top-1", ID: "tenant", Renter: true},
				{UnitID: "top-2", ID: "landlord", Owner: true}, {UnitID: "top-2", ID: "previous", Renter: true, ValidTo: "2025-06-30"}, {UnitID: "top-2", ID: "current", Renter: true, ValidFrom: "2025-07-01"},
				{UnitID: "top-6", ID: "landlord", Owner: true},
			},
		},
		Result: store.AnnualStatementRunResult{
			Units: []store.AnnualStatementRunUnit{
				{UnitID: "top-1", Label: "Top 1", AllocatedCents: 187047, PrepaidCents: 84000, BalanceCents: 103047},
				{UnitID: "top-2", Label: "Top 2", AllocatedCents: 148994, PrepaidCents: 84000, BalanceCents: 64994},
				{UnitID: "top-6", Label: "Top 6", AllocatedCents: 49617, PrepaidCents: 42000, BalanceCents: 7617},
			},
			PartyShares: []store.AnnualStatementPartyShare{
				{UnitID: "top-2", PartyID: "landlord", Unit: store.AnnualStatementRunUnit{UnitID: "top-2"}},
				{UnitID: "top-2", PartyID: "previous", Unit: store.AnnualStatementRunUnit{UnitID: "top-2", AllocatedCents: 28336, PrepaidCents: 9000, BalanceCents: 19336}},
				{UnitID: "top-2", PartyID: "current", Unit: store.AnnualStatementRunUnit{UnitID: "top-2", AllocatedCents: 120658, PrepaidCents: 75000, BalanceCents: 45658}},
			},
			Vacancy: []store.AnnualStatementVacancyLine{{UnitID: "top-6", Label: "Top 6", From: "2025-03-01", To: "2025-08-31", AmountCents: 50433}},
		},
	}
	before, _ := json.Marshal(run)
	view := annualStatementRunView(savedMRGRunRepository{run: run}, nil, 2025, "", "", nil)
	if len(view.Units) != 3 {
		t.Fatal(view.Units)
	}
	for i, count := range []int{1, 2, 2} {
		unit := view.Units[i]
		if len(unit.PDFs) != count || len(unit.Copies) != 1 {
			t.Fatalf("rows/copies: %+v", unit)
		}
	}
	if view.Units[0].PDFs[0].Label != "tenant" || view.Units[0].PDFs[0].Allocated != "1.870,47 €" {
		t.Fatal(view.Units[0])
	}
	top6 := view.Units[2].PDFs
	if !strings.Contains(top6[0].Label, "keine Mietpartei gespeichert") || top6[0].Allocated != "496,17 €" || top6[0].Prepaid != "420,00 €" || top6[0].Balance != "Nachzahlung 76,17 €" || top6[0].URL != "" {
		t.Fatal(top6)
	}
	if top6[0].Period != "01.01.2025 – 28.02.2025 · 01.09.2025 – 31.12.2025" {
		t.Fatal(top6[0].Period)
	}
	if !top6[1].Vacancy || top6[1].Allocated != "504,33 €" || top6[1].Prepaid != "0,00 €" {
		t.Fatal(top6[1])
	}
	after, _ := json.Marshal(run)
	if string(before) != string(after) {
		t.Fatal("presentation mutated saved run")
	}
	// A saved nonzero owner allocation must not disappear with information copies.
	run.Result.PartyShares[0].Unit.AllocatedCents = 1234
	run.Result.PartyShares[0].Unit.BalanceCents = 1234
	view = annualStatementRunView(savedMRGRunRepository{run: run}, nil, 2025, "", "", nil)
	if len(view.Units[1].PDFs) != 3 || view.Units[1].PDFs[0].Allocated != "12,34 €" || !strings.HasPrefix(view.Units[1].PDFs[0].Label, "Eigentümer trägt") {
		t.Fatal(view.Units[1])
	}
	// A dual-role PDF combines slices; table rows still count vacancy only once.
	run.Input.Parties[len(run.Input.Parties)-1].Renter = true
	view = annualStatementRunView(savedMRGRunRepository{run: run}, nil, 2025, "", "", nil)
	if view.Units[2].PDFs[0].Allocated != "496,17 €" || len(view.Units[2].PDFs) != 2 {
		t.Fatal(view.Units[2])
	}
}
