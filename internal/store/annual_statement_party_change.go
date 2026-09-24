package store

import (
	"fmt"
	"math/big"
	"sort"
	"time"
)

const AnnualStatementCalculationVersionParties = 4

// PartyShare is stored, never recalculated by PDF rendering or approval.
type AnnualStatementPartyShare struct {
	UnitID              string                 `json:"unit_id"`
	PartyID             string                 `json:"party_id"`
	Unit                AnnualStatementRunUnit `json:"unit"`
	HeatingPrepayments  map[string]int64       `json:"heating_prepayments"`
	Note                string                 `json:"note"`
	DueOn               string                 `json:"due_on"`
	SettlementRecipient bool                   `json:"settlement_recipient"`
}

func hasDatedAnnualParties(input AnnualStatementRunInput) bool {
	for _, p := range input.Parties {
		if p.ValidFrom != "" || p.ValidTo != "" {
			return true
		}
	}
	return false
}
func datedAnnualUnit(input AnnualStatementRunInput, id string) bool {
	for _, p := range input.Parties {
		if p.UnitID == id && (p.ValidFrom != "" || p.ValidTo != "") {
			return true
		}
	}
	return false
}
func annualPartyStatementDate(input *AnnualStatementRunInput, at time.Time) {
	if !hasDatedAnnualParties(*input) {
		return
	}
	loc, _ := time.LoadLocation("Europe/Vienna")
	if loc != nil {
		at = at.In(loc)
	}
	input.StatementOn = at.Format("2006-01-02")
}

func AnnualStatementPartyDueOn(input AnnualStatementRunInput) string {
	at, err := time.Parse("2006-01-02", input.StatementOn)
	if err != nil {
		return ""
	}
	switch input.Structure.Legal.Regime {
	case "weg":
		return ShiftStatementDate(at, 2).Format("2006-01-02")
	case "mrg_voll":
		months := 1
		if at.Day() >= 5 {
			months = 2
		}
		return time.Date(at.Year(), at.Month()+time.Month(months), 5, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	default:
		return input.Structure.Legal.PartyDueOn
	}
}

func calculateAnnualStatementPartyRun(input AnnualStatementRunInput) (AnnualStatementRunResult, []AnnualStatementRunIssue) {
	// Dated parties supersede the old vacancy-day approximation on those units:
	// ordinary MRG costs belong to the recipient on the due date, including the
	// landlord when no tenant is present. Other units keep historical behaviour.
	basisInput := input
	basisInput.Structure.UnitBases = append([]AnnualStatementPeriodUnitBasis(nil), input.Structure.UnitBases...)
	for i := range basisInput.Structure.UnitBases {
		b := &basisInput.Structure.UnitBases[i]
		if datedAnnualUnit(input, b.UnitID) {
			b.VacantFrom, b.VacantTo = "", ""
		}
	}
	result, issues := calculateAnnualStatementRun(basisInput, true)
	if len(issues) > 0 {
		return result, issues
	}
	if input.Structure.Legal.ShowVAT {
		if code := applyAnnualStatementVAT(&result, basisInput); code != "" {
			return AnnualStatementRunResult{}, []AnnualStatementRunIssue{{Code: code}}
		}
	}
	due := AnnualStatementPartyDueOn(input)
	if _, err := time.Parse("2006-01-02", due); err != nil {
		return AnnualStatementRunResult{}, []AnnualStatementRunIssue{{Code: "party-due-date"}}
	}
	for _, unit := range result.Units {
		if !datedAnnualUnit(input, unit.UnitID) {
			continue
		}
		shares, code := annualUnitPartyShares(input, unit, due)
		if code != "" {
			return AnnualStatementRunResult{}, []AnnualStatementRunIssue{{Code: code, UnitID: unit.UnitID}}
		}
		result.PartyShares = append(result.PartyShares, shares...)
	}
	return result, nil
}

// A dated unit needs one identifiable recipient at each relevant instant.
// Overlapping joint parties are not silently assigned invented ownership shares.
func annualPartyAt(parties []AnnualStatementRunParty, on string, preferRenters bool) (int, bool) {
	for _, renter := range []bool{preferRenters, false} {
		found := -1
		for i, p := range parties {
			role := p.Owner
			if renter {
				role = p.Renter
			}
			if !role || !partyActive(p.ValidFrom, p.ValidTo, on) {
				continue
			}
			if found >= 0 {
				return -1, false
			}
			found = i
		}
		if found >= 0 {
			return found, true
		}
		if !preferRenters {
			break
		}
	}
	return -1, false
}

type annualPartySegment struct {
	from, to time.Time
	party    int
}

func annualPartySegments(input AnnualStatementRunInput, parties []AnnualStatementRunParty) ([]annualPartySegment, bool) {
	start, err := time.Parse("2006-01-02", input.Period.StartsOn)
	if err != nil {
		return nil, false
	}
	last, err := time.Parse("2006-01-02", input.Period.EndsOn)
	if err != nil {
		return nil, false
	}
	end := last.AddDate(0, 0, 1)
	bounds := []time.Time{start, end}
	for _, p := range parties {
		if p.ValidFrom != "" {
			d, _ := time.Parse("2006-01-02", p.ValidFrom)
			if d.After(start) && d.Before(end) {
				bounds = append(bounds, d)
			}
		}
		if p.ValidTo != "" {
			d, _ := time.Parse("2006-01-02", p.ValidTo)
			d = d.AddDate(0, 0, 1)
			if d.After(start) && d.Before(end) {
				bounds = append(bounds, d)
			}
		}
	}
	sort.Slice(bounds, func(i, j int) bool { return bounds[i].Before(bounds[j]) })
	var segments []annualPartySegment
	for i := 1; i < len(bounds); i++ {
		if bounds[i].Equal(bounds[i-1]) {
			continue
		}
		party, ok := annualPartyAt(parties, bounds[i-1].Format("2006-01-02"), true)
		if !ok {
			return nil, false
		}
		if len(segments) > 0 && segments[len(segments)-1].party == party {
			segments[len(segments)-1].to = bounds[i]
		} else {
			segments = append(segments, annualPartySegment{bounds[i-1], bounds[i], party})
		}
	}
	return segments, true
}

// Each calendar month has equal weight. Partial months are fractions of that
// month, so a leap February and a 31-day July do not become unequal months.
func monthlyPartyWeights(segments []annualPartySegment, n int) []*big.Rat {
	weights := make([]*big.Rat, n)
	for i := range weights {
		weights[i] = new(big.Rat)
	}
	for _, s := range segments {
		for d := s.from; d.Before(s.to); d = d.AddDate(0, 0, 1) {
			days := time.Date(d.Year(), d.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
			weights[s.party].Add(weights[s.party], big.NewRat(1, int64(days)))
		}
	}
	return weights
}
func partyCents(amount int64, weights []*big.Rat) []int64 {
	out := make([]int64, len(weights))
	total := new(big.Rat)
	for _, w := range weights {
		total.Add(total, w)
	}
	if total.Sign() == 0 {
		return out
	}
	rests := make([]*big.Rat, len(weights))
	order := make([]int, len(weights))
	var assigned int64
	for i, w := range weights {
		exact := new(big.Rat).Mul(new(big.Rat).SetInt64(amount), w)
		exact.Quo(exact, total)
		out[i] = new(big.Int).Quo(exact.Num(), exact.Denom()).Int64()
		assigned += out[i]
		rests[i] = new(big.Rat).Sub(exact, new(big.Rat).SetInt64(out[i]))
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool { return rests[order[i]].Cmp(rests[order[j]]) > 0 })
	for i := int64(0); i < amount-assigned; i++ {
		out[order[i]]++
	}
	return out
}

func interimPartyWeights(input AnnualStatementRunInput, unit, cost string, segments []annualPartySegment, n int) ([]*big.Rat, bool, string) {
	evidence := append(append([]AnnualStatementConsumptionEvidence(nil), input.Evidence...), input.PartyEvidence...)
	byDate := map[string]AnnualStatementConsumptionEvidence{}
	loc, _ := time.LoadLocation("Europe/Vienna")
	var source, measure string
	for _, e := range evidence {
		if e.UnitID != unit || e.CostTypeKey != cost {
			continue
		}
		at := e.MeasuredAt.In(loc)
		if at.Hour() != 0 || at.Minute() != 0 || at.Second() != 0 || at.Nanosecond() != 0 {
			continue
		}
		key := at.Format("2006-01-02")
		if old, ok := byDate[key]; ok && (old.ValueMicros != e.ValueMicros || old.SourceID != e.SourceID || old.SourceKind != e.SourceKind) {
			return nil, false, "party-reading"
		}
		if source != "" && (source != e.SourceKind+":"+e.SourceID || measure != e.MeasurementUnit) {
			return nil, false, "party-reading"
		}
		source, measure = e.SourceKind+":"+e.SourceID, e.MeasurementUnit
		byDate[key] = e
	}
	hasInterim := false
	for _, e := range input.PartyEvidence {
		if e.UnitID == unit && e.CostTypeKey == cost {
			hasInterim = true
		}
	}
	weights := make([]*big.Rat, n)
	for i := range weights {
		weights[i] = new(big.Rat)
	}
	for _, s := range segments {
		a, aOK := byDate[s.from.Format("2006-01-02")]
		b, bOK := byDate[s.to.Format("2006-01-02")]
		if !aOK || !bOK {
			if hasInterim {
				return nil, false, "party-reading"
			}
			return nil, false, ""
		}
		if a.ValueMicros < 0 || b.ValueMicros < a.ValueMicros {
			return nil, false, "party-reading"
		}
		weights[s.party].Add(weights[s.party], new(big.Rat).SetInt64(b.ValueMicros-a.ValueMicros))
	}
	sum := new(big.Rat)
	for _, w := range weights {
		sum.Add(sum, w)
	}
	for _, measured := range input.Consumption[cost].Units {
		if measured.UnitID == unit && sum.Cmp(new(big.Rat).SetInt64(measured.ValueMicros)) != 0 {
			return nil, false, "party-reading"
		}
	}
	if sum.Sign() == 0 {
		return nil, false, ""
	}
	return weights, true, ""
}

func annualUnitPartyShares(input AnnualStatementRunInput, unit AnnualStatementRunUnit, due string) ([]AnnualStatementPartyShare, string) {
	var parties []AnnualStatementRunParty
	for _, p := range input.Parties {
		if p.UnitID == unit.UnitID {
			parties = append(parties, p)
		}
	}
	sort.Slice(parties, func(i, j int) bool { return parties[i].ID < parties[j].ID })
	for i, p := range parties {
		if p.ID == "" || i > 0 && parties[i-1].ID == p.ID || ValidateUnitPartyContacts([]UnitPartyContact{{ValidFrom: p.ValidFrom, ValidTo: p.ValidTo}}) != nil {
			return nil, "party-dates"
		}
	}
	recipient, ok := annualPartyAt(parties, due, input.Structure.Legal.Regime != "weg")
	if !ok {
		return nil, "party-recipient"
	}
	shares := make([]AnnualStatementPartyShare, len(parties))
	note := fmt.Sprintf("Parteienwechsel: Betriebskostensaldo vollständig an die Partei am Fälligkeitstag %s; keine zeitanteilige Aufteilung.", due)
	if input.Structure.Legal.Regime == "weg" {
		note += " § 34 Abs. 4 WEG."
	} else if input.Structure.Legal.Regime == "mrg_voll" {
		note += " Raumschuldnerprinzip (§ 21 MRG)."
	} else {
		note += " Fälligkeit laut Vertrag."
	}
	if !parties[recipient].Renter && input.Structure.Legal.Regime != "weg" {
		note += " Kein Mieter am Fälligkeitstag: Eigentümeranteil."
	}
	for i, p := range parties {
		shares[i] = AnnualStatementPartyShare{UnitID: unit.UnitID, PartyID: p.ID, Unit: AnnualStatementRunUnit{UnitID: unit.UnitID, Label: unit.Label}, HeatingPrepayments: map[string]int64{}, Note: note, DueOn: due, SettlementRecipient: i == recipient}
	}
	var monthly []*big.Rat
	var segments []annualPartySegment
	if input.Structure.Legal.HeizKGApplies {
		segments, ok = annualPartySegments(input, parties)
		if !ok {
			return nil, "party-coverage"
		}
		monthly = monthlyPartyWeights(segments, len(parties))
	}
	var heatPrepaid int64
	for _, cost := range unit.Costs {
		amounts := make([]int64, len(parties))
		amounts[recipient] = cost.AmountCents
		if input.Structure.Legal.HeizKGApplies && IsAnnualHeatingCost(cost.CostTypeKey) {
			weights, measured, code := interimPartyWeights(input, unit.UnitID, cost.CostTypeKey, segments, len(parties))
			if code != "" {
				return nil, code
			}
			amounts = partyCents(cost.AmountCents, monthly)
			method := "gleiche monatliche Anteile"
			if measured {
				// Preserve the already rounded unit consumption pool; only that pool uses
				// meter deltas. Fixed/area costs continue by equal monthly shares.
				consumed := annualUnitConsumptionCents(input, unit.UnitID, cost.CostTypeKey)
				amounts = partyCents(consumed, weights)
				fixed := partyCents(cost.AmountCents-consumed, monthly)
				for i := range amounts {
					amounts[i] += fixed[i]
				}
				method = "Zwischenablesung (Verbrauch); gleiche monatliche Anteile (Fläche)"
			}
			paid := input.Structure.Legal.HeatingPrepayments[unit.UnitID][cost.CostTypeKey]
			heatPrepaid += paid
			payments := partyCents(paid, monthly)
			for i := range shares {
				shares[i].HeatingPrepayments[cost.CostTypeKey] = payments[i]
				shares[i].Unit.PrepaidCents += payments[i]
				shares[i].Note += " " + cost.Name + ": " + method + " (§ 23 HeizKG); erfasstes Einheiten-Akonto nach Monatsanteilen."
			}
		}
		// Split the unit's VAT by the resulting gross cents. This preserves both
		// the house VAT cent and every recipient's gross = net + VAT identity.
		vats := distributeLargestRemainder(cost.VATCents, amounts)
		for i := range shares {
			c := cost
			c.AmountCents = amounts[i]
			c.VATCents = vats[i]
			c.NetCents = c.AmountCents - c.VATCents
			shares[i].Unit.Costs = append(shares[i].Unit.Costs, c)
			shares[i].Unit.AllocatedCents += c.AmountCents
		}
	}
	shares[recipient].Unit.PrepaidCents += unit.PrepaidCents - heatPrepaid
	for i := range shares {
		u := &shares[i].Unit
		u.BalanceCents = u.AllocatedCents - u.PrepaidCents
		if input.Structure.Legal.ShowVAT {
			u.VAT = annualStatementVATGroups(u.Costs)
		}
	}
	return shares, ""
}

func annualUnitConsumptionCents(input AnnualStatementRunInput, id, key string) int64 {
	var units []Unit
	for _, u := range input.Units {
		units = append(units, Unit{ID: u.ID})
	}
	sort.Slice(units, func(i, j int) bool { return units[i].ID < units[j].ID })
	shares, ok := AnnualStatementConsumptionShares(input.Consumption[key], units)
	if !ok {
		return 0
	}
	_, _, pool, _ := AnnualStatementHeatingPools(input, key)
	cents := annualStatementRunCents(shares, pool)
	for i, s := range shares {
		if s.UnitID == id {
			return cents[i]
		}
	}
	return 0
}

func AnnualStatementPartyAllocation(run AnnualStatementRun, unit, party string) (AnnualStatementPartyShare, bool) {
	for _, s := range run.Result.PartyShares {
		if s.UnitID == unit && s.PartyID == party {
			return s, true
		}
	}
	return AnnualStatementPartyShare{}, false
}

func annualStatementVATGroups(costs []AnnualStatementRunCost) []AnnualStatementVATGroup {
	byRate := map[int]AnnualStatementVATGroup{}
	var rates []int
	for _, c := range costs {
		g, ok := byRate[c.VATRatePercent]
		if !ok {
			rates = append(rates, c.VATRatePercent)
		}
		g.RatePercent = c.VATRatePercent
		g.NetCents += c.NetCents
		g.VATCents += c.VATCents
		g.GrossCents += c.AmountCents
		byRate[c.VATRatePercent] = g
	}
	sort.Ints(rates)
	var out []AnnualStatementVATGroup
	for _, r := range rates {
		out = append(out, byRate[r])
	}
	return out
}

// Only meter facts at actual party-change boundaries are copied. Annual
// boundary evidence remains in Evidence, so frequent sampling does not inflate
// immutable runs. A missing intermediate fact uses the monthly fallback.
func annualPartyReadingDates(input AnnualStatementRunInput) map[string][]time.Time {
	out := map[string][]time.Time{}
	loc, _ := time.LoadLocation("Europe/Vienna")
	seen := map[string]bool{}
	for _, p := range input.Parties {
		for i, raw := range []string{p.ValidFrom, p.ValidTo} {
			if raw == "" {
				continue
			}
			d, err := time.ParseInLocation("2006-01-02", raw, loc)
			if err != nil {
				continue
			}
			if i == 1 {
				d = d.AddDate(0, 0, 1)
			}
			key := d.Format("2006-01-02")
			if key <= input.Period.StartsOn || key > input.Period.EndsOn || seen[p.UnitID+key] {
				continue
			}
			seen[p.UnitID+key] = true
			out[p.UnitID] = append(out[p.UnitID], d)
		}
	}
	for id := range out {
		sort.Slice(out[id], func(i, j int) bool { return out[id][i].Before(out[id][j]) })
	}
	return out
}
