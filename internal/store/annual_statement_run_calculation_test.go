package store

import (
	"math"
	"reflect"
	"slices"
	"testing"
)

func annualRunFixture() AnnualStatementRunInput {
	return AnnualStatementRunInput{
		Period: AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31"},
		Structure: AnnualStatementPeriodStructure{
			CostTypes: []AnnualStatementCostType{{Key: "tax", Name: "Abgabe", Allocatable: true, AllocationKey: AllocationKeyNutzwert}, {Key: "service", Name: "Betreuung", Allocatable: true, AllocationKey: AllocationKeyPersonen}, {Key: "excluded", Name: "Nicht umlagefähig"}},
			UnitBases: []AnnualStatementPeriodUnitBasis{{UnitID: "a", MiteigentumsanteilPPM: 250000, Persons: 1, PersonsRecorded: true}, {UnitID: "b", MiteigentumsanteilPPM: 750000, Persons: 1, PersonsRecorded: true}}},
		Units:       []AnnualStatementRunUnitIdentity{{"a", "Top 1"}, {"b", "Top 2"}},
		Receipts:    []AnnualStatementReceipt{{ID: "r1", DocumentID: "d1", PeriodYear: 2025, CostTypeKey: "tax", AmountCents: 10001, InvoiceDate: "2025-02-01"}, {ID: "r2", DocumentID: "d2", PeriodYear: 2025, CostTypeKey: "service", AmountCents: 101, InvoiceDate: "2025-02-01"}, {ID: "r3", DocumentID: "d3", PeriodYear: 2025, CostTypeKey: "excluded", AmountCents: 999, InvoiceDate: "2025-02-01"}},
		Documents:   []AnnualStatementRunDocument{{ID: "d1"}, {ID: "d2"}, {ID: "d3"}},
		Prepayments: []AnnualStatementPrepayment{{PeriodYear: 2025, UnitID: "a", AmountCents: 3000}, {PeriodYear: 2025, UnitID: "b", AmountCents: 0}},
	}
}

func TestAnnualStatementRunCalculation(t *testing.T) {
	in := annualRunFixture()
	result, issues := CalculateAnnualStatementRun(in)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	if result.TotalCents != 10102 || result.ExcludedCents != 999 || len(result.Units) != 2 {
		t.Fatalf("%+v", result)
	}
	if result.Units[0].AllocatedCents != 2551 || result.Units[0].BalanceCents != -449 || result.Units[1].AllocatedCents != 7551 || result.Units[1].BalanceCents != 7551 {
		t.Fatalf("units=%+v", result.Units)
	}
	slices.Reverse(in.Units)
	slices.Reverse(in.Structure.CostTypes)
	slices.Reverse(in.Receipts)
	again, issues := CalculateAnnualStatementRun(in)
	if len(issues) > 0 || !reflect.DeepEqual(result, again) {
		t.Fatalf("unstable result: %+v %v", again, issues)
	}
}

func TestAnnualStatementRunMissingInputsBlockAllResults(t *testing.T) {
	for name, mutate := range map[string]func(*AnnualStatementRunInput){
		"period":       func(in *AnnualStatementRunInput) { in.Period = AnnualStatementPeriod{} },
		"units":        func(in *AnnualStatementRunInput) { in.Units = nil },
		"deleted unit": func(in *AnnualStatementRunInput) { in.Units = in.Units[:1] },
		"new unit": func(in *AnnualStatementRunInput) {
			in.Units = append(in.Units, AnnualStatementRunUnitIdentity{"c", "Top 3"})
		},
		"duplicate unit": func(in *AnnualStatementRunInput) { in.Units[1].ID = "a" },
		"structure":      func(in *AnnualStatementRunInput) { in.Structure.UnitBases = nil },
		"cost types":     func(in *AnnualStatementRunInput) { in.Structure.CostTypes = nil },
		"key":            func(in *AnnualStatementRunInput) { in.Structure.CostTypes[0].AllocationKey = "" },
		"unknown key":    func(in *AnnualStatementRunInput) { in.Structure.CostTypes[0].AllocationKey = "made-up" },
		"basis":          func(in *AnnualStatementRunInput) { in.Structure.UnitBases[0].PersonsRecorded = false },
		"zero total": func(in *AnnualStatementRunInput) {
			in.Structure.UnitBases[0].Persons = 0
			in.Structure.UnitBases[1].Persons = 0
		},
		"nutzwert total":         func(in *AnnualStatementRunInput) { in.Structure.UnitBases[0].MiteigentumsanteilPPM = 1 },
		"receipts":               func(in *AnnualStatementRunInput) { in.Receipts = nil },
		"missing cost receipt":   func(in *AnnualStatementRunInput) { in.Receipts = in.Receipts[1:] },
		"document":               func(in *AnnualStatementRunInput) { in.Documents = in.Documents[1:] },
		"duplicate receipt":      func(in *AnnualStatementRunInput) { in.Receipts = append(in.Receipts, in.Receipts[0]) },
		"foreign receipt period": func(in *AnnualStatementRunInput) { in.Receipts[0].PeriodYear = 2024 },
		"unknown cost":           func(in *AnnualStatementRunInput) { in.Receipts[0].CostTypeKey = "missing" },
		"zero receipt":           func(in *AnnualStatementRunInput) { in.Receipts[0].AmountCents = 0 },
		"negative receipt":       func(in *AnnualStatementRunInput) { in.Receipts[0].AmountCents = -1 },
		"prepayment":             func(in *AnnualStatementRunInput) { in.Prepayments = in.Prepayments[:1] },
		"foreign prepayment":     func(in *AnnualStatementRunInput) { in.Prepayments[0].PeriodYear = 2024 },
		"negative prepayment":    func(in *AnnualStatementRunInput) { in.Prepayments[0].AmountCents = -1 },
		"overflow":               func(in *AnnualStatementRunInput) { in.Receipts[0].AmountCents = math.MaxInt64 },
		"basis overflow":         func(in *AnnualStatementRunInput) { in.Structure.UnitBases[0].Persons = math.MaxInt },
		"measurement rule":       func(in *AnnualStatementRunInput) { in.Structure.CostTypes[0].AllocationKey = AllocationKeyVerbrauch },
	} {
		t.Run(name, func(t *testing.T) {
			in := annualRunFixture()
			mutate(&in)
			result, issues := CalculateAnnualStatementRun(in)
			if len(issues) == 0 || !reflect.DeepEqual(result, AnnualStatementRunResult{}) {
				t.Fatalf("partial result=%+v issues=%+v", result, issues)
			}
		})
	}
}

func TestAnnualStatementRunCentsPreserveMaximum(t *testing.T) {
	shares := []AnnualStatementUnitShare{{SharePPM: 333334}, {SharePPM: 333333}, {SharePPM: 333333}}
	for _, amount := range []int64{1, 2, 10001, math.MaxInt64} {
		cents := annualStatementRunCents(shares, amount)
		var sum int64
		for _, value := range cents {
			if value < 0 {
				t.Fatal("negative allocation")
			}
			sum += value
		}
		if sum != amount {
			t.Fatalf("sum=%d want=%d", sum, amount)
		}
	}
}

func TestAnnualStatementRunCentsIncompleteSharesDoNotPanic(t *testing.T) {
	if got := annualStatementRunCents(nil, 1); len(got) != 0 {
		t.Fatalf("empty shares=%v", got)
	}
	if got := annualStatementRunCents([]AnnualStatementUnitShare{{SharePPM: 1}}, 100); len(got) != 1 {
		t.Fatalf("incomplete shares=%v", got)
	}
}
