package store

// Zero is an explicit exclusion; missing, extra or invalid units block the key.
func validAgreedShareTotal(shares map[string]int) bool {
	total := 0
	for id, ppm := range shares {
		if id == "" || NormalizeUnitID(id) != id || ppm < 0 || ppm > 1_000_000-total {
			return false
		}
		total += ppm
	}
	return total == 1_000_000
}

func AnnualStatementAgreedPreview(cost string, units []Unit, shares map[string]int) AnnualStatementAllocationPreview {
	p := AnnualStatementAllocationPreview{Key: AllocationKeyAgreed, CostTypeKeys: []string{cost}}
	p.Blocked = !validAgreedShareTotal(shares) || len(shares) != len(units) || len(units) == 0
	seen := map[string]bool{}
	for _, unit := range units {
		ppm, found := shares[unit.ID]
		mapped := found && ppm >= 0 && ppm <= 1_000_000 && !seen[unit.ID]
		seen[unit.ID] = true
		if !mapped {
			p.Blocked = true
			p.UnmappedUnits = append(p.UnmappedUnits, unit.Label)
		}
		if mapped {
			p.BasisTotal += ppm
		}
		p.Shares = append(p.Shares, AnnualStatementUnitShare{UnitID: unit.ID, Label: unit.Label, Basis: ppm, SharePPM: ppm, Mapped: mapped})
	}
	return p
}

func annualStatementHasAgreedShares(input AnnualStatementRunInput) bool {
	for _, cost := range input.Structure.CostTypes {
		if cost.Allocatable && cost.AllocationKey == AllocationKeyAgreed {
			return true
		}
	}
	return false
}
