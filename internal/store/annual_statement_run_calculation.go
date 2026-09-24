package store

import (
	"math"
	"sort"
	"time"
)

const AnnualStatementCalculationVersion = 2

// AnnualStatementRunInput is an immutable copy of the facts used by a run.
// Documents contains only originals verified as readable by the repository.
type AnnualStatementRunInput struct {
	Presentation AnnualStatementRunPresentation              `json:"presentation"`
	Parties      []AnnualStatementRunParty                   `json:"parties,omitempty"`
	Period       AnnualStatementPeriod                       `json:"period"`
	Structure    AnnualStatementPeriodStructure              `json:"structure"`
	Units        []AnnualStatementRunUnitIdentity            `json:"units"`
	Receipts     []AnnualStatementReceipt                    `json:"receipts"`
	Prepayments  []AnnualStatementPrepayment                 `json:"prepayments"`
	Documents    []AnnualStatementRunDocument                `json:"documents"`
	Consumption  map[string]AnnualStatementConsumptionVector `json:"consumption"`
	Evidence     []AnnualStatementConsumptionEvidence        `json:"evidence"`
}

type AnnualStatementRunUnitIdentity struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// UnitType is the normalised register type at run time (HAUSV-639). It only
	// orders the presentation: runs stored before it existed carry none.
	UnitType string `json:"unit_type,omitempty"`
}

type AnnualStatementRunDocument struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Filename string `json:"filename"`
}

type AnnualStatementRunIssue struct {
	Code        string
	CostTypeKey string
	UnitID      string
}

type AnnualStatementRunBlockedError struct{ Issues []AnnualStatementRunIssue }

func (e *AnnualStatementRunBlockedError) Error() string { return "annual statement run blocked" }

type AnnualStatementRunCost struct {
	CostTypeKey   string `json:"cost_type_key"`
	Name          string `json:"name"`
	AllocationKey string `json:"allocation_key"`
	SharePPM      int    `json:"share_ppm"`
	AmountCents   int64  `json:"amount_cents"`
}

type AnnualStatementRunUnit struct {
	UnitID         string                   `json:"unit_id"`
	Label          string                   `json:"label"`
	Costs          []AnnualStatementRunCost `json:"costs"`
	AllocatedCents int64                    `json:"allocated_cents"`
	PrepaidCents   int64                    `json:"prepaid_cents"`
	// Positive means an amount due; negative means a credit.
	BalanceCents int64 `json:"balance_cents"`
}

type AnnualStatementRunResult struct {
	Proposals     []AnnualStatementPrepaymentProposal `json:"proposals,omitempty"`
	TotalCents    int64                               `json:"total_cents"`
	ExcludedCents int64                               `json:"excluded_cents"`
	Units         []AnnualStatementRunUnit            `json:"units"`
}

// CalculateAnnualStatementRun never returns partial monetary results.
func CalculateAnnualStatementRun(input AnnualStatementRunInput) (AnnualStatementRunResult, []AnnualStatementRunIssue) {
	return calculateAnnualStatementRun(input, true)
}

// Replay dispatches by the stored algorithm version; version 1 retains its
// original single-key heating allocation even when current settings differ.
func ReplayAnnualStatementRun(run AnnualStatementRun) (AnnualStatementRunResult, []AnnualStatementRunIssue) {
	switch run.CalculationVersion {
	case 1:
		return calculateAnnualStatementRun(run.Input, false)
	case 2:
		return calculateAnnualStatementRun(run.Input, true)
	default:
		return AnnualStatementRunResult{}, []AnnualStatementRunIssue{{Code: "calculation-version"}}
	}
}
func calculateAnnualStatementRun(input AnnualStatementRunInput, heatingSplit bool) (AnnualStatementRunResult, []AnnualStatementRunIssue) {
	var issues []AnnualStatementRunIssue
	issue := func(code, cost, unit string) { issues = append(issues, AnnualStatementRunIssue{code, cost, unit}) }
	if validateAnnualStatementPeriod(input.Period) != nil {
		issue("period", "", "")
	}
	if len(input.Units) == 0 {
		issue("units", "", "")
	}
	units := make([]Unit, 0, len(input.Units))
	unitIDs := map[string]bool{}
	for _, item := range input.Units {
		id := NormalizeUnitID(item.ID)
		if id == "" || id != item.ID || unitIDs[id] {
			issue("units", "", item.ID)
		}
		unitIDs[id] = true
		units = append(units, Unit{ID: id, Label: item.Label})
	}
	sort.Slice(units, func(i, j int) bool { return units[i].ID < units[j].ID })
	bases := map[string]AnnualStatementPeriodUnitBasis{}
	for _, basis := range input.Structure.UnitBases {
		if _, duplicate := bases[basis.UnitID]; duplicate || !unitIDs[basis.UnitID] {
			issue("structure", "", basis.UnitID)
		}
		bases[basis.UnitID] = basis
	}
	for i := range units {
		basis, found := bases[units[i].ID]
		if !found {
			issue("structure", "", units[i].ID)
		}
		units[i].MiteigentumsanteilPPM = basis.MiteigentumsanteilPPM
		units[i].UsableAreaM2Hundredths = basis.UsableAreaM2Hundredths
		units[i].UsableAreaRecorded = basis.UsableAreaRecorded
		units[i].Persons = basis.Persons
		units[i].PersonsRecorded = basis.PersonsRecorded
	}
	costs := append([]AnnualStatementCostType(nil), input.Structure.CostTypes...)
	sort.Slice(costs, func(i, j int) bool { return costs[i].Key < costs[j].Key })
	byCost := map[string]AnnualStatementCostType{}
	if len(costs) == 0 {
		issue("cost-types", "", "")
	}
	for _, cost := range costs {
		if _, duplicate := byCost[cost.Key]; duplicate || !annualStatementCostTypeKeyPattern.MatchString(cost.Key) {
			issue("cost-types", cost.Key, "")
		}
		byCost[cost.Key] = cost
		if cost.Allocatable && !ValidAllocationKey(cost.AllocationKey) {
			issue("key", cost.Key, "")
		}
	}
	documents := map[string]bool{}
	for _, doc := range input.Documents {
		documents[doc.ID] = true
	}
	totals := map[string]int64{}
	counts := map[string]int{}
	seenReceipts, seenDocuments := map[string]bool{}, map[string]bool{}
	result := AnnualStatementRunResult{}
	for _, receipt := range input.Receipts {
		cost, known := byCost[receipt.CostTypeKey]
		_, dateErr := time.Parse("2006-01-02", receipt.InvoiceDate)
		if !known || receipt.PeriodYear != input.Period.Year || receipt.ID == "" || seenReceipts[receipt.ID] || seenDocuments[receipt.DocumentID] || receipt.AmountCents <= 0 || dateErr != nil {
			issue("receipt", receipt.CostTypeKey, "")
			continue
		}
		seenReceipts[receipt.ID], seenDocuments[receipt.DocumentID] = true, true
		if receipt.DocumentID == "" || !documents[receipt.DocumentID] {
			issue("document", receipt.CostTypeKey, "")
		}
		counts[cost.Key]++
		if totals[cost.Key] > math.MaxInt64-receipt.AmountCents {
			issue("overflow", cost.Key, "")
			continue
		}
		totals[cost.Key] += receipt.AmountCents
		if cost.Allocatable {
			if result.TotalCents > math.MaxInt64-receipt.AmountCents {
				issue("overflow", cost.Key, "")
			} else {
				result.TotalCents += receipt.AmountCents
			}
		} else {
			if result.ExcludedCents > math.MaxInt64-receipt.AmountCents {
				issue("overflow", cost.Key, "")
			} else {
				result.ExcludedCents += receipt.AmountCents
			}
		}
	}
	prepaid := map[string]int64{}
	for _, item := range input.Prepayments {
		_, duplicate := prepaid[item.UnitID]
		if duplicate || !unitIDs[item.UnitID] || item.PeriodYear != input.Period.Year || item.AmountCents < 0 {
			issue("prepayment", "", item.UnitID)
		}
		prepaid[item.UnitID] = item.AmountCents
	}
	for _, unit := range units {
		if _, found := prepaid[unit.ID]; !found {
			issue("prepayment", "", unit.ID)
		}
	}
	sharesByCost := map[string][]AnnualStatementUnitShare{}
	for _, cost := range costs {
		if !cost.Allocatable {
			continue
		}
		if counts[cost.Key] == 0 {
			issue("missing-receipt", cost.Key, "")
		}
		if !ValidAllocationKey(cost.AllocationKey) {
			continue
		}
		if cost.AllocationKey == AllocationKeyVerbrauch {
			if cost.Key != "heizung" && cost.Key != "warmwasser" {
				issue("measurement-rule", cost.Key, "")
				continue
			}
			// Use the validated report vector stored in the snapshot. Boundary
			// evidence alone cannot reveal a reset inside the period.
			vector, found := input.Consumption[cost.Key]
			if !found || vector.PeriodYear != input.Period.Year || vector.CostTypeKey != cost.Key {
				issue("consumption", cost.Key, "")
				continue
			}
			shares, complete := AnnualStatementConsumptionShares(vector, units)
			if !complete {
				issue("consumption", cost.Key, "")
			} else {
				sharesByCost[cost.Key] = shares
			}
		} else {
			previews := AnnualStatementAllocationPreviews([]AnnualStatementCostType{cost}, units)
			if len(previews) != 1 || previews[0].Blocked {
				issue("basis", cost.Key, "")
				continue
			}
			if cost.AllocationKey == AllocationKeyNutzwert && previews[0].BasisTotal != MiteigentumsanteilTotalPPM {
				issue("nutzwert-total", cost.Key, "")
				continue
			}
			sharesByCost[cost.Key] = previews[0].Shares
		}
	}
	if len(issues) > 0 {
		return AnnualStatementRunResult{}, issues
	}
	for _, unit := range units {
		result.Units = append(result.Units, AnnualStatementRunUnit{UnitID: unit.ID, Label: unit.Label, PrepaidCents: prepaid[unit.ID]})
	}
	for _, cost := range costs {
		if !cost.Allocatable {
			continue
		}
		shares := sharesByCost[cost.Key]
		cents := annualStatementRunCents(shares, totals[cost.Key])
		if heatingSplit && input.Structure.Legal.HeizKGApplies && IsAnnualHeatingCost(cost.Key) {
			var code string
			cents, code = annualStatementHeatingCents(input, cost, units, shares)
			if code != "" {
				return AnnualStatementRunResult{}, []AnnualStatementRunIssue{{Code: code, CostTypeKey: cost.Key}}
			}
		}

		for i, share := range shares {
			result.Units[i].Costs = append(result.Units[i].Costs, AnnualStatementRunCost{cost.Key, cost.Name, cost.AllocationKey, share.SharePPM, cents[i]})
			result.Units[i].AllocatedCents += cents[i]
		}
	}
	for i := range result.Units {
		result.Units[i].BalanceCents = result.Units[i].AllocatedCents - result.Units[i].PrepaidCents
	}
	if heatingSplit {
		result.Proposals = AnnualStatementPrepaymentProposals(input, result)
	}
	return result, nil
}

func annualStatementRunCents(shares []AnnualStatementUnitShare, amount int64) []int64 {
	type remainder struct {
		index int
		value int64
	}
	out := make([]int64, len(shares))
	rests := make([]remainder, len(shares))
	var assigned int64
	for i, share := range shares {
		ppm := int64(share.SharePPM)
		fraction := (amount % 1_000_000) * ppm
		out[i] = (amount/1_000_000)*ppm + fraction/1_000_000
		assigned += out[i]
		rests[i] = remainder{i, fraction % 1_000_000}
	}
	sort.SliceStable(rests, func(i, j int) bool { return rests[i].value > rests[j].value })
	for i := int64(0); i < amount-assigned && i < int64(len(rests)); i++ {
		out[rests[i].index]++
	}
	return out
}
