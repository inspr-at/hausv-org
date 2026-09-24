package store

import (
	"math"
	"math/big"
	"sort"
)

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
	Key          string
	CostTypeKeys []string
	Shares       []AnnualStatementUnitShare
	// BasisTotal is the denominator the shares were computed against. For
	// Nutzwert a total other than 1 000 000 ppm means a unit's share is
	// missing or the units were recorded as raw Nutzwerte; the caller must
	// show it rather than let the remaining units absorb it silently.
	BasisTotal    int
	UnmappedUnits []string
	Blocked       bool
}

// MiteigentumsanteilTotalPPM is the expected Nutzwert denominator when every
// unit's Miteigentumsanteil has been recorded in parts per million.
const MiteigentumsanteilTotalPPM = 1_000_000

// AllocationBasis returns the raw value a unit contributes to a key and whether
// that value is usable. Nutzwert reads the Miteigentumsanteil, where 0 means
// "not recorded" (the unit form has no separate flag) and is unmapped.
// Nutzfläche and Personen carry an explicit Recorded flag: blank is unmapped
// and blocks, an explicitly recorded 0 (Stellplatz without Nutzfläche, vacant
// flat without persons) is mapped with a zero share.
func AllocationBasis(key string, item Unit) (int, bool) {
	switch key {
	case AllocationKeyNutzwert:
		return item.MiteigentumsanteilPPM, item.MiteigentumsanteilPPM > 0
	case AllocationKeyFlaeche:
		return item.UsableAreaM2Hundredths, item.UsableAreaRecorded && item.UsableAreaM2Hundredths >= 0
	case AllocationKeyPersonen:
		return item.Persons, item.PersonsRecorded && item.Persons >= 0
	default:
		// Consumption is supplied separately as period-bound meter evidence.
		return 0, false
	}
}

// AnnualStatementCostTypesWithoutKey lists allocatable cost types that carry
// no valid allocation key, e.g. rows written before migration 0036 into a
// store that was not backfilled. They violate "one key per allocatable cost
// type" and must block a run rather than be skipped.
func AnnualStatementCostTypesWithoutKey(costTypes []AnnualStatementCostType) []string {
	out := []string{}
	for _, costType := range costTypes {
		if costType.Allocatable && !ValidAllocationKey(costType.AllocationKey) {
			out = append(out, costType.Key)
		}
	}
	return out
}

// AnnualStatementRunBlocked is the single answer a statement run (HAUSV-580)
// must consult before touching money: true when any allocatable cost type has
// no key or any key in use has an unmapped unit.
func AnnualStatementRunBlocked(costTypes []AnnualStatementCostType, units []Unit) bool {
	if len(AnnualStatementCostTypesWithoutKey(costTypes)) > 0 {
		return true
	}
	for _, preview := range AnnualStatementAllocationPreviews(costTypes, units) {
		if preview.Blocked {
			return true
		}
	}
	return false
}

// AnnualStatementAllocationPreviews computes one preview per allocation key
// that at least one allocatable cost type uses, in AllocationKeys order.
// Shares use largest-remainder rounding so every mapped key sums to exactly
// 1 000 000 ppm; a later run can therefore split cents without drift.
func AnnualStatementAllocationPreviews(costTypes []AnnualStatementCostType, units []Unit, agreed ...map[string]map[string]int) []AnnualStatementAllocationPreview {
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
		if key == AllocationKeyAgreed {
			for _, cost := range costTypeKeys {
				var shares map[string]int
				if len(agreed) > 0 {
					shares = agreed[0][cost]
				}
				out = append(out, AnnualStatementAgreedPreview(cost, units, shares))
			}
			continue
		}
		preview := AnnualStatementAllocationPreview{Key: key, CostTypeKeys: costTypeKeys}
		total := 0
		overflow := false
		for _, item := range units {
			basis, mapped := AllocationBasis(key, item)
			share := AnnualStatementUnitShare{UnitID: item.ID, Label: item.Label, Basis: basis, Mapped: mapped}
			if !mapped {
				preview.UnmappedUnits = append(preview.UnmappedUnits, item.Label)
			} else {
				if basis > math.MaxInt-total {
					overflow = true
				} else {
					total += basis
				}
			}
			preview.Shares = append(preview.Shares, share)
		}
		preview.BasisTotal = total
		// A key whose mapped bases sum to zero (every unit recorded 0 persons)
		// has nothing to split; treat it as blocked rather than divide by zero.
		preview.Blocked = overflow || len(units) == 0 || len(preview.UnmappedUnits) > 0 || total <= 0
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
		scaled := new(big.Int).Mul(big.NewInt(int64(shares[index].Basis)), big.NewInt(million))
		quotient, rest := new(big.Int), new(big.Int)
		quotient.QuoRem(scaled, big.NewInt(int64(total)), rest)
		shares[index].SharePPM = int(quotient.Int64())
		assigned += shares[index].SharePPM
		remainders = append(remainders, struct{ index, remainder int }{index, int(rest.Int64())})
	}
	sort.SliceStable(remainders, func(i, j int) bool { return remainders[i].remainder > remainders[j].remainder })
	for step := 0; step < million-assigned && step < len(remainders); step++ {
		shares[remainders[step].index].SharePPM++
	}
}
