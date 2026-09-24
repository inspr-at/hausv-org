package store

import (
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
	// 100,00 m² × 1,12 € = 112,00 € per month, 1.344,00 € for twelve months.
	units := []Unit{{ID: "a", MiteigentumsanteilPPM: 1_000_000, UsableAreaM2Hundredths: 10000, UsableAreaRecorded: true}}
	period := AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31"}
	at := func(cents int64) AnnualStatementReserveResult {
		result, ok := AnnualStatementReserveBalance([]AnnualStatementReserveEntry{{Kind: ReserveKindContribution, AmountCents: cents}}, period, units)
		if !ok {
			t.Fatal("balance")
		}
		return result
	}
	if at(134400).MinimumWarning || at(134400).MinimumPeriodCents != 134400 {
		t.Fatalf("threshold must hold: %+v", at(134400))
	}
	if !at(134399).MinimumWarning {
		t.Fatal("one cent below the floor must warn")
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
