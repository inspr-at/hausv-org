package store

import (
	"math/big"
	"sort"
)

const AnnualStatementCalculationVersionVAT = 3

// DefaultAnnualStatementVATPercent is 20 % for heat and hot water and 10 %
// for every other cost type. Receipts stay the gross invoice total; the rate
// only splits that gross when the period shows VAT.
func DefaultAnnualStatementVATPercent(key string) int {
	if key == "heizung" || key == "warmwasser" {
		return 20
	}
	return 10
}

func ValidAnnualStatementVATPercent(percent int) bool {
	return percent == 0 || percent == 10 || percent == 20
}

// annualStatementPeriodVATPercent keeps an explicit 10 or 20. A zero on the
// first snapshot is the unset catalogue value and takes the kind default.
// The cost-type form writes 0 % through the update path instead.
func annualStatementPeriodVATPercent(cost AnnualStatementCostType) int {
	if cost.VATRatePercent == 10 || cost.VATRatePercent == 20 {
		return cost.VATRatePercent
	}
	return DefaultAnnualStatementVATPercent(cost.Key)
}

// AnnualStatementVATGroup is one rate's net, VAT and gross. The gross is the
// allocated receipt total; net and VAT are rounded once for that rate.
type AnnualStatementVATGroup struct {
	RatePercent int   `json:"rate_percent"`
	NetCents    int64 `json:"net_cents"`
	VATCents    int64 `json:"vat_cents"`
	GrossCents  int64 `json:"gross_cents"`
}

// annualStatementVATFromGross rounds VAT half up from a gross amount.
// 10 % of gross is gross×10/110; 20 % is gross×20/120; 0 % is zero.
func annualStatementVATFromGross(gross int64, rate int) int64 {
	if gross <= 0 || rate == 0 {
		return 0
	}
	num := new(big.Int).Mul(big.NewInt(gross), big.NewInt(int64(rate)))
	div := big.NewInt(int64(100 + rate))
	quot, rem := new(big.Int).QuoRem(num, div, new(big.Int))
	if new(big.Int).Lsh(rem, 1).Cmp(div) >= 0 {
		quot.Add(quot, big.NewInt(1))
	}
	return quot.Int64()
}

// distributeLargestRemainder splits amount across weights. Weights of zero
// stay zero. Equal remainders keep the earlier index, matching the cost split.
func distributeLargestRemainder(amount int64, weights []int64) []int64 {
	out := make([]int64, len(weights))
	var sum int64
	for _, weight := range weights {
		sum += weight
	}
	if amount == 0 || sum == 0 || len(weights) == 0 {
		return out
	}
	type remainder struct {
		index int
		value *big.Int
	}
	rests := make([]remainder, len(weights))
	total := big.NewInt(amount)
	denom := big.NewInt(sum)
	var assigned int64
	for i, weight := range weights {
		prod := new(big.Int).Mul(total, big.NewInt(weight))
		quot, rem := new(big.Int).QuoRem(prod, denom, new(big.Int))
		out[i] = quot.Int64()
		assigned += out[i]
		rests[i] = remainder{i, rem}
	}
	sort.SliceStable(rests, func(i, j int) bool { return rests[i].value.Cmp(rests[j].value) > 0 })
	for n := int64(0); n < amount-assigned && n < int64(len(rests)); n++ {
		out[rests[n].index]++
	}
	return out
}

// applyAnnualStatementVAT splits each unit's gross by rate group. The group
// VAT is rounded once on the house gross, then largest-remainder assigns it
// to units and to cost lines so every sum matches the gross to the cent.
func applyAnnualStatementVAT(result *AnnualStatementRunResult, input AnnualStatementRunInput) string {
	rateOf := map[string]int{}
	for _, cost := range input.Structure.CostTypes {
		if !ValidAnnualStatementVATPercent(cost.VATRatePercent) {
			return "vat-rate"
		}
		rateOf[cost.Key] = cost.VATRatePercent
	}
	seen := map[int]bool{}
	var rates []int
	for _, cost := range input.Structure.CostTypes {
		if !cost.Allocatable || seen[rateOf[cost.Key]] {
			continue
		}
		seen[rateOf[cost.Key]] = true
		rates = append(rates, rateOf[cost.Key])
	}
	sort.Ints(rates)
	for _, rate := range rates {
		weights := make([]int64, len(result.Units))
		var houseGross int64
		for ui := range result.Units {
			for _, line := range result.Units[ui].Costs {
				if rateOf[line.CostTypeKey] != rate {
					continue
				}
				weights[ui] += line.AmountCents
				houseGross += line.AmountCents
			}
		}
		groupVAT := annualStatementVATFromGross(houseGross, rate)
		unitVATs := distributeLargestRemainder(groupVAT, weights)
		result.VATGroups = append(result.VATGroups, AnnualStatementVATGroup{
			RatePercent: rate, NetCents: houseGross - groupVAT, VATCents: groupVAT, GrossCents: houseGross,
		})
		for ui := range result.Units {
			var indexes []int
			var lineWeights []int64
			for i, line := range result.Units[ui].Costs {
				if rateOf[line.CostTypeKey] != rate {
					continue
				}
				indexes = append(indexes, i)
				lineWeights = append(lineWeights, line.AmountCents)
			}
			if len(indexes) == 0 {
				continue
			}
			lineVATs := distributeLargestRemainder(unitVATs[ui], lineWeights)
			var unitNet, unitGross int64
			for j, idx := range indexes {
				line := &result.Units[ui].Costs[idx]
				line.VATRatePercent = rate
				line.VATCents = lineVATs[j]
				line.NetCents = line.AmountCents - line.VATCents
				unitNet += line.NetCents
				unitGross += line.AmountCents
			}
			result.Units[ui].VAT = append(result.Units[ui].VAT, AnnualStatementVATGroup{
				RatePercent: rate, NetCents: unitNet, VATCents: unitVATs[ui], GrossCents: unitGross,
			})
		}
	}
	return ""
}
