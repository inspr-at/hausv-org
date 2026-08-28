package store

import (
	"testing"
	"time"
)

func TestAnnualStatementCostTypeStorage(t *testing.T) {
	backends := map[string]func(t *testing.T) AnnualStatementCostTypeStorage{
		"memory": func(t *testing.T) AnnualStatementCostTypeStorage {
			return NewMemoryAnnualStatementCostTypeStore()
		},
		"sql": func(t *testing.T) AnnualStatementCostTypeStorage {
			_, lanes := testLanes(t)
			return NewSQLAnnualStatementCostTypeStore(lanes)
		},
	}
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			storage := build(t)
			demo, ok := BindAnnualStatementCostTypeRepository(storage, testTenantRef("demo"))
			if !ok {
				t.Fatal("bind demo repository")
			}
			other, ok := BindAnnualStatementCostTypeRepository(storage, testTenantRef("other"))
			if !ok {
				t.Fatal("bind other repository")
			}
			if invalid, ok := BindAnnualStatementCostTypeRepository(storage, TenantRef{}); ok || invalid != nil {
				t.Fatal("invalid tenant must not bind")
			}

			if err := demo.EnsureDefaults("Manager@Example.com"); err != nil {
				t.Fatalf("ensure defaults: %v", err)
			}
			if err := demo.EnsureDefaults("manager@example.com"); err != nil {
				t.Fatalf("ensure defaults idempotently: %v", err)
			}
			defaults := demo.List()
			if len(defaults) != 5 {
				t.Fatalf("default catalogue length = %d, want 5: %+v", len(defaults), defaults)
			}
			byKey := map[string]AnnualStatementCostType{}
			for _, costType := range defaults {
				byKey[costType.Key] = costType
			}
			for _, key := range []string{"grundsteuer", "muellabfuhr", "hausbetreuung", "gebaeudeversicherung", "gartenpflege"} {
				if costType, exists := byKey[key]; !exists || !costType.Allocatable {
					t.Errorf("starter %q = %+v, exists=%t", key, costType, exists)
				}
			}

			created, err := demo.Save(AnnualStatementCostType{
				Key: "  ruecklage_aufzug ", Name: " Rücklage Aufzug ", Allocatable: false,
				UpdatedAt: now, UpdatedBy: "MANAGER@EXAMPLE.COM",
			})
			if err != nil {
				t.Fatalf("save custom cost type: %v", err)
			}
			if created.Key != "ruecklage_aufzug" || created.Name != "Rücklage Aufzug" || created.Allocatable || created.UpdatedBy != "manager@example.com" || !created.UpdatedAt.Equal(now) {
				t.Fatalf("normalized cost type = %+v", created)
			}
			created.Name = "Aufzugsrücklage"
			created.Allocatable = true
			created.AllocationKey = AllocationKeyFlaeche
			if _, err := demo.Save(created); err != nil {
				t.Fatalf("update cost type: %v", err)
			}
			if got := demo.List(); len(got) != 6 {
				t.Fatalf("catalogue length after update = %d, want 6: %+v", len(got), got)
			}
			if got := other.List(); len(got) != 0 {
				t.Fatalf("other tenant saw %+v", got)
			}
			if _, err := demo.Save(AnnualStatementCostType{Key: "Ungültig!", Name: "Test"}); err == nil {
				t.Fatal("invalid key must fail")
			}
			if _, err := demo.Save(AnnualStatementCostType{Key: "leer", Name: " "}); err == nil {
				t.Fatal("empty name must fail")
			}
		})
	}
}
