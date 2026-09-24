package indexation

import (
	"math"
	"math/big"
	"testing"
	"time"
)

func date(s string) time.Time { d, _ := time.Parse(time.DateOnly, s); return d }

func TestAnnualCeiling(t *testing.T) {
	for _, tc := range []struct {
		name       string
		year       int
		current    string
		restricted bool
		months     int
		base, want int64
	}{
		{"half above 3", 2026, "106", false, 12, 100000, 104500},
		{"exactly 3", 2026, "103", false, 12, 100000, 103000},
		{"1 percent cap", 2026, "106", true, 12, 100000, 101000},
		{"2 percent cap", 2027, "106", true, 12, 100000, 102000},
		{"no special cap 2028", 2028, "106", true, 12, 100000, 104500},
		{"deflation", 2026, "98", true, 12, 100000, 98000},
		{"exact half cent rounds down", 2026, "103", false, 12, 150, 154},
		{"E3 new February lease", 2027, "103.4", false, 10, 120000, 123200},
		{"E3 new February full MRG", 2027, "103.4", true, 10, 120000, 122000},
		{"cap before proration", 2026, "106", true, 6, 100000, 100500},
		{"December contributes zero", 2027, "106", true, 0, 100000, 100000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := AnnualCeiling(AnnualModel{Year: tc.year, BaseAmountCents: tc.base, PreviousAverage: 100 * Unit, CurrentAverage: dec(t, tc.current), FullMonths: tc.months, PriceControlled: tc.restricted})
			if err != nil || got.AmountCents != tc.want {
				t.Fatalf("got %+v %v, want %d", got, err, tc.want)
			}
		})
	}
	if _, err := AnnualCeiling(AnnualModel{Year: 2026, BaseAmountCents: math.MaxInt64, PreviousAverage: Unit, CurrentAverage: 2 * Unit, FullMonths: 12}); err == nil {
		t.Fatal("ceiling overflow accepted")
	}
	for _, months := range []int{-1, 13} {
		if _, err := AnnualCeiling(AnnualModel{Year: 2026, BaseAmountCents: 100, PreviousAverage: Unit, CurrentAverage: Unit, FullMonths: months}); err == nil {
			t.Fatal("bad proration accepted")
		}
	}
}

func TestLawReportParallelCurves(t *testing.T) {
	s, err := LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	clause := Clause{Series: VPI2020, BaseMonth: "2024-09", BaseValue: dec(t, "123.6"), AmountCents: 100000, ThresholdKind: PercentThreshold, Threshold: 5 * Unit, PercentRounding: UnroundedPercent}
	contract, err := Evaluate(clause, s.Data, "2025-12")
	if err != nil || contract.TriggerMonth != "2025-12" || contract.NewAmountCents != 105016 {
		t.Fatalf("E1 contract: %+v %v", contract, err)
	}
	for _, tc := range []struct {
		scope      RentalScope
		restricted bool
		want       int64
	}{
		{ResidentialPartialMRG, false, 104028},
		{ResidentialFullMRG, true, 101735},
	} {
		ceiling, err := CapCurve("2024-09", 100000, s.Annual, tc.restricted, 2026)
		if err != nil || ceiling.AmountCents != tc.want {
			t.Fatalf("E1/E2 cap: %+v %v", ceiling, err)
		}
		if len(ceiling.Years) != 2 || ceiling.Years[0].FullMonths != 3 {
			t.Fatalf("lost first-year proration: %+v", ceiling)
		}
		context := StatutoryContext{Scope: tc.scope, CurrentAmountCents: 100000, ContractEffectiveOn: date("2026-02-01"), Ceiling: ceiling}
		before, err := ApplyMieWeG(contract, context, date("2026-03-31"))
		if err != nil || before.AllowedNowCents != 100000 || before.DeferredCents != tc.want-100000 || before.CappedCents != 105016-tc.want {
			t.Fatalf("deferred E1/E2: %+v %v", before, err)
		}
		for _, asOf := range []string{"2026-04-01", "2026-09-24"} {
			got, err := ApplyMieWeG(contract, context, date(asOf))
			if err != nil || got.AllowedNowCents != tc.want || got.DeferredCents != 0 || got.EffectiveOn != date("2026-04-01") {
				t.Fatalf("E1/E2 on %s: %+v %v", asOf, got, err)
			}
		}
		// Model stays based on the independent anchor, even after prescription.
		again, err := CapCurve("2024-09", 100000, s.Annual, tc.restricted, 2026)
		if err != nil || again.ExactAmountCents != ceiling.ExactAmountCents {
			t.Fatal("non-deterministic curve")
		}
	}
	// The default rounded-% clause intentionally does not cross at 5.016%:
	// 5.0% is still within the exclusive 5% threshold. Clause wording matters.
	clause.PercentRounding = OneDecimalPercent
	rounded, err := Evaluate(clause, s.Data, "2025-12")
	if err != nil || rounded.Crossed {
		t.Fatalf("rounding mode ignored: %+v %v", rounded, err)
	}
}

func TestCapCurveCarriesExactIntermediateAmounts(t *testing.T) {
	averages := []AnnualValue{{Series: VPI2020, Year: 2023, Value: 100 * Unit}, {Series: VPI2020, Year: 2024, Value: 103 * Unit}, {Series: VPI2020, Year: 2025, Value: dec(t, "106.09")}}
	// €1.50 * 1.03 * 1.03 = €1.59135 -> €1.59; yearly cent rounding
	// would instead yield €1.54 * 1.03 = €1.5862 -> €1.59 (check exact state too).
	got, err := CapCurve("2023-12", 150, averages, false, 2026)
	if err != nil || got.ExactAmountCents != "31827/200" {
		t.Fatalf("exact carry lost: %+v %v", got, err)
	}
	for _, mutate := range []func([]AnnualValue) []AnnualValue{
		func(a []AnnualValue) []AnnualValue { return a[:2] },
		func(a []AnnualValue) []AnnualValue { a[2].Preliminary = true; return a },
		func(a []AnnualValue) []AnnualValue { return append(a, a[2]) },
		func(a []AnnualValue) []AnnualValue { a[1].Value = 0; return a },
	} {
		bad := mutate(append([]AnnualValue(nil), averages...))
		if _, err := CapCurve("2023-12", 150, bad, false, 2026); err == nil {
			t.Fatal("bad annual data accepted")
		}
	}
	for _, tc := range []struct {
		s    string
		want int64
	}{{"101734.5", 101734}, {"101734.51", 101735}, {"101735.5", 101735}} {
		r, _ := new(big.Rat).SetString(tc.s)
		got, err := roundRat(r, false)
		if err != nil || got != tc.want {
			t.Fatalf("half-down %s: %d %v", tc.s, got, err)
		}
	}
}

func TestStatutoryExemptionsAndTiming(t *testing.T) {
	contract := Evaluation{Crossed: true, NewAmountCents: 110000}
	ceiling, _ := AnnualCeiling(AnnualModel{Year: 2026, BaseAmountCents: 100000, PreviousAverage: 100 * Unit, CurrentAverage: 106 * Unit, FullMonths: 12, PriceControlled: true})
	for _, scope := range []RentalScope{Commercial, CommercialFullMRG, ResidentialExcluded} {
		got, err := ApplyMieWeG(contract, StatutoryContext{Scope: scope, CurrentAmountCents: 100000, ContractEffectiveOn: date("2026-02-01")}, date("2026-02-01"))
		if err != nil || got.AllowedNowCents != 110000 || got.CappedCents != 0 || got.RequiresMRGNotice != (scope == CommercialFullMRG) {
			t.Fatalf("exemption: %+v %v", got, err)
		}
	}
	context := StatutoryContext{Scope: ResidentialFullMRG, CurrentAmountCents: 100000, ContractEffectiveOn: date("2026-05-01"), Ceiling: ceiling}
	if _, err := ApplyMieWeG(contract, context, date("2026-05-01")); err == nil {
		t.Fatal("May increase moved backwards into April")
	}
	context.Ceiling, _ = AnnualCeiling(AnnualModel{Year: 2027, BaseAmountCents: 101000, PreviousAverage: 100 * Unit, CurrentAverage: 106 * Unit, FullMonths: 12, PriceControlled: true})
	got, err := ApplyMieWeG(contract, context, date("2026-09-01"))
	if err != nil || got.AllowedNowCents != 100000 || got.PermittedCents != 103020 || got.DeferredCents != 3020 {
		t.Fatalf("next April: %+v %v", got, err)
	}
	contract.NewAmountCents = 94000
	got, err = ApplyMieWeG(contract, context, date("2026-05-01"))
	if err != nil || got.AllowedNowCents != 94000 {
		t.Fatalf("decrease delayed: %+v %v", got, err)
	}
	contract.Crossed = false
	got, err = ApplyMieWeG(contract, context, date("2027-04-01"))
	if err != nil || got.AllowedNowCents != 100000 {
		t.Fatalf("no clause jump manufactured: %+v %v", got, err)
	}
	for _, scope := range []RentalScope{"", "wgg", "unknown"} {
		context.Scope = scope
		if _, err := ApplyMieWeG(contract, context, date("2027-04-01")); err == nil {
			t.Fatal("unknown legal scope accepted")
		}
	}
}

func TestNewContractProration(t *testing.T) {
	for _, tc := range []struct {
		month Month
		want  int
	}{{"2026-02", 10}, {"2026-06", 6}, {"2026-12", 0}} {
		got, err := FirstAdjustmentMonths(tc.month, 2027)
		if err != nil || got != tc.want {
			t.Fatalf("proration %s: %d %v", tc.month, got, err)
		}
	}
	if _, err := FirstAdjustmentMonths("2026-06", 2026); err == nil {
		t.Fatal("new lease indexed in same year")
	}
}
