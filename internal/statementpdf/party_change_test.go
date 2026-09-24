package statementpdf

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestDatedPartyPDFShowsOnlyItsShare(t *testing.T) {
	run := combinedRun(t, "weg")
	run.Input.StatementOn = "2026-06-01"
	run.Input.Parties[1].ValidTo = "2025-06-30"
	run.Input.Parties = append(run.Input.Parties, store.AnnualStatementRunParty{UnitID: "a", ID: "tenant-new", Name: "Neue Mietpartei", Renter: true, ValidFrom: "2025-07-01"})
	result, issues := store.CalculateAnnualStatementRun(run.Input)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	run.CalculationVersion = store.AnnualStatementCalculationVersionParties
	run.Result = result
	for _, party := range []string{"owner", "tenant", "tenant-new"} {
		allocation, ok := store.AnnualStatementPartyAllocation(run, "a", party)
		if !ok {
			t.Fatal(party)
		}
		docs, err := Documents(run, "a", party)
		if err != nil || len(docs) != 1 {
			t.Fatal(err)
		}
		doc := docs[0]
		if doc.Total != money(allocation.Unit.AllocatedCents) || doc.Prepaid != money(allocation.Unit.PrepaidCents) {
			t.Fatal(doc, allocation)
		}
		if !strings.Contains(strings.Join(doc.Basis, " "), "Parteienwechsel") || !strings.Contains(strings.Join(doc.Basis, " "), "monatliche Anteile") {
			t.Fatal(doc.Basis)
		}
		if party != "owner" && (len(doc.Reserve) > 0 || len(doc.Proposals) > 0) {
			t.Fatal("outgoing/heating-only party received owner forecast", doc)
		}
		text := strings.Join(doc.PaymentTerms, " ")
		if !strings.Contains(text, "01.08.2026") {
			t.Fatal(text)
		}
		for _, cost := range doc.Costs {
			if cost.Name == "Heizung" && !strings.Contains(strings.Join(cost.Measurements, " "), "Akonto dieser Heizkostenart: "+money(allocation.HeatingPrepayments["heizung"])) {
				t.Fatal(cost)
			}
		}
		raw, err := Render(run, "a", party)
		if err != nil || !bytes.HasPrefix(raw, []byte("%PDF-")) {
			t.Fatal(err)
		}
	}
	// A later approval changes the approval notice, never the snapshotted party
	// allocation or the due-date calculation used to select the recipient.
	run.Approval = &store.AnnualStatementRunApproval{ApprovedAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), ApprovedBy: "manager", Role: "Verwalter"}
	doc, err := Documents(run, "a", "tenant-new")
	if err != nil || !strings.Contains(strings.Join(doc[0].PaymentTerms, " "), "01.08.2026") {
		t.Fatal(err, doc)
	}
}
