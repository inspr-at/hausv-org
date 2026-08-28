package store

import (
	"reflect"
	"testing"
)

func unitBasisBackends() map[string]func(t *testing.T) UnitStorage {
	return map[string]func(t *testing.T) UnitStorage{
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
}

func TestUnitAllocationBasesUpdateBothBackends(t *testing.T) {
	for name, build := range unitBasisBackends() {
		t.Run(name, func(t *testing.T) {
			storage := build(t)
			demo, ok := BindUnitRepository(storage, testTenantRef("demo"))
			if !ok {
				t.Fatal("bind demo repository")
			}
			if err := demo.SetUnits([]Unit{
				{ID: "top-1", Label: "Top 1", MiteigentumsanteilPPM: 400_000, OwnerEmails: []string{"a@example.com"}},
				{ID: "top-2", Label: "Top 2", MiteigentumsanteilPPM: 600_000, Persons: 3},
			}); err != nil {
				t.Fatalf("seed: %v", err)
			}
			if got := demo.List()[1]; got.Persons != 0 || got.PersonsRecorded {
				t.Fatalf("persons without the recorded flag must normalize to unrecorded, got %+v", got)
			}
			unknown, err := demo.UpdateAllocationBases([]UnitAllocationBasisUpdate{
				{UnitID: "top-1", UsableAreaM2Hundredths: 7_250, UsableAreaRecorded: true, Persons: 2, PersonsRecorded: true},
				{UnitID: "top-9", UsableAreaM2Hundredths: 1, UsableAreaRecorded: true},
			})
			if err != nil || !unknown {
				t.Fatalf("unknown unit must reject the whole update: unknown=%t err=%v", unknown, err)
			}
			unknown, err = demo.UpdateAllocationBases([]UnitAllocationBasisUpdate{
				{UnitID: "top-1", UsableAreaM2Hundredths: 7_250, UsableAreaRecorded: true, Persons: 2, PersonsRecorded: true},
				{UnitID: "   ", UsableAreaM2Hundredths: 1, UsableAreaRecorded: true},
			})
			if err != nil || !unknown {
				t.Fatalf("blank unit id is malformed input and must reject the whole update: unknown=%t err=%v", unknown, err)
			}
			if got := demo.List()[0]; got.UsableAreaM2Hundredths != 0 || got.UsableAreaRecorded || got.Persons != 0 || got.PersonsRecorded {
				t.Fatalf("rejected updates leaked into top-1: %+v", got)
			}
			unknown, err = demo.UpdateAllocationBases([]UnitAllocationBasisUpdate{
				{UnitID: " TOP-1 ", UsableAreaM2Hundredths: 7_250, UsableAreaRecorded: true, Persons: 2, PersonsRecorded: true},
				{UnitID: "top-2", UsableAreaM2Hundredths: 0, UsableAreaRecorded: true, Persons: 0, PersonsRecorded: true},
			})
			if err != nil || unknown {
				t.Fatalf("update: unknown=%t err=%v", unknown, err)
			}
			units := demo.List()
			if units[0].UsableAreaM2Hundredths != 7_250 || !units[0].UsableAreaRecorded || units[0].Persons != 2 || !units[0].PersonsRecorded || units[0].MiteigentumsanteilPPM != 400_000 || len(units[0].OwnerEmails) != 1 {
				t.Fatalf("top-1 after update = %+v", units[0])
			}
			if units[1].UsableAreaM2Hundredths != 0 || !units[1].UsableAreaRecorded || units[1].Persons != 0 || !units[1].PersonsRecorded {
				t.Fatalf("top-2 must keep an explicitly recorded 0 m² and 0 persons: %+v", units[1])
			}
			if unknown, err := demo.UpdateAllocationBases([]UnitAllocationBasisUpdate{{UnitID: "top-2", UsableAreaM2Hundredths: 9, Persons: 9}}); err != nil || unknown {
				t.Fatalf("clear update: unknown=%t err=%v", unknown, err)
			}
			if got := demo.List()[1]; got.UsableAreaM2Hundredths != 0 || got.UsableAreaRecorded || got.Persons != 0 || got.PersonsRecorded {
				t.Fatalf("values without the recorded flag must normalize to unrecorded: %+v", got)
			}
			if unknown, err := demo.UpdateAllocationBases([]UnitAllocationBasisUpdate{{UnitID: "top-2", UsableAreaRecorded: true, PersonsRecorded: true}}); err != nil || unknown {
				t.Fatalf("restore update: unknown=%t err=%v", unknown, err)
			}
			// An ordinary unit edit rebuilds the record from the form; the
			// bases survive only when carried over, on every backend.
			fresh := Unit{ID: "top-1", Label: "Top 1 (neu)", UnitType: UnitTypeResidential, MiteigentumsanteilPPM: 400_000}
			if duplicate, err := demo.UpsertUnit("top-1", CarryAllocationBases(units[0], fresh)); err != nil || duplicate {
				t.Fatalf("upsert carried unit: duplicate=%t err=%v", duplicate, err)
			}
			if got := demo.List()[0]; got.Label != "Top 1 (neu)" || got.UsableAreaM2Hundredths != 7_250 || !got.UsableAreaRecorded || got.Persons != 2 || !got.PersonsRecorded {
				t.Fatalf("carried bases lost on upsert: %+v", got)
			}
			other, _ := BindUnitRepository(storage, testTenantRef("other"))
			if unknown, _ := other.UpdateAllocationBases([]UnitAllocationBasisUpdate{{UnitID: "top-1", Persons: 1, PersonsRecorded: true}}); !unknown {
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
		{ID: "top-1", Label: "Top 1", MiteigentumsanteilPPM: 1, Persons: 1, PersonsRecorded: true},
		{ID: "top-2", Label: "Top 2", MiteigentumsanteilPPM: 1, Persons: 1, PersonsRecorded: true},
		{ID: "top-3", Label: "Top 3", MiteigentumsanteilPPM: 1},
	}
	previews := AnnualStatementAllocationPreviews(costTypes, units)
	if len(previews) != 3 {
		t.Fatalf("previews = %+v, want nutzwert, personen, verbrauch", previews)
	}
	nutzwert := previews[0]
	if nutzwert.Key != AllocationKeyNutzwert || !reflect.DeepEqual(nutzwert.CostTypeKeys, []string{"grundsteuer", "hausbetreuung"}) || nutzwert.Blocked || nutzwert.BasisTotal != 3 {
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
		t.Fatalf("personen preview must block on the unit without recorded persons: %+v", personen)
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
	if !AnnualStatementRunBlocked(costTypes, units) {
		t.Fatal("run must be blocked while any key in use has an unmapped unit")
	}

	// A Stellplatz with an explicitly recorded 0 persons is mapped with a
	// zero share; it does not block and does not distort the others.
	units[2].PersonsRecorded = true
	personen = AnnualStatementAllocationPreviews(costTypes[2:3], units)[0]
	if personen.Blocked || personen.BasisTotal != 2 || personen.Shares[0].SharePPM != 500_000 || personen.Shares[2].SharePPM != 0 || !personen.Shares[2].Mapped {
		t.Fatalf("recorded zero persons must be mapped with a zero share: %+v", personen)
	}
	allZero := []Unit{{ID: "g1", Label: "Garage 1", Persons: 0, PersonsRecorded: true}}
	if got := AnnualStatementAllocationPreviews(costTypes[2:3], allZero)[0]; !got.Blocked {
		t.Fatalf("a key with nothing to split must block, not divide by zero: %+v", got)
	}
	flaeche := []AnnualStatementCostType{{Key: "reinigung", Allocatable: true, AllocationKey: AllocationKeyFlaeche}}
	areaUnits := []Unit{
		{ID: "top-1", Label: "Top 1", UsableAreaM2Hundredths: 7_500, UsableAreaRecorded: true},
		{ID: "g-1", Label: "Garage 1", UsableAreaM2Hundredths: 0, UsableAreaRecorded: true},
		{ID: "top-2", Label: "Top 2", UsableAreaM2Hundredths: 5_000},
	}
	if got := AnnualStatementAllocationPreviews(flaeche, areaUnits)[0]; !got.Blocked || !reflect.DeepEqual(got.UnmappedUnits, []string{"Top 2"}) || got.Shares[1].Mapped != true {
		t.Fatalf("area without the recorded flag must block while a recorded 0 m² is mapped: %+v", got)
	}
	areaUnits[2].UsableAreaRecorded = true
	if got := AnnualStatementAllocationPreviews(flaeche, areaUnits)[0]; got.Blocked || got.Shares[0].SharePPM != 600_000 || got.Shares[1].SharePPM != 0 || got.Shares[2].SharePPM != 400_000 {
		t.Fatalf("recorded 0 m² must be a zero share, not a blocker: %+v", got)
	}

	if empty := AnnualStatementAllocationPreviews(costTypes[:1], nil); len(empty) != 1 || !empty[0].Blocked {
		t.Fatalf("no units must block, not silently pass: %+v", empty)
	}
	if none := AnnualStatementAllocationPreviews(costTypes[4:], units); len(none) != 0 {
		t.Fatalf("non-allocatable cost types need no preview: %+v", none)
	}
	keyless := []AnnualStatementCostType{{Key: "alt", Allocatable: true}}
	if got := AnnualStatementCostTypesWithoutKey(keyless); !reflect.DeepEqual(got, []string{"alt"}) || !AnnualStatementRunBlocked(keyless, units) {
		t.Fatalf("allocatable cost type without key must be reported and block the run: %v", got)
	}
	if AnnualStatementRunBlocked(costTypes[:2], units) {
		t.Fatal("fully mapped Nutzwert keys must not block the run")
	}
}
