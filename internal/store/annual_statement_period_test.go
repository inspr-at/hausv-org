package store

import (
	"reflect"
	"testing"
	"time"
)

func TestAnnualStatementPeriodStorage(t *testing.T) {
	backends := map[string]func(t *testing.T) AnnualStatementPeriodStorage{
		"memory": func(t *testing.T) AnnualStatementPeriodStorage {
			return NewMemoryAnnualStatementPeriodStore()
		},
		"sql": func(t *testing.T) AnnualStatementPeriodStorage {
			_, lanes := testLanes(t)
			return NewSQLAnnualStatementPeriodStore(lanes)
		},
	}
	now := time.Date(2026, 8, 27, 8, 30, 0, 0, time.UTC)
	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			storage := build(t)
			demo, ok := BindAnnualStatementPeriodRepository(storage, testTenantRef("demo"))
			if !ok {
				t.Fatal("bind demo repository")
			}
			other, ok := BindAnnualStatementPeriodRepository(storage, testTenantRef("other"))
			if !ok {
				t.Fatal("bind other repository")
			}
			if invalid, ok := BindAnnualStatementPeriodRepository(storage, TenantRef{}); ok || invalid != nil {
				t.Fatal("invalid tenant must not bind")
			}

			if _, err := demo.Save(AnnualStatementPeriod{Year: 2026, StartsOn: "2026-12-31", EndsOn: "2026-01-01"}); err == nil {
				t.Fatal("reversed period must fail")
			}
			created, err := demo.Save(AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedAt: now, UpdatedBy: "Manager@Example.com"})
			if err != nil {
				t.Fatalf("save: %v", err)
			}
			if created.UpdatedBy != "manager@example.com" || !created.UpdatedAt.Equal(now) {
				t.Fatalf("normalized period = %+v", created)
			}
			if _, err := demo.Save(AnnualStatementPeriod{Year: 2025, StartsOn: "2025-04-01", EndsOn: "2026-03-31"}); err != nil {
				t.Fatalf("save cross-calendar period: %v", err)
			}
			if _, created, err := demo.Create(AnnualStatementPeriod{Year: 2026, StartsOn: "2026-03-01", EndsOn: "2027-02-28"}); err != nil || created {
				t.Fatalf("create must not overwrite existing period: created=%t err=%v", created, err)
			}
			createdNext, inserted, err := demo.Create(AnnualStatementPeriod{Year: 2027, StartsOn: "2027-01-01", EndsOn: "2027-12-31", UpdatedBy: "manager@example.com"})
			if err != nil || !inserted || createdNext.Year != 2027 {
				t.Fatalf("create new period = %+v created=%t err=%v", createdNext, inserted, err)
			}
			if _, err := demo.Save(AnnualStatementPeriod{Year: 2026, StartsOn: "2026-02-01", EndsOn: "2027-01-31", UpdatedBy: "manager@example.com"}); err != nil {
				t.Fatalf("update: %v", err)
			}
			got := demo.List()
			if len(got) != 3 || got[0].Year != 2027 || got[1].Year != 2026 || got[1].StartsOn != "2026-02-01" || got[2].Year != 2025 {
				t.Fatalf("list = %+v", got)
			}
			if otherPeriods := other.List(); len(otherPeriods) != 0 {
				t.Fatalf("other tenant saw %+v", otherPeriods)
			}
		})
	}
}

func TestAnnualStatementPeriodStructureCloneIsIndependentAndAtomic(t *testing.T) {
	backends := map[string]func(t *testing.T) AnnualStatementPeriodStorage{
		"memory": func(t *testing.T) AnnualStatementPeriodStorage { return NewMemoryAnnualStatementPeriodStore() },
		"sql": func(t *testing.T) AnnualStatementPeriodStorage {
			_, lanes := testLanes(t)
			return NewSQLAnnualStatementPeriodStore(lanes)
		},
	}
	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			storage := build(t)
			demo, _ := BindAnnualStatementPeriodRepository(storage, testTenantRef("demo"))
			other, _ := BindAnnualStatementPeriodRepository(storage, testTenantRef("other"))
			if _, err := demo.Save(AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com"}); err != nil {
				t.Fatal(err)
			}
			costTypes := []AnnualStatementCostType{{
				Key: "grundsteuer", Name: "Grundsteuer", Allocatable: true, AllocationKey: AllocationKeyNutzwert, UpdatedBy: "manager@example.com",
			}}
			units := []Unit{{
				ID: "top-1", Label: "Top 1", MiteigentumsanteilPPM: 1_000_000,
				UsableAreaM2Hundredths: 7_500, UsableAreaRecorded: true, Persons: 2, PersonsRecorded: true,
			}}
			if err := demo.EnsureStructure(2026, costTypes, units, "manager@example.com"); err != nil {
				t.Fatal(err)
			}
			sourceBefore, ok := demo.Structure(2026)
			if !ok {
				t.Fatal("source structure missing")
			}
			if _, err := demo.SaveWithStructure(AnnualStatementPeriod{
				Year: 2026, StartsOn: "2026-02-01", EndsOn: "2027-01-31", UpdatedBy: "manager@example.com",
			}, []AnnualStatementCostType{{
				Key: "changed", Name: "Must not replace snapshot", Allocatable: true, AllocationKey: AllocationKeyPersonen, UpdatedBy: "manager@example.com",
			}}, []Unit{{ID: "top-1", MiteigentumsanteilPPM: 42}}); err != nil {
				t.Fatal(err)
			}
			sourceAfterPeriodEdit, ok := demo.Structure(2026)
			if !ok || !reflect.DeepEqual(sourceBefore, sourceAfterPeriodEdit) {
				t.Fatalf("editing period dates replaced its snapshot: before=%+v after=%+v", sourceBefore, sourceAfterPeriodEdit)
			}
			if periods := demo.List(); len(periods) != 1 || periods[0].StartsOn != "2026-02-01" || periods[0].EndsOn != "2027-01-31" {
				t.Fatalf("period dates were not updated while preserving snapshot: %+v", periods)
			}
			targetPeriod := AnnualStatementPeriod{Year: 2027, StartsOn: "2027-01-01", EndsOn: "2027-12-31", UpdatedBy: "manager@example.com"}
			if _, created, err := demo.CloneStructure(2026, targetPeriod); err != nil || !created {
				t.Fatalf("clone: created=%t err=%v", created, err)
			}
			if _, err := demo.SaveStructureCostType(2027, AnnualStatementCostType{
				Key: "grundsteuer", Name: "Grundsteuer Zieljahr", Allocatable: true, AllocationKey: AllocationKeyPersonen, UpdatedBy: "manager@example.com",
			}); err != nil {
				t.Fatal(err)
			}
			if err := demo.SaveStructureUnitBases(2027, []AnnualStatementPeriodUnitBasis{{
				UnitID: "top-1", MiteigentumsanteilPPM: 900_000, UsableAreaM2Hundredths: 9_000, UsableAreaRecorded: true, Persons: 4, PersonsRecorded: true,
			}}); err != nil {
				t.Fatal(err)
			}
			sourceAfter, ok := demo.Structure(2026)
			if !ok || !reflect.DeepEqual(sourceBefore, sourceAfter) {
				t.Fatalf("target edits changed source: before=%+v after=%+v", sourceBefore, sourceAfter)
			}
			targetEdited, _ := demo.Structure(2027)
			if len(targetEdited.UnitBases) != 1 || targetEdited.UnitBases[0].MiteigentumsanteilPPM != 900_000 {
				t.Fatalf("target Nutzwert basis was not independently editable: %+v", targetEdited.UnitBases)
			}
			targetBeforeDuplicate, _ := demo.Structure(2027)
			if _, created, err := demo.CloneStructure(2026, targetPeriod); err != nil || created {
				t.Fatalf("duplicate clone: created=%t err=%v", created, err)
			}
			targetAfterDuplicate, _ := demo.Structure(2027)
			if !reflect.DeepEqual(targetBeforeDuplicate, targetAfterDuplicate) {
				t.Fatalf("duplicate clone partially overwrote target: before=%+v after=%+v", targetBeforeDuplicate, targetAfterDuplicate)
			}
			if _, found := other.Structure(2026); found {
				t.Fatal("other tenant saw demo period structure")
			}
		})
	}
}

func TestAnnualStatementSaveWithStructureRollsBackPeriodWhenSnapshotInsertFails(t *testing.T) {
	database, lanes := testLanes(t)
	repository, ok := BindAnnualStatementPeriodRepository(NewSQLAnnualStatementPeriodStore(lanes), testTenantRef("demo"))
	if !ok {
		t.Fatal("bind repository")
	}
	if _, err := database.Exec(`CREATE TRIGGER fail_period_structure
		BEFORE INSERT ON annual_statement_period_cost_types
		BEGIN SELECT RAISE(ABORT, 'injected period snapshot failure'); END`); err != nil {
		t.Fatal(err)
	}
	_, err := repository.SaveWithStructure(AnnualStatementPeriod{
		Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com",
	}, []AnnualStatementCostType{{
		Key: "grundsteuer", Name: "Grundsteuer", Allocatable: true, AllocationKey: AllocationKeyNutzwert, UpdatedBy: "manager@example.com",
	}}, []Unit{{ID: "top-1", MiteigentumsanteilPPM: 1_000_000}})
	if err == nil {
		t.Fatal("injected snapshot failure unexpectedly succeeded")
	}
	if periods := repository.List(); len(periods) != 0 {
		t.Fatalf("failed snapshot initialization left orphan periods: %+v", periods)
	}
	for _, table := range []string{"annual_statement_periods", "annual_statement_period_cost_types", "annual_statement_period_unit_bases"} {
		var count int
		if err := database.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("failed snapshot initialization left %d rows in %s", count, table)
		}
	}
}
