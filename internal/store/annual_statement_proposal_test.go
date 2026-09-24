package store

import "testing"

func TestAnnualPrepaymentProposalByRegime(t *testing.T) {
	for _, regime := range []string{"mrg_voll", "weg", "mrg_teil"} {
		in := annualRunFixture()
		in.Structure.Legal.Regime = regime
		result, issues := CalculateAnnualStatementRun(in)
		if len(issues) > 0 {
			t.Fatal(issues)
		}
		proposals := AnnualStatementPrepaymentProposals(in, result)
		if len(proposals) != 4 {
			t.Fatal(proposals)
		}
		if regime == "mrg_voll" {
			if proposals[0].MonthlyCents != 4 || proposals[0].Missing {
				t.Fatal(proposals)
			}
			// A's Betreuung annual cost is 51 cents. A manual EUR 1/month is >110%
			// of 51/12 cents; compare exact integer annual amounts, not rounded bases.
			in.Structure.Legal.MonthlyProposals = map[string]map[string]int64{"a": {"service": 100}}
			proposals = AnnualStatementPrepaymentProposals(in, result)
			if !proposals[0].AboveTenPercent {
				t.Fatal(proposals)
			}
		} else if !proposals[0].Missing {
			t.Fatal("WEG/contract require manual value", proposals)
		}
	}
	in := heatingFixture()
	result, issues := CalculateAnnualStatementRun(in)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	proposals := AnnualStatementPrepaymentProposals(in, result)
	if proposals[0].MonthlyCents != 3542 || proposals[1].MonthlyCents != 6458 || proposals[0].Missing {
		t.Fatal(proposals)
	}
}

func TestAutomaticMRGProposalDoesNotWarnOnCentRounding(t *testing.T) {
	input := AnnualStatementRunInput{Structure: AnnualStatementPeriodStructure{Legal: AnnualStatementLegalSettings{Regime: "mrg_voll"}}}
	result := AnnualStatementRunResult{Units: []AnnualStatementRunUnit{{UnitID: "a", Costs: []AnnualStatementRunCost{{CostTypeKey: "water", AmountCents: 6}}}}}
	got := AnnualStatementPrepaymentProposals(input, result)
	if got[0].MonthlyCents != 1 || got[0].AboveTenPercent {
		t.Fatal(got)
	}
}
