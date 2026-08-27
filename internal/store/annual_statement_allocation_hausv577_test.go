package store

import (
	"reflect"
	"testing"
)

func TestUnitAllocationBasesUpdateBothBackends(t *testing.T) {
	backends := map[string]func(t *testing.T) UnitStorage{
		"json": func(t *testing.T) UnitStorage {
			s, err := NewUnitStore(t.TempDir() + "/units.json")
			if err != nil {
				t.Fatalf("json unit store: %v", err)
			}
			return s
		},
		"sql": func(t *testing.T) UnitStorage {
			_, lanes := testLanes(t)
			return NewSQLUnitStore(lanes)
		},
	}
	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			demo, ok := BindUnitRepository(build(t), testTenantRef("demo"))
			if !ok {
				t.Fatal("bind demo repository")
			}
			if err := demo.SetUnits([]Unit{
				{ID: "top-1", Label: "Top 1", MiteigentumsanteilPPM: 400_000, OwnerEmails: []string{"a@example.com"}},
				{ID: "top-2", Label: "Top 2", MiteigentumsanteilPPM: 600_000, Persons: -3},
			}); err != nil {
				t.Fatalf("seed: %v", err)
			}
			if got := demo.List()[1].Persons; got != 0 {
				t.Fatalf("negative persons must normalize to unrecorded, got %d", got)
			}
			unknown, err := demo.UpdateAllocationBases([]UnitAllocationBasisUpdate{
				{UnitID: "top-1", UsableAreaM2Hundredths: 7_250, Persons: 2},
				{UnitID: "top-9", UsableAreaM2Hundredths: 1},
			})
			if err != nil || !unknown {
				t.Fatalf("unknown unit must reject the whole update: unknown=%t err=%v", unknown, err)
			}
			if got := demo.List()[0]; got.UsableAreaM2Hundredths != 0 || got.Persons != 0 {
				t.Fatalf("rejected update leaked into top-1: %+v", got)
			}
			unknown, err = demo.UpdateAllocationBases([]UnitAllocationBasisUpdate{
				{UnitID: " TOP-1 ", UsableAreaM2Hundredths: 7_250, Persons: 2},
				{UnitID: "top-2", UsableAreaM2Hundredths: -5, Persons: 3},
			})
			if err != nil || unknown {
				t.Fatalf("update: unknown=%t err=%v", unknown, err)
			}
			units := demo.List()
			if units[0].UsableAreaM2Hundredths != 7_250 || units[0].Persons != 2 || units[0].MiteigentumsanteilPPM != 400_000 || len(units[0].OwnerEmails) != 1 {
				t.Fatalf("top-1 after update = %+v", units[0])
			}
			if units[1].UsableAreaM2Hundredths != 0 || units[1].Persons != 3 {
				t.Fatalf("top-2 after update = %+v", units[1])
			}
			other, _ := BindUnitRepository(build(t), testTenantRef("other"))
			if unknown, _ := other.UpdateAllocationBases([]UnitAllocationBasisUpdate{{UnitID: "top-1", Persons: 1}}); !unknown {
				t.Fatal("other tenant must not reach demo units")
			}
		})
	}
}

func TestAnnualStatementCostTypeAllocationKeyIsRequiredExactlyWhenAllocatable(t *testing.T) {
	demo, _ := BindAnnualStatementCostTypeRepository(NewMemoryAnnualStatementCostTypeStore(), testTenantRef("demo"))
	if err := demo.EnsureDefaults("manager@example.com"); err != nil {
		t.Fatal(err)
	}
	for _, costType := range demo.List() {
		if costType.AllocationKey != AllocationKeyNutzwert {
			t.Fatalf("starter %q must default to Nutzwert, got %q", costType.Key, costType.AllocationKey)
		}
	}
	if _, err := demo.Save(AnnualStatementCostType{Key: "wasser", Name: "Wasser", Allocatable: true}); err == nil {
		t.Fatal("allocatable cost type without key must fail")
	}
	if _, err := demo.Save(AnnualStatementCostType{Key: "wasser", Name: "Wasser", Allocatable: true, AllocationKey: "mea"}); err == nil {
		t.Fatal("unknown key must fail")
	}
	saved, err := demo.Save(AnnualStatementCostType{Key: "wasser", Name: "Wasser", Allocatable: true, AllocationKey: " Verbrauch "})
	if err != nil || saved.AllocationKey != AllocationKeyVerbrauch {
		t.Fatalf("save verbrauch: %+v %v", saved, err)
	}
	saved, err = demo.Save(AnnualStatementCostType{Key: "ruecklage", Name: "Rücklage", Allocatable: false, AllocationKey: "nutzwert"})
	if err != nil || saved.AllocationKey != "" {
		t.Fatalf("non-allocatable cost type must clear its key: %+v %v", saved, err)
	}
}

func TestAnnualStatementAllocationPreviewsSplitExactlyAndBlockUnmappedUnits(t *testing.T) {
	costTypes := []AnnualStatementCostType{
		{Key: "grundsteuer", Allocatable: true, AllocationKey: AllocationKeyNutzwert},
		{Key: "hausbetreuung", Allocatable: true, AllocationKey: AllocationKeyNutzwert},
		{Key: "wasser", Allocatable: true, AllocationKey: AllocationKeyPersonen},
		{Key: "heizung", Allocatable: true, AllocationKey: AllocationKeyVerbrauch},
		{Key: "ruecklage", Allocatable: false},
	}
	units := []Unit{
		{ID: "top-1", Label: "Top 1", MiteigentumsanteilPPM: 1, Persons: 1},
		{ID: "top-2", Label: "Top 2", MiteigentumsanteilPPM: 1, Persons: 1},
		{ID: "top-3", Label: "Top 3", MiteigentumsanteilPPM: 1},
	}
	previews := AnnualStatementAllocationPreviews(costTypes, units)
	if len(previews) != 3 {
		t.Fatalf("previews = %+v, want nutzwert, personen, verbrauch", previews)
	}
	nutzwert := previews[0]
	if nutzwert.Key != AllocationKeyNutzwert || !reflect.DeepEqual(nutzwert.CostTypeKeys, []string{"grundsteuer", "hausbetreuung"}) || nutzwert.Blocked {
		t.Fatalf("nutzwert preview = %+v", nutzwert)
	}
	sum := 0
	for _, share := range nutzwert.Shares {
		sum += share.SharePPM
	}
	if sum != 1_000_000 || nutzwert.Shares[0].SharePPM != 333_334 || nutzwert.Shares[2].SharePPM != 333_333 {
		t.Fatalf("thirds must sum to exactly one million with the remainder on the first unit: %+v", nutzwert.Shares)
	}
	personen := previews[1]
	if personen.Key != AllocationKeyPersonen || !personen.Blocked || !reflect.DeepEqual(personen.UnmappedUnits, []string{"Top 3"}) {
		t.Fatalf("personen preview must block on the unit without persons: %+v", personen)
	}
	for _, share := range personen.Shares {
		if share.SharePPM != 0 {
			t.Fatalf("blocked preview must not publish shares: %+v", personen.Shares)
		}
	}
	verbrauch := previews[2]
	if verbrauch.Key != AllocationKeyVerbrauch || !verbrauch.Blocked || len(verbrauch.UnmappedUnits) != 3 {
		t.Fatalf("verbrauch has no source before HAUSV-578 and must block: %+v", verbrauch)
	}
	if empty := AnnualStatementAllocationPreviews(costTypes[:1], nil); len(empty) != 1 || !empty[0].Blocked {
		t.Fatalf("no units must block, not silently pass: %+v", empty)
	}
	if none := AnnualStatementAllocationPreviews(costTypes[4:], units); len(none) != 0 {
		t.Fatalf("non-allocatable cost types need no preview: %+v", none)
	}
}
