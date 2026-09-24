package statementpdf

import (
	"github.com/inspr-at/hausv-org/internal/store"
	"strings"
	"testing"
)

func TestMonthlyProposalIsPrinted(t *testing.T) {
	run := fixture()
	run.Input.Structure.Legal.NextPrepaymentOn = "2026-05-01"
	run.Result.Proposals = []store.AnnualStatementPrepaymentProposal{{UnitID: "a", Name: "Heizung", MonthlyCents: 3542, Basis: "Vorperiode / 12"}}
	docs, err := Documents(run, "a", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(docs[0].Proposals, " ")
	if !strings.Contains(text, "Neue monatliche Vorauszahlung ab 01.05.2026:") || !strings.Contains(text, "35,42 €") {
		t.Fatal(text)
	}
}

func TestMonthlyProposalUsesOwnerLanguage(t *testing.T) {
	for _, tc := range []struct{ basis, want string }{
		{"Manuelle Vorausschau / Vereinbarung", "vereinbarte monatliche Vorauszahlung"},
		{"Vorperiode / 12", "Vorauszahlung auf Basis des Vorjahres"},
	} {
		t.Run(tc.basis, func(t *testing.T) {
			run := fixture()
			run.Result.Proposals = []store.AnnualStatementPrepaymentProposal{{UnitID: "a", Name: "Wasser", MonthlyCents: 1000, Basis: tc.basis}}
			text := strings.Join(proposalLines(run, "a"), " ")
			if !strings.Contains(text, tc.want) || strings.Contains(text, tc.basis) {
				t.Fatal(text)
			}
			if run.Result.Proposals[0].Basis != tc.basis {
				t.Fatal("stored basis changed")
			}
		})
	}
}
