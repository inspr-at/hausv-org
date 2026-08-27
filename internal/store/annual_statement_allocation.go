package store

import "sort"

// AnnualStatementUnitShare is one unit's share of an allocation key. Basis is
// the raw per-unit value the key reads (Miteigentumsanteil, m² hundredths,
// persons); SharePPM is the resulting split in parts per million. Mapped is
// false when the unit has no usable basis for this key.
type AnnualStatementUnitShare struct {
	UnitID   string
	Label    string
	Basis    int
	SharePPM int
	Mapped   bool
}

// AnnualStatementAllocationPreview shows, for one allocation key, how the
// allocatable cost types using it would be split across units before any run.
// Blocked is true when at least one unit is unmapped; a statement run must
// refuse to start while any key in use is blocked.
type AnnualStatementAllocationPreview struct {
	Key           string
	CostTypeKeys  []string
	Shares        []AnnualStatementUnitShare
	UnmappedUnits []string
	Blocked       bool
}

// AllocationBasis returns the raw value a unit contributes to a key and whether
// that value is usable. Zero is never a real basis: a unit without a recorded
// Miteigentumsanteil, area or person count is unmapped, not "0 %".
func AllocationBasis(key string, item Unit) (int, bool) {
	var basis int
	switch key {
	case AllocationKeyNutzwert:
		basis = item.MiteigentumsanteilPPM
	case AllocationKeyFlaeche:
		basis = item.UsableAreaM2Hundredths
	case AllocationKeyPersonen:
		basis = item.Persons
	default:
		// Verbrauch has no source until HAUSV-578 wires measured values.
		return 0, false
	}
	return basis, basis > 0
}

// AnnualStatementAllocationPreviews computes one preview per allocation key
// that at least one allocatable cost type uses, in AllocationKeys order.
// Shares use largest-remainder rounding so every mapped key sums to exactly
// 1 000 000 ppm; a later run can therefore split cents without drift.
func AnnualStatementAllocationPreviews(costTypes []AnnualStatementCostType, units []Unit) []AnnualStatementAllocationPreview {
	byKey := map[string][]string{}
	for _, costType := range costTypes {
		if costType.Allocatable && ValidAllocationKey(costType.AllocationKey) {
			byKey[costType.AllocationKey] = append(byKey[costType.AllocationKey], costType.Key)
		}
	}
	out := []AnnualStatementAllocationPreview{}
	for _, key := range AllocationKeys {
		costTypeKeys, used := byKey[key]
		if !used {
			continue
		}
		sort.Strings(costTypeKeys)
		preview := AnnualStatementAllocationPreview{Key: key, CostTypeKeys: costTypeKeys}
		total := 0
		for _, item := range units {
			basis, mapped := AllocationBasis(key, item)
			share := AnnualStatementUnitShare{UnitID: item.ID, Label: item.Label, Basis: basis, Mapped: mapped}
			if !mapped {
				preview.UnmappedUnits = append(preview.UnmappedUnits, item.Label)
			} else {
				total += basis
			}
			preview.Shares = append(preview.Shares, share)
		}
		preview.Blocked = len(units) == 0 || len(preview.UnmappedUnits) > 0
		if !preview.Blocked {
			distributePPM(preview.Shares, total)
		}
		out = append(out, preview)
	}
	return out
}

// distributePPM assigns SharePPM by largest remainder so the shares sum to
// exactly 1 000 000. Ties go to the earlier unit, keeping the result stable.
func distributePPM(shares []AnnualStatementUnitShare, total int) {
	const million = 1_000_000
	if total <= 0 {
		return
	}
	remainders := make([]struct{ index, remainder int }, 0, len(shares))
	assigned := 0
	for index := range shares {
		scaled := shares[index].Basis * million
		shares[index].SharePPM = scaled / total
		assigned += shares[index].SharePPM
		remainders = append(remainders, struct{ index, remainder int }{index, scaled % total})
	}
	sort.SliceStable(remainders, func(i, j int) bool { return remainders[i].remainder > remainders[j].remainder })
	for step := 0; step < million-assigned && step < len(remainders); step++ {
		shares[remainders[step].index].SharePPM++
	}
}
