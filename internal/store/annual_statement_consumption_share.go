package store

import (
	"math/big"
	"sort"
)

// AnnualStatementConsumptionShares accepts only a complete period vector.
// Values remain integer micro-units; big integers avoid overflow in sums and
// the multiplication by a million. A measured zero is distinct from a gap.
func AnnualStatementConsumptionShares(vector AnnualStatementConsumptionVector, units []Unit) ([]AnnualStatementUnitShare, bool) {
	if len(units) == 0 || len(vector.Gaps) != 0 || len(vector.Units) != len(units) {
		return nil, false
	}
	values := map[string]int64{}
	measurementUnit := ""
	for _, item := range vector.Units {
		id := NormalizeUnitID(item.UnitID)
		switch item.MeasurementUnit {
		case "Wh", "kWh", "MWh", "m³", "l":
		default:
			return nil, false
		}
		if _, duplicate := values[id]; duplicate || id == "" || item.ValueMicros < 0 || item.MeasurementUnit == "" {
			return nil, false
		}
		if measurementUnit != "" && measurementUnit != item.MeasurementUnit {
			return nil, false
		}
		measurementUnit = item.MeasurementUnit
		values[id] = item.ValueMicros
	}
	total := new(big.Int)
	shares := make([]AnnualStatementUnitShare, len(units))
	for i, unit := range units {
		id := NormalizeUnitID(unit.ID)
		value, found := values[id]
		if !found {
			return nil, false
		}
		delete(values, id)
		total.Add(total, big.NewInt(value))
		shares[i] = AnnualStatementUnitShare{UnitID: unit.ID, Label: unit.Label, Mapped: true}
	}
	if total.Sign() <= 0 {
		return nil, false
	}
	values = map[string]int64{}
	for _, item := range vector.Units {
		values[NormalizeUnitID(item.UnitID)] = item.ValueMicros
	}
	type remainder struct {
		index int
		value *big.Int
	}
	remainders := make([]remainder, len(shares))
	assigned := 0
	for i := range shares {
		scaled := new(big.Int).Mul(big.NewInt(values[NormalizeUnitID(shares[i].UnitID)]), big.NewInt(1_000_000))
		quotient, rest := new(big.Int), new(big.Int)
		quotient.QuoRem(scaled, total, rest)
		shares[i].SharePPM = int(quotient.Int64())
		assigned += shares[i].SharePPM
		remainders[i] = remainder{i, rest}
	}
	sort.SliceStable(remainders, func(i, j int) bool { return remainders[i].value.Cmp(remainders[j].value) > 0 })
	for i := 0; i < 1_000_000-assigned; i++ {
		shares[remainders[i].index].SharePPM++
	}
	return shares, true
}
