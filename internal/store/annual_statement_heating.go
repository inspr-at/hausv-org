package store

import "math/big"

func IsAnnualHeatingCost(key string) bool { return key == "heizung" || key == "warmwasser" }

func AnnualStatementHeatingPools(input AnnualStatementRunInput, key string) (energy, other, consumed, area int64) {
	for _, r := range input.Receipts {
		if r.CostTypeKey != key {
			continue
		}
		if r.HeatingCategory == "energie" {
			energy += r.AmountCents
		} else {
			other += r.AmountCents
		}
	}
	// One cent rounding at house pool level, half up. The area pool is the
	// exact residual, so the two pools always conserve the receipt total.
	scaled := new(big.Int).Mul(big.NewInt(energy), big.NewInt(int64(input.Structure.Legal.HeatingConsumptionPercent)))
	scaled.Add(scaled, big.NewInt(50)).Quo(scaled, big.NewInt(100))
	consumed = scaled.Int64()
	area = energy - consumed + other
	return
}
func annualStatementHeatingCents(input AnnualStatementRunInput, cost AnnualStatementCostType, units []Unit, consumption []AnnualStatementUnitShare) ([]int64, string) {
	legal := input.Structure.Legal
	for _, u := range units {
		prepay, ok := legal.HeatingPrepayments[u.ID][cost.Key]
		if !ok || prepay < 0 {
			return nil, "heating-prepayment"
		}
		var total int64
		for key, cents := range legal.HeatingPrepayments[u.ID] {
			if !IsAnnualHeatingCost(key) || cents < 0 || cents > 9223372036854775807-total {
				return nil, "heating-prepayment"
			}
			total += cents
		}
		for _, p := range input.Prepayments {
			if p.UnitID == u.ID && total > p.AmountCents {
				return nil, "heating-prepayment"
			}
		}
	}

	if legal.HeatingConsumptionPercent < 55 || legal.HeatingConsumptionPercent > 85 {
		return nil, "heating-share"
	}
	if cost.AllocationKey != AllocationKeyVerbrauch {
		return nil, "heating-key"
	}
	areas := append([]Unit(nil), units...)
	for i := range areas {
		area, found := legal.HeatableAreas[areas[i].ID]
		if !found || area < 0 || area > 9999999 {
			return nil, "heating-area"
		}
		areas[i].UsableAreaM2Hundredths = area
		areas[i].UsableAreaRecorded = true
	}
	for _, r := range input.Receipts {
		if r.CostTypeKey == cost.Key && r.HeatingCategory != "energie" && r.HeatingCategory != "sonstige_betriebskosten" {
			return nil, "heating-category"
		}
	}
	preview := AnnualStatementAllocationPreviews([]AnnualStatementCostType{{Key: cost.Key, Allocatable: true, AllocationKey: AllocationKeyFlaeche}}, areas)
	if len(preview) != 1 || preview[0].Blocked {
		return nil, "heating-area"
	}
	_, _, consumed, pool := AnnualStatementHeatingPools(input, cost.Key)
	byConsumption := annualStatementRunCents(consumption, consumed)
	byArea := annualStatementRunCents(preview[0].Shares, pool)
	for i := range byConsumption {
		byConsumption[i] += byArea[i]
	}
	return byConsumption, ""
}
