package store

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"reflect"
	"slices"
	"testing"
	"time"
)

func TestAnnualStatementCalculationProperties(t *testing.T) {
	rng := rand.New(rand.NewSource(767))
	for _, key := range AllocationKeys {
		t.Run(key, func(t *testing.T) {
			for trial := 0; trial < 100; trial++ {
				input := AnnualStatementRunInput{Period: AnnualStatementPeriod{Year: 2024, StartsOn: "2024-01-01", EndsOn: "2024-12-31"}, Consumption: map[string]AnnualStatementConsumptionVector{}}
				var units []Unit
				n := 1 + rng.Intn(20)
				for i := 0; i < n; i++ {
					id := fmt.Sprintf("u-%02d", i)
					basis := rng.Intn(10000)
					if i == 0 && n > 1 {
						basis = 0
					}
					if i == n-1 {
						basis++
					}
					ppm := 1000000 / n
					if i == n-1 {
						ppm += 1000000 % n
					}
					unit := Unit{ID: id, Label: id, MiteigentumsanteilPPM: ppm, Persons: basis, PersonsRecorded: true, UsableAreaM2Hundredths: basis, UsableAreaRecorded: true}
					units = append(units, unit)
					input.Units = append(input.Units, AnnualStatementRunUnitIdentity{ID: id, Label: id})
					input.Structure.UnitBases = append(input.Structure.UnitBases, AnnualStatementPeriodUnitBasis{UnitID: id, MiteigentumsanteilPPM: ppm, Persons: basis, PersonsRecorded: true, UsableAreaM2Hundredths: basis, UsableAreaRecorded: true})
					input.Prepayments = append(input.Prepayments, AnnualStatementPrepayment{PeriodYear: 2024, UnitID: id, AmountCents: rng.Int63n(1000000)})
				}
				for i, cost := range []string{"heizung", "warmwasser"} {
					amount := 1 + rng.Int63n(100000000)
					if trial == 0 {
						amount = math.MaxInt64 / 2
					}
					input.Structure.CostTypes = append(input.Structure.CostTypes, AnnualStatementCostType{Key: cost, Name: cost, Allocatable: true, AllocationKey: key})
					id := fmt.Sprint(i)
					input.Receipts = append(input.Receipts, AnnualStatementReceipt{ID: id, DocumentID: id, PeriodYear: 2024, CostTypeKey: cost, AmountCents: amount, InvoiceDate: "2024-02-29"})
					input.Documents = append(input.Documents, AnnualStatementRunDocument{ID: id})
					vector := AnnualStatementConsumptionVector{PeriodYear: 2024, CostTypeKey: cost}
					for _, unit := range units {
						vector.Units = append(vector.Units, AnnualStatementUnitConsumption{UnitID: unit.ID, ValueMicros: int64(unit.Persons), MeasurementUnit: "kWh"})
					}
					input.Consumption[cost] = vector
				}
				result, issues := CalculateAnnualStatementRun(input)
				if len(issues) > 0 {
					t.Fatalf("trial %d: %+v", trial, issues)
				}
				costSums, shareSums := map[string]int64{}, map[string]int{}
				var total int64
				for i, unit := range result.Units {
					var allocated int64
					for _, cost := range unit.Costs {
						if cost.AmountCents < 0 || cost.SharePPM < 0 {
							t.Fatal("negative allocation")
						}
						if key != AllocationKeyNutzwert && units[i].Persons == 0 && (cost.SharePPM != 0 || cost.AmountCents != 0) {
							t.Fatal("zero basis charged")
						}
						costSums[cost.CostTypeKey] += cost.AmountCents
						shareSums[cost.CostTypeKey] += cost.SharePPM
						allocated += cost.AmountCents
					}
					if allocated != unit.AllocatedCents || unit.BalanceCents != allocated-input.Prepayments[i].AmountCents {
						t.Fatal("unit balance invariant failed")
					}
					total += allocated
				}
				if total != result.TotalCents {
					t.Fatal("house total differs")
				}
				for _, receipt := range input.Receipts {
					if costSums[receipt.CostTypeKey] != receipt.AmountCents || shareSums[receipt.CostTypeKey] != 1000000 {
						t.Fatalf("cost not conserved: %+v", receipt)
					}
				}
				slices.Reverse(units)
				preview, ready := AnnualStatementSettlementPreview(input.Structure.CostTypes, input.Receipts, units, input.Consumption)
				if !ready || len(preview) != len(result.Units) {
					t.Fatal("valid run has no preview")
				}
				for i := range preview {
					if preview[i].UnitID != result.Units[i].UnitID || preview[i].AllocatedCents != result.Units[i].AllocatedCents {
						t.Fatal("preview/run mismatch")
					}
				}
				slices.Reverse(input.Units)
				slices.Reverse(input.Structure.CostTypes)
				slices.Reverse(input.Receipts)
				again, issues := CalculateAnnualStatementRun(input)
				if len(issues) > 0 || !reflect.DeepEqual(result, again) {
					t.Fatal("row order changed calculation")
				}
			}
		})
	}
}

func TestAnnualStatementCentRemainders(t *testing.T) {
	for _, tc := range []struct {
		ppm    []int
		amount int64
		want   []int64
	}{
		{[]int{500000, 500000, 0}, 1, []int64{1, 0, 0}},
		{[]int{250000, 750000}, 3, []int64{1, 2}},
		{[]int{333334, 333333, 333333}, 2, []int64{1, 1, 0}},
		{[]int{0, 1000000}, math.MaxInt64, []int64{0, math.MaxInt64}},
	} {
		shares := make([]AnnualStatementUnitShare, len(tc.ppm))
		for i, ppm := range tc.ppm {
			shares[i].SharePPM = ppm
		}
		got := annualStatementRunCents(shares, tc.amount)
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%v / %d: %v != %v", tc.ppm, tc.amount, got, tc.want)
		}
		for i, cents := range got {
			// Independent arbitrary-precision floor/quota check, including MaxInt64.
			product := new(big.Int).Mul(big.NewInt(tc.amount), big.NewInt(int64(tc.ppm[i])))
			floor := new(big.Int).Quo(product, big.NewInt(1000000)).Int64()
			if cents < floor || cents-floor > 1 {
				t.Fatal("outside largest-remainder quota")
			}
		}
	}
}

func TestAnnualStatementVacancyRemainsUnitBound(t *testing.T) {
	input := annualRunFixture()
	input.Structure.UnitBases[0].Persons = 0
	input.Parties = []AnnualStatementRunParty{{UnitID: "a", ID: "owner@example.com", Owner: true}}
	result, issues := CalculateAnnualStatementRun(input)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	vacant := result.Units[0]
	if vacant.AllocatedCents != 2500 || vacant.Costs[0].AmountCents != 0 || vacant.BalanceCents != -500 {
		t.Fatalf("vacancy: %+v", vacant)
	}
	input.Parties = nil
	again, issues := CalculateAnnualStatementRun(input)
	if len(issues) > 0 || !reflect.DeepEqual(result, again) {
		t.Fatal("missing recipient shifted vacancy costs to others")
	}
}

func TestAnnualStatementConsumptionCalendarBoundaries(t *testing.T) {
	for _, tc := range []struct {
		start, end, exclusive string
		hours                 int
	}{
		{"2024-01-01", "2024-12-31", "2025-01-01", 366 * 24},
		{"2024-02-29", "2024-02-29", "2024-03-01", 24},
		{"2024-03-31", "2024-03-31", "2024-04-01", 23},
		{"2024-10-27", "2024-10-27", "2024-10-28", 25},
	} {
		t.Run(tc.start, func(t *testing.T) {
			period := AnnualStatementPeriod{Year: 2024, StartsOn: tc.start, EndsOn: tc.end}
			start, end, _, err := annualStatementConsumptionQuery(period, "heizung", []string{"a"}, mustViennaLocation(t))
			if err != nil || end.Format("2006-01-02") != tc.exclusive || end.Sub(start) != time.Duration(tc.hours)*time.Hour {
				t.Fatalf("boundaries: %v %v %v", start, end, err)
			}
			repo, _ := BindAnnualStatementConsumptionRepository(NewMemoryAnnualStatementConsumptionStore(), testTenantRef("demo"))
			appendConsumption(t, repo, consumptionEvidence("a", "heizung", "sensor.a", start, 10, "kWh"))
			appendConsumption(t, repo, consumptionEvidence("a", "heizung", "sensor.a", end, 30, "kWh"))
			report, err := repo.ConsumptionReport(period, "heizung", []string{"a"}, mustViennaLocation(t))
			if err != nil || len(report.Vector.Gaps) != 0 || len(report.Vector.Units) != 1 || report.Vector.Units[0].ValueMicros != 20 {
				t.Fatalf("calendar report: %+v %v", report, err)
			}
		})
	}
}
