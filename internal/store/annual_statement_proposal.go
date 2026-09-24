package store

import "math/big"

type AnnualStatementPrepaymentProposal struct {
	UnitID          string `json:"unit_id"`
	Component       string `json:"component"`
	Name            string `json:"name"`
	MonthlyCents    int64  `json:"monthly_cents"`
	Basis           string `json:"basis"`
	Missing         bool   `json:"missing"`
	AboveTenPercent bool   `json:"above_ten_percent"`
}

func AnnualStatementPrepaymentProposals(input AnnualStatementRunInput, result AnnualStatementRunResult) []AnnualStatementPrepaymentProposal {
	legal := input.Structure.Legal
	if legal.Regime == "" {
		return nil
	}
	var out []AnnualStatementPrepaymentProposal
	for _, unit := range result.Units {
		for _, cost := range unit.Costs {
			proposal := AnnualStatementPrepaymentProposal{UnitID: unit.UnitID, Component: cost.CostTypeKey, Name: cost.Name}
			manual, found := legal.MonthlyProposals[unit.UnitID][cost.CostTypeKey]
			heating := legal.HeizKGApplies && IsAnnualHeatingCost(cost.CostTypeKey)
			if heating || legal.Regime == "mrg_voll" {
				proposal.MonthlyCents = cost.AmountCents / 12
				if cost.AmountCents%12 >= 6 {
					proposal.MonthlyCents++
				}
				proposal.Basis = "Vorperiode / 12"
			} else {
				proposal.Basis = "Manuelle Vorausschau / Vereinbarung"
				proposal.Missing = !found
			}
			if found && !heating {
				proposal.MonthlyCents = manual
			}
			if proposal.MonthlyCents < 0 {
				proposal.Missing = true
			}
			if legal.Regime == "mrg_voll" && !heating && found {
				annual := new(big.Int).Mul(big.NewInt(proposal.MonthlyCents), big.NewInt(1200))
				limit := new(big.Int).Mul(big.NewInt(cost.AmountCents), big.NewInt(110))
				proposal.AboveTenPercent = annual.Cmp(limit) > 0
			}
			out = append(out, proposal)
		}
	}
	return out
}
