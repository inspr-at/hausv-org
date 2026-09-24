package store

import (
	"fmt"
	"strings"
	"time"
)

const AnnualStatementVacancyLabel = "Leerstand – Eigentümeranteil"

// annualStatementVacancyApplies is the MRG vacancy rule (§ 17 Abs 1 for full
// application; Teilanwendung uses the same landlord line when a range is
// recorded). WEG leaves the result on the unit, because the owner is already
// the party. Ausnahme follows the contract and is likewise unchanged.
func annualStatementVacancyApplies(regime string) bool {
	return regime == "mrg_voll" || regime == "mrg_teil"
}

func NormalizeAnnualStatementVacancy(from, to string) (string, string, error) {
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	if from == "" && to == "" {
		return "", "", nil
	}
	start, err1 := time.Parse("2006-01-02", from)
	end, err2 := time.Parse("2006-01-02", to)
	if err1 != nil || err2 != nil || end.Before(start) {
		return "", "", fmt.Errorf("invalid vacancy range")
	}
	return start.Format("2006-01-02"), end.Format("2006-01-02"), nil
}

// annualStatementVacancyDays counts inclusive civil days. The vacancy is
// clipped to the period. Leap days count because the dates are calendar dates.
func annualStatementVacancyDays(periodStart, periodEnd, from, to string) (vacant, total int, ok bool) {
	ps, err1 := time.Parse("2006-01-02", periodStart)
	pe, err2 := time.Parse("2006-01-02", periodEnd)
	vs, err3 := time.Parse("2006-01-02", from)
	ve, err4 := time.Parse("2006-01-02", to)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || pe.Before(ps) || ve.Before(vs) {
		return 0, 0, false
	}
	total = int(pe.Sub(ps).Hours()/24) + 1
	if total <= 0 {
		return 0, 0, false
	}
	if vs.Before(ps) {
		vs = ps
	}
	if ve.After(pe) {
		ve = pe
	}
	if ve.Before(vs) {
		return 0, total, true
	}
	return int(ve.Sub(vs).Hours()/24) + 1, total, true
}

// applyAnnualStatementVacancy moves the vacant-day slice of each unit's
// already rounded cents onto a landlord line. Other units are untouched, and
// tenant cents plus landlord cents stay equal to the original unit total.
func applyAnnualStatementVacancy(result *AnnualStatementRunResult, input AnnualStatementRunInput) {
	if result == nil || !annualStatementVacancyApplies(input.Structure.Legal.Regime) {
		return
	}
	bases := map[string]AnnualStatementPeriodUnitBasis{}
	for _, basis := range input.Structure.UnitBases {
		bases[basis.UnitID] = basis
	}
	var lines []AnnualStatementVacancyLine
	for i := range result.Units {
		unit := &result.Units[i]
		basis := bases[unit.UnitID]
		if basis.VacantFrom == "" && basis.VacantTo == "" {
			continue
		}
		vacant, total, ok := annualStatementVacancyDays(input.Period.StartsOn, input.Period.EndsOn, basis.VacantFrom, basis.VacantTo)
		if !ok || vacant <= 0 || total <= 0 {
			continue
		}
		line := AnnualStatementVacancyLine{
			UnitID: unit.UnitID, Label: unit.Label, From: basis.VacantFrom, To: basis.VacantTo,
			VacantDays: vacant, PeriodDays: total,
		}
		var ownerTotal int64
		for j := range unit.Costs {
			owner := unit.Costs[j].AmountCents * int64(vacant) / int64(total)
			if owner == 0 {
				continue
			}
			unit.Costs[j].AmountCents -= owner
			ownerTotal += owner
			cost := unit.Costs[j]
			cost.AmountCents = owner
			line.Costs = append(line.Costs, cost)
		}
		if ownerTotal == 0 {
			continue
		}
		unit.AllocatedCents -= ownerTotal
		line.AmountCents = ownerTotal
		lines = append(lines, line)
	}
	result.Vacancy = lines
}
