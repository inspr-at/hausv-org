package store

import "sort"

// AnnualStatementRunDisplayOrder returns a copy of the run's unit results in
// the order a register reads them (HAUSV-639): residential and other units
// first, parking last, each group naturally by label (Top 2 before Top 10),
// then by ID. The calculation keeps its own ID order because the cent
// distribution breaks ties on it; only the presentation sorts. Runs stored
// before the unit type was snapshotted fall back to labels alone.
func AnnualStatementRunDisplayOrder(run AnnualStatementRun) []AnnualStatementRunUnit {
	types := make(map[string]string, len(run.Input.Units))
	for _, identity := range run.Input.Units {
		types[identity.ID] = identity.UnitType
	}
	units := append([]AnnualStatementRunUnit(nil), run.Result.Units...)
	sort.SliceStable(units, func(i, j int) bool {
		return annualStatementRunUnitDisplayLess(units[i], types[units[i].UnitID], units[j], types[units[j].UnitID])
	})
	return units
}

func annualStatementRunUnitDisplayLess(a AnnualStatementRunUnit, aType string, b AnnualStatementRunUnit, bType string) bool {
	aParking := NormalizeUnitType(aType) == UnitTypeParking
	bParking := NormalizeUnitType(bType) == UnitTypeParking
	if aParking != bParking {
		return !aParking
	}
	if a.Label != b.Label {
		return UnitLabelLess(a.Label, b.Label)
	}
	return a.UnitID < b.UnitID
}
