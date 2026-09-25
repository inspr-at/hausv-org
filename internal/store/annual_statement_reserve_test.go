package store

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestReserveBalanceConservesClosingAcrossMEAShares(t *testing.T) {
	units := []Unit{
		{ID: "a", MiteigentumsanteilPPM: 200_000},
		{ID: "b", MiteigentumsanteilPPM: 300_000},
		{ID: "c", MiteigentumsanteilPPM: 500_000},
	}
	for _, unit := range units {
		unit.UsableAreaRecorded = true
	}
	units[0].UsableAreaM2Hundredths = 10000
	units[1].UsableAreaM2Hundredths = 10000
	units[2].UsableAreaM2Hundredths = 10000
	period := AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31"}
	entries := []AnnualStatementReserveEntry{
		{Kind: ReserveKindOpening, AmountCents: 100},
		{Kind: ReserveKindContribution, AmountCents: 50},
		{Kind: ReserveKindWithdrawal, AmountCents: 30},
		{Kind: ReserveKindInterest, AmountCents: 7},
	}
	result, ok := AnnualStatementReserveBalance(entries, period, units)
	if !ok || result.ClosingCents != 127 {
		t.Fatalf("closing=%d ok=%t", result.ClosingCents, ok)
	}
	var sum int64
	for _, share := range result.Shares {
		sum += share.AmountCents
	}
	if sum != result.ClosingCents || len(result.Shares) != 3 {
		t.Fatalf("shares do not conserve the closing: %+v", result.Shares)
	}
}

func TestReserveMinimumWarningAtTheThreshold(t *testing.T) {
	units := []Unit{{ID: "a", MiteigentumsanteilPPM: 1_000_000, UsableAreaM2Hundredths: 10000, UsableAreaRecorded: true}}
	for _, tc := range []struct {
		name, start, end string
		minimum, monthly int64
		rates            []int64
	}{
		{"2025", "2025-01-01", "2025-12-31", 127200, 10600, []int64{106}},
		{"2026", "2026-01-01", "2026-12-31", 134400, 11200, []int64{112}},
		{"2027", "2027-01-01", "2027-12-31", 134400, 11200, []int64{112}},
		{"2024 change", "2023-07-01", "2024-06-30", 117600, 0, []int64{90, 106}},
		{"2026 change", "2025-07-01", "2026-06-30", 130800, 0, []int64{106, 112}},
		{"introduced mid-year", "2022-01-01", "2022-12-31", 54000, 0, []int64{90}},
		{"first month", "2022-07-01", "2022-07-31", 9000, 9000, []int64{90}},
		{"partial months retain calendar-month rule", "2025-12-15", "2026-01-14", 21800, 0, []int64{106, 112}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Year deliberately differs: effective dates, not the year label, govern.
			period := AnnualStatementPeriod{Year: 2026, StartsOn: tc.start, EndsOn: tc.end}
			for _, contribution := range []int64{tc.minimum - 1, tc.minimum, tc.minimum + 1} {
				result, ok := AnnualStatementReserveBalance([]AnnualStatementReserveEntry{{Kind: ReserveKindContribution, AmountCents: contribution}}, period, units)
				if !ok || result.MinimumUnavailable || result.AreaIncomplete || result.MinimumPeriodCents != tc.minimum || result.MinimumMonthlyCents != tc.monthly || result.MinimumWarning != (contribution < tc.minimum) {
					t.Fatalf("contribution=%d: %+v, ok=%t", contribution, result, ok)
				}
				var rates []int64
				for _, rate := range result.MinimumRates {
					rates = append(rates, rate.CentsPerSquareMetreMonth)
				}
				if !reflect.DeepEqual(rates, tc.rates) {
					t.Fatalf("rates=%v want=%v", rates, tc.rates)
				}
			}
		})
	}
}

func TestReserveMinimumMonthlyRoundingAndUnavailablePeriods(t *testing.T) {
	for _, tc := range []struct {
		name, start, end        string
		area                    int
		recorded                bool
		minimum                 int64
		incomplete, unavailable bool
	}{
		{"half cent down each month", "2025-01-01", "2025-12-31", 25, true, 312, false, false},
		{"above half cent up", "2025-01-01", "2025-12-31", 26, true, 336, false, false},
		{"round each rate separately", "2025-07-01", "2026-06-30", 25, true, 324, false, false},
		{"no numeric floor yet", "2022-01-01", "2022-06-30", 10000, true, 0, false, false},
		{"area missing", "2025-01-01", "2025-12-31", 10000, false, 0, true, false},
		{"invalid date", "invalid", "2025-12-31", 10000, true, 0, false, true},
		{"reversed", "2026-01-01", "2025-12-31", 10000, true, 0, false, true},
		{"next rate unpublished", "2027-07-01", "2028-06-30", 10000, true, 0, false, true},
		{"overflow", "2026-01-01", "2026-12-31", math.MaxInt, true, 0, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, ok := AnnualStatementReserveBalance(nil, AnnualStatementPeriod{StartsOn: tc.start, EndsOn: tc.end}, []Unit{{UsableAreaM2Hundredths: tc.area, UsableAreaRecorded: tc.recorded}})
			if !ok || result.MinimumPeriodCents != tc.minimum || result.AreaIncomplete != tc.incomplete || result.MinimumUnavailable != tc.unavailable || result.MinimumWarning != (tc.minimum > 0 || tc.incomplete || tc.unavailable) {
				t.Fatalf("%+v, ok=%t", result, ok)
			}
		})
	}
}

func TestReserveDatedRateRunAndLegacyReplay(t *testing.T) {
	in := annualRunFixture()
	in.Structure.Legal.Regime = "weg"
	in.StatementOn = "2026-01-20"
	for i := range in.Structure.UnitBases {
		in.Structure.UnitBases[i].UsableAreaRecorded = true
		in.Structure.UnitBases[i].UsableAreaM2Hundredths = 5000
	}
	in.Reserve = []AnnualStatementReserveEntry{{Kind: ReserveKindContribution, AmountCents: 127200, EntryDate: "2025-01-01"}}
	result, issues := CalculateAnnualStatementRun(in)
	if len(issues) != 0 || result.Reserve == nil || result.Reserve.MinimumPeriodCents != 127200 || result.Reserve.MinimumWarning {
		t.Fatal(result.Reserve, issues)
	}
	run, err := newAnnualStatementRun(in, result, 1, "manager@example.com", time.Now())
	if err != nil || run.CalculationVersion != AnnualStatementCalculationVersionReserveRates {
		t.Fatal(run, err)
	}
	raw, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	var saved AnnualStatementRun
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	replay, issues := ReplayAnnualStatementRun(saved)
	if len(issues) != 0 || !reflect.DeepEqual(replay, saved.Result) {
		t.Fatal(replay, issues)
	}
	for _, version := range []int{1, 2, 3, 4, 5} {
		saved.CalculationVersion = version
		replay, issues := ReplayAnnualStatementRun(saved)
		if len(issues) != 0 || replay.Reserve == nil || replay.Reserve.MinimumPeriodCents != 134400 || !replay.Reserve.MinimumWarning || replay.Reserve.MinimumRates != nil {
			t.Fatalf("v%d: %+v %v", version, replay.Reserve, issues)
		}
	}
}

func TestHistoricalRunReplayIgnoresReserve(t *testing.T) {
	in := annualRunFixture()
	in.Structure.Legal.Regime = "weg"
	before, issues := CalculateAnnualStatementRun(in)
	if len(issues) > 0 || before.Reserve != nil {
		t.Fatal(before.Reserve, issues)
	}
	replay, issues := ReplayAnnualStatementRun(AnnualStatementRun{CalculationVersion: 2, Input: in})
	if len(issues) > 0 || !reflect.DeepEqual(before, replay) {
		t.Fatal(replay, issues)
	}
	legacy, issues := ReplayAnnualStatementRun(AnnualStatementRun{CalculationVersion: 1, Input: in})
	if len(issues) > 0 || legacy.Reserve != nil || legacy.Units[0].AllocatedCents != before.Units[0].AllocatedCents {
		t.Fatal(legacy, issues)
	}
	in.Reserve = []AnnualStatementReserveEntry{{Kind: ReserveKindOpening, AmountCents: 10, EntryDate: "2025-01-01"}}
	withReserve, issues := CalculateAnnualStatementRun(in)
	if len(issues) > 0 || withReserve.Reserve == nil || withReserve.Reserve.ClosingCents != 10 || withReserve.Units[0].AllocatedCents != before.Units[0].AllocatedCents {
		t.Fatal(withReserve, issues)
	}
}

func TestReservePostgresStoreAndRLS(t *testing.T) {
	if strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN")) == "" {
		t.Skip("set HAUSV_TEST_POSTGRES_DSN to check PostgreSQL reserve storage and RLS")
	}
	t.Setenv("HAUSV_STORE_TEST_POSTGRES", "1")
	lanes := testStoreDB(t)
	demo := testTenantRef("demo")
	other := testTenantRef("other")
	periods, ok := BindAnnualStatementPeriodRepository(NewSQLAnnualStatementPeriodStore(lanes), demo)
	if !ok {
		t.Fatal("bind periods")
	}
	if _, err := periods.Save(AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedAt: time.Now(), UpdatedBy: "manager@example.com"}); err != nil {
		t.Fatal(err)
	}
	repo, ok := BindAnnualStatementReserveRepository(NewSQLAnnualStatementReserveStore(lanes), demo)
	if !ok {
		t.Fatal("bind")
	}
	saved, err := repo.Add(AnnualStatementReserveEntry{PeriodYear: 2026, Kind: ReserveKindContribution, EntryDate: "2026-03-01", AmountCents: 250, Note: "März", CreatedAt: time.Now(), CreatedBy: "manager@example.com"})
	if err != nil || saved.ID == "" {
		t.Fatal(err)
	}
	if got := repo.ListByPeriod(2026); len(got) != 1 || got[0].AmountCents != 250 {
		t.Fatalf("%+v", got)
	}
	hidden, ok := BindAnnualStatementReserveRepository(NewSQLAnnualStatementReserveStore(lanes), other)
	if !ok || len(hidden.ListByPeriod(2026)) != 0 {
		t.Fatal("other tenant can see the reserve booking")
	}
}
