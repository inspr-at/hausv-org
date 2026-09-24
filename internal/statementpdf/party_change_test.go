package statementpdf

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestDatedPartyPDFShowsOnlyItsShare(t *testing.T) {
	run := combinedRun(t, "mrg_voll")
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
		if party != "tenant-new" && (len(doc.Reserve) > 0 || len(doc.Proposals) > 0) {
			t.Fatal("outgoing/heating-only party received owner forecast", doc)
		}
		text := strings.Join(doc.PaymentTerms, " ")
		if !strings.Contains(text, "05.07.2026") {
			t.Fatal(text)
		}
		for _, cost := range doc.Costs {
			if cost.Amount == "0,00 €" {
				t.Fatal("zero cost row in party PDF", cost)
			}
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
	if err != nil || !strings.Contains(strings.Join(doc[0].PaymentTerms, " "), "05.07.2026") {
		t.Fatal(err, doc)
	}
}

func TestWEGOwnerChangePDFOperatingReserveAndHeating(t *testing.T) {
	run := combinedRun(t, "weg")
	run.Input.StatementOn = "2026-06-01"
	// Model the owner-only snapshot produced by the repository; route and
	// persistence tests cover exclusion of tenants from the live register.
	run.Input.Parties[0].ValidTo = "2025-06-30"
	run.Input.Parties[1] = store.AnnualStatementRunParty{UnitID: "a", ID: "new-owner", Owner: true, ValidFrom: "2025-07-01"}
	run.CalculationVersion = 4
	var issues []store.AnnualStatementRunIssue
	run.Result, issues = store.CalculateAnnualStatementRun(run.Input)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	old, err := Documents(run, "a", "owner")
	if err != nil || len(old) != 1 || len(old[0].Costs) != 1 || old[0].Costs[0].Name != "Heizung" || old[0].Total != "212,50 €" || len(old[0].Reserve) != 0 {
		t.Fatal("outgoing owner must receive only half the heating", old, err)
	}
	next, err := Documents(run, "a", "new-owner")
	if err != nil || len(next) != 1 || len(next[0].Costs) != 2 || len(next[0].Reserve) == 0 || next[0].Costs[1].Amount != "27,50 €" {
		t.Fatal("due-date owner must receive all operating costs and reserve", next, err)
	}
}
