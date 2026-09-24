package statementpdf

import (
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestOwnerStatementCarriesVacancyShare(t *testing.T) {
	input := store.AnnualStatementRunInput{
		Period: store.AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31"},
		Structure: store.AnnualStatementPeriodStructure{
			Legal:     store.AnnualStatementLegalSettings{Regime: "mrg_voll", HeatingConsumptionPercent: 70},
			CostTypes: []store.AnnualStatementCostType{{Key: "tax", Name: "Abgabe", Allocatable: true, AllocationKey: store.AllocationKeyNutzwert}},
			UnitBases: []store.AnnualStatementPeriodUnitBasis{
				{UnitID: "a", MiteigentumsanteilPPM: 250000, VacantFrom: "2025-03-01", VacantTo: "2025-05-31"},
				{UnitID: "b", MiteigentumsanteilPPM: 750000},
			},
		},
		Units:       []store.AnnualStatementRunUnitIdentity{{ID: "a", Label: "Top 1"}, {ID: "b", Label: "Top 2"}},
		Receipts:    []store.AnnualStatementReceipt{{ID: "r1", DocumentID: "d1", PeriodYear: 2025, CostTypeKey: "tax", AmountCents: 36500, InvoiceDate: "2025-02-01"}},
		Documents:   []store.AnnualStatementRunDocument{{ID: "d1"}},
		Prepayments: []store.AnnualStatementPrepayment{{PeriodYear: 2025, UnitID: "a", AmountCents: 1000}, {PeriodYear: 2025, UnitID: "b"}},
		Parties: []store.AnnualStatementRunParty{
			{UnitID: "a", ID: "owner@example.com", Name: "Eigentümer", Owner: true},
			{UnitID: "a", ID: "tenant@example.com", Name: "Mieter", Renter: true},
		},
	}
	result, issues := store.CalculateAnnualStatementRun(input)
	if len(issues) > 0 || len(result.Vacancy) != 1 {
		t.Fatal(issues, result.Vacancy)
	}
	run := store.AnnualStatementRun{ID: "vacancy", PeriodYear: 2025, Revision: 1, CalculationVersion: store.AnnualStatementCalculationVersion, CreatedAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), Input: input, Result: result}
	owner, err := Documents(run, "a", "owner@example.com")
	if err != nil || len(owner) != 1 {
		t.Fatal(err, owner)
	}
	if !strings.Contains(strings.Join(owner[0].Basis, "\n"), store.AnnualStatementVacancyLabel) || owner[0].Total != money(result.Vacancy[0].AmountCents) {
		t.Fatalf("owner letter: %+v", owner[0])
	}
	tenant, err := Documents(run, "a", "tenant@example.com")
	if err != nil || tenant[0].Total != money(result.Units[0].AllocatedCents) {
		t.Fatalf("tenant letter: %+v %v", tenant, err)
	}
	if _, err := Render(run, "a", "owner@example.com"); err != nil {
		t.Fatal(err)
	}
}
