package statementpdf

import (
	"github.com/inspr-at/hausv-org/internal/store"
	"strings"
	"testing"
	"time"
)

func TestStatementPaymentTerms(t *testing.T) {
	for _, tc := range []struct {
		regime string
		heat   bool
		day    int
		want   string
	}{
		{"mrg_voll", false, 3, "05.07.2026"}, {"mrg_voll", false, 6, "05.08.2026"},
		{"weg", false, 30, "30.08.2026"}, {"mrg_teil", true, 30, "30.08.2026"},
		{"ausnahme", false, 30, "Vertrag"},
	} {
		run := fixture()
		if tc.regime == "mrg_voll" || tc.regime == "mrg_teil" {
			run.Input.Parties[0].Owner, run.Input.Parties[0].Renter = false, true
		}
		run.Input.Structure.Legal = store.AnnualStatementLegalSettings{Regime: tc.regime, HeizKGApplies: tc.heat}
		run.Approval = &store.AnnualStatementRunApproval{ApprovedAt: time.Date(2026, 6, tc.day, 12, 0, 0, 0, time.UTC), Role: store.RoleManager}
		docs, err := Documents(run, "b", "zoe@example.com")
		if err != nil || !strings.Contains(strings.Join(docs[0].PaymentTerms, " "), tc.want) {
			t.Fatal(tc, docs, err)
		}
	}
	run := fixture()
	run.Input.Structure.Legal.Regime = "weg"
	docs, _ := Documents(run, "a", "owner@example.com")
	if !strings.Contains(strings.Join(docs[0].PaymentTerms, " "), "künftige Vorauszahlungen") {
		t.Fatal(docs)
	}
}
