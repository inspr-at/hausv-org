package statementpdf

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/view"
)

func TestCalculatedRunMatchesEveryPartyPDF(t *testing.T) {
	input := store.AnnualStatementRunInput{
		Period: store.AnnualStatementPeriod{Year: 2024, StartsOn: "2024-01-01", EndsOn: "2024-12-31"},
		Units:  []store.AnnualStatementRunUnitIdentity{{ID: "a", Label: "Top 1"}, {ID: "b", Label: "Top 2"}, {ID: "c", Label: "Top 3"}},
		Structure: store.AnnualStatementPeriodStructure{
			CostTypes: []store.AnnualStatementCostType{{Key: "wasser", Name: "Wasser", Allocatable: true, AllocationKey: store.AllocationKeyPersonen}},
			UnitBases: []store.AnnualStatementPeriodUnitBasis{{UnitID: "a", Persons: 1, PersonsRecorded: true}, {UnitID: "b", Persons: 2, PersonsRecorded: true}, {UnitID: "c", PersonsRecorded: true}},
		},
		Receipts:    []store.AnnualStatementReceipt{{ID: "r", DocumentID: "d", PeriodYear: 2024, CostTypeKey: "wasser", AmountCents: 10001, InvoiceDate: "2024-02-29"}},
		Documents:   []store.AnnualStatementRunDocument{{ID: "d"}},
		Prepayments: []store.AnnualStatementPrepayment{{PeriodYear: 2024, UnitID: "a", AmountCents: 4000}, {PeriodYear: 2024, UnitID: "b", AmountCents: 5000}, {PeriodYear: 2024, UnitID: "c"}},
		Parties:     []store.AnnualStatementRunParty{{UnitID: "a", ID: "owner@example.com", Owner: true}, {UnitID: "a", ID: "tenant@example.com", Renter: true}, {UnitID: "b", ID: "second@example.com", Owner: true}, {UnitID: "c", ID: "vacant-owner@example.com", Owner: true}},
	}
	result, issues := store.CalculateAnnualStatementRun(input)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	run := store.AnnualStatementRun{ID: "calculated", PeriodYear: 2024, Revision: 1, CalculationVersion: store.AnnualStatementCalculationVersion, CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Input: input, Result: result}
	documents, err := Documents(run, "", "")
	if err != nil || len(documents) != 4 {
		t.Fatalf("documents: %+v %v", documents, err)
	}
	for _, unit := range result.Units {
		for _, doc := range documents {
			if doc.UnitID != unit.UnitID {
				continue
			}
			if doc.Total != view.FormatEURCents(unit.AllocatedCents) || doc.Prepaid != view.FormatEURCents(unit.PrepaidCents) || doc.Costs[0].Amount != view.FormatEURCents(unit.Costs[0].AmountCents) || doc.Costs[0].Total != view.FormatEURCents(result.TotalCents) {
				t.Fatalf("PDF totals differ from calculated run: %+v", doc)
			}
			want := "Ausgeglichen 0,00 €"
			if unit.BalanceCents > 0 {
				want = "Nachzahlung " + view.FormatEURCents(unit.BalanceCents)
			}
			if unit.BalanceCents < 0 {
				want = "Guthaben " + view.FormatEURCents(-unit.BalanceCents)
			}
			if doc.Balance != want {
				t.Fatalf("balance %q != %q", doc.Balance, want)
			}
			pdf, err := Render(run, doc.UnitID, doc.PartyID)
			if err != nil {
				t.Fatal(err)
			}
			for _, value := range []string{doc.Total, doc.Prepaid, doc.Balance, doc.Costs[0].Total, doc.Costs[0].Amount} {
				if !bytes.Contains(pdf, []byte(strings.TrimSuffix(value, " €"))) {
					t.Fatalf("PDF bytes omit %q", value)
				}
			}
		}
	}
	raw, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	var stored store.AnnualStatementRun
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatal(err)
	}
	one, err := Render(run, "", "")
	if err != nil {
		t.Fatal(err)
	}
	two, err := Render(stored, "", "")
	if err != nil || !bytes.Equal(one, two) {
		t.Fatal("calculated PDF changed after persistence", err)
	}
}
