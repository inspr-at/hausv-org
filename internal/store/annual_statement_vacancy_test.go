package store

import (
	"os"
	"reflect"
	"testing"
)

func TestAnnualStatementVacancyBillsOwnerAndKeepsOtherTenants(t *testing.T) {
	base := annualRunFixture()
	base.Structure.Legal.Regime = "mrg_voll"
	without, issues := CalculateAnnualStatementRun(base)
	if len(issues) > 0 || len(without.Vacancy) != 0 {
		t.Fatalf("baseline: %+v %v", without, issues)
	}
	with := base
	with.Structure.UnitBases[0].VacantFrom = "2025-03-01"
	with.Structure.UnitBases[0].VacantTo = "2025-05-31"
	result, issues := CalculateAnnualStatementRun(with)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	if result.TotalCents != without.TotalCents {
		t.Fatalf("house total %d != %d", result.TotalCents, without.TotalCents)
	}
	if result.Units[1].AllocatedCents != without.Units[1].AllocatedCents || !reflect.DeepEqual(result.Units[1].Costs, without.Units[1].Costs) {
		t.Fatalf("other tenant changed: %+v vs %+v", result.Units[1], without.Units[1])
	}
	if len(result.Vacancy) != 1 || result.Vacancy[0].Label != "Top 1" || result.Vacancy[0].VacantDays != 92 || result.Vacancy[0].PeriodDays != 365 {
		t.Fatalf("owner line: %+v", result.Vacancy)
	}
	owner := result.Vacancy[0].AmountCents
	if owner <= 0 || result.Units[0].AllocatedCents+owner != without.Units[0].AllocatedCents {
		t.Fatalf("split %d + %d != %d", result.Units[0].AllocatedCents, owner, without.Units[0].AllocatedCents)
	}
	var costSum int64
	for _, unit := range result.Units {
		costSum += unit.AllocatedCents
	}
	costSum += owner
	if costSum != result.TotalCents {
		t.Fatalf("cents not conserved: %d != %d", costSum, result.TotalCents)
	}
}

func TestAnnualStatementVacancyLeapYearAndRegimes(t *testing.T) {
	in := annualRunFixture()
	in.Period = AnnualStatementPeriod{Year: 2024, StartsOn: "2024-01-01", EndsOn: "2024-12-31"}
	for i := range in.Receipts {
		in.Receipts[i].PeriodYear = 2024
		in.Receipts[i].InvoiceDate = "2024-02-29"
	}
	for i := range in.Prepayments {
		in.Prepayments[i].PeriodYear = 2024
	}
	in.Structure.Legal.Regime = "mrg_teil"
	in.Structure.UnitBases[0].VacantFrom = "2024-02-29"
	in.Structure.UnitBases[0].VacantTo = "2024-02-29"
	result, issues := CalculateAnnualStatementRun(in)
	if len(issues) > 0 || len(result.Vacancy) != 1 || result.Vacancy[0].VacantDays != 1 || result.Vacancy[0].PeriodDays != 366 {
		t.Fatalf("leap vacancy: %+v %v", result.Vacancy, issues)
	}
	full := result.Units[0].AllocatedCents + result.Vacancy[0].AmountCents
	if result.Vacancy[0].AmountCents != full/366 {
		t.Fatalf("one leap day of %d = %d", full, result.Vacancy[0].AmountCents)
	}
	in.Structure.Legal.Regime = "weg"
	weg, issues := CalculateAnnualStatementRun(in)
	if len(issues) > 0 || len(weg.Vacancy) != 0 || weg.Units[0].AllocatedCents != full {
		t.Fatalf("WEG must ignore vacancy: %+v %v", weg, issues)
	}
	in.Structure.Legal.Regime = "ausnahme"
	exception, issues := CalculateAnnualStatementRun(in)
	if len(issues) > 0 || len(exception.Vacancy) != 0 {
		t.Fatal(exception.Vacancy, issues)
	}
}

func TestAnnualStatementVacancyHistoricalReplayUnchanged(t *testing.T) {
	in := annualRunFixture()
	current, issues := CalculateAnnualStatementRun(in)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	replay, issues := ReplayAnnualStatementRun(AnnualStatementRun{CalculationVersion: AnnualStatementCalculationVersion, Input: in})
	if len(issues) > 0 || !reflect.DeepEqual(current, replay) || len(replay.Vacancy) != 0 {
		t.Fatalf("replay drifted: %+v %v", replay, issues)
	}
	again, issues := ReplayAnnualStatementRun(AnnualStatementRun{CalculationVersion: 2, Input: in})
	if len(issues) > 0 || !reflect.DeepEqual(again, current) {
		t.Fatal(again, issues)
	}
}

func TestAnnualStatementVacancyPersists(t *testing.T) {
	backends := map[string]func(t *testing.T) AnnualStatementPeriodStorage{
		"memory": func(t *testing.T) AnnualStatementPeriodStorage { return NewMemoryAnnualStatementPeriodStore() },
		"sql": func(t *testing.T) AnnualStatementPeriodStorage {
			_, lanes := testLanes(t)
			return NewSQLAnnualStatementPeriodStore(lanes)
		},
	}
	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			repo, ok := BindAnnualStatementPeriodRepository(build(t), testTenantRef("demo"))
			if !ok {
				t.Fatal("bind")
			}
			if _, err := repo.SaveWithStructure(AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31", UpdatedBy: "manager@example.com"}, AnnualStatementDefaultCostTypes("manager@example.com"), []Unit{{ID: "top-1", Label: "Top 1", MiteigentumsanteilPPM: 1000000, Persons: 1, PersonsRecorded: true, UsableAreaRecorded: true, UsableAreaM2Hundredths: 5000}}); err != nil {
				t.Fatal(err)
			}
			if err := repo.SaveLegal(2025, AnnualStatementLegalSettings{Regime: "mrg_voll", HeatingConsumptionPercent: 70}); err != nil {
				t.Fatal(err)
			}
			if err := repo.SaveStructureUnitBases(2025, []AnnualStatementPeriodUnitBasis{{
				UnitID: "top-1", MiteigentumsanteilPPM: 1000000, Persons: 1, PersonsRecorded: true, UsableAreaRecorded: true, UsableAreaM2Hundredths: 5000,
				VacantFrom: "2025-03-01", VacantTo: "2025-05-31",
			}}); err != nil {
				t.Fatal(err)
			}
			structure, found := repo.Structure(2025)
			if !found || structure.UnitBases[0].VacantFrom != "2025-03-01" || structure.UnitBases[0].VacantTo != "2025-05-31" || structure.Legal.Regime != "mrg_voll" {
				t.Fatalf("stored vacancy: %+v found=%t", structure, found)
			}
			if err := repo.SaveStructureUnitBases(2025, []AnnualStatementPeriodUnitBasis{{
				UnitID: "top-1", MiteigentumsanteilPPM: 1000000, VacantFrom: "2025-05-31", VacantTo: "2025-03-01",
			}}); err == nil {
				t.Fatal("reversed vacancy range must fail")
			}
		})
	}
}

func TestAnnualStatementVacancyPersistsPostgres(t *testing.T) {
	if os.Getenv("HAUSV_TEST_POSTGRES_DSN") == "" {
		t.Skip("set HAUSV_TEST_POSTGRES_DSN to check the PostgreSQL vacancy columns")
	}
	t.Setenv("HAUSV_STORE_TEST_POSTGRES", "1")
	_, lanes := testLanes(t)
	repo, ok := BindAnnualStatementPeriodRepository(NewSQLAnnualStatementPeriodStore(lanes), testTenantRef("demo"))
	if !ok {
		t.Fatal("bind")
	}
	if _, err := repo.SaveWithStructure(AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31", UpdatedBy: "manager@example.com"}, AnnualStatementDefaultCostTypes("manager@example.com"), []Unit{{ID: "top-1", Label: "Top 1", MiteigentumsanteilPPM: 1000000}}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveStructureUnitBases(2025, []AnnualStatementPeriodUnitBasis{{UnitID: "top-1", MiteigentumsanteilPPM: 1000000, VacantFrom: "2025-03-01", VacantTo: "2025-05-31"}}); err != nil {
		t.Fatal(err)
	}
	structure, found := repo.Structure(2025)
	if !found || structure.UnitBases[0].VacantFrom != "2025-03-01" || structure.UnitBases[0].VacantTo != "2025-05-31" {
		t.Fatalf("%+v", structure.UnitBases)
	}
}
