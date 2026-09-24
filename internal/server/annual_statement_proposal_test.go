package server

import (
	"github.com/inspr-at/hausv-org/internal/store"
	"math"
	"testing"
)

type proposalRuns struct {
	store.AnnualStatementRunRepository
	runs []store.AnnualStatementRun
}

func (r proposalRuns) List(int) ([]store.AnnualStatementRun, error) { return r.runs, nil }
func TestAnnualPrefillUsesApprovedCompleteProposals(t *testing.T) {
	approved := store.AnnualStatementRun{Approval: &store.AnnualStatementRunApproval{}, Result: store.AnnualStatementRunResult{Proposals: []store.AnnualStatementPrepaymentProposal{{UnitID: "a", MonthlyCents: 100}, {UnitID: "a", MonthlyCents: 200}, {UnitID: "b", Missing: true}, {UnitID: "c", MonthlyCents: math.MaxInt64}}}}
	draft := store.AnnualStatementRun{Result: store.AnnualStatementRunResult{Proposals: []store.AnnualStatementPrepaymentProposal{{UnitID: "a", MonthlyCents: 999}}}}
	got := annualStatementPrepaymentPrefill(proposalRuns{runs: []store.AnnualStatementRun{draft, approved}}, 2026)
	if len(got) != 1 || got["a"] != 3600 {
		t.Fatal(got)
	}
}
