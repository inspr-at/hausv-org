package store

import (
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
			if _, err := demo.Save(AnnualStatementPeriod{Year: 2026, StartsOn: "2026-02-01", EndsOn: "2027-01-31", UpdatedBy: "manager@example.com"}); err != nil {
				t.Fatalf("update: %v", err)
			}
			got := demo.List()
			if len(got) != 2 || got[0].Year != 2026 || got[0].StartsOn != "2026-02-01" || got[1].Year != 2025 {
				t.Fatalf("list = %+v", got)
			}
			if otherPeriods := other.List(); len(otherPeriods) != 0 {
				t.Fatalf("other tenant saw %+v", otherPeriods)
			}
		})
	}
}
