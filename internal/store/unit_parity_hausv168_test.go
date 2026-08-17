package store

import (
	"path/filepath"
	"testing"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func TestUnitStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) UnitStorage{
		"json": func(t *testing.T) UnitStorage {
			s, err := NewUnitStore(filepath.Join(t.TempDir(), "units.json"))
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) UnitStorage {
			database := dbtest.Open(t)
			t.Cleanup(func() { database.Close() })
			return NewSQLUnitStore(database)
		},
	}

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			storage := build(t)
			s, ok := BindUnitRepository(storage, testTenantRef("demo"))
			if !ok {
				t.Fatal("bind demo repository")
			}
			other, ok := BindUnitRepository(storage, testTenantRef("other"))
			if !ok {
				t.Fatal("bind other repository")
			}
			if unscoped, ok := BindUnitRepository(storage, testTenantRef("")); ok || unscoped != nil {
				t.Fatal("empty tenant must not produce a repository")
			}

			if err := s.SetUnits([]Unit{
				{ID: "top-1", Label: "Top 1", MiteigentumsanteilPPM: 400000, OwnerEmails: []string{"a@example.com"}},
				{ID: "top-2", Label: "Top 2", MiteigentumsanteilPPM: 600000, RenterEmails: []string{"b@example.com"}},
			}); err != nil {
				t.Fatalf("set units: %v", err)
			}
			if got := s.UnitCount(); got != 2 {
				t.Fatalf("count = %d", got)
			}
			// Another tenant is untouched by a tenant-scoped write.
			if err := other.SetUnits([]Unit{{ID: "x-1", Label: "X"}}); err != nil {
				t.Fatalf("set other: %v", err)
			}
			if got := s.UnitCount(); got != 2 {
				t.Fatalf("demo count after other tenant write = %d", got)
			}
			if got := other.UnitCount(); got != 1 {
				t.Fatalf("other count = %d", got)
			}

			// Membership lookups.
			if got := s.UnitsForEmail("a@example.com"); len(got) != 1 || got[0].Relation != RoleOwner {
				t.Fatalf("owner membership = %+v", got)
			}
			if got := s.UnitsForEmail("b@example.com"); len(got) != 1 || got[0].Relation != RoleRenter {
				t.Fatalf("renter membership = %+v", got)
			}
			if got := s.UnitsForEmail("nobody@example.com"); len(got) != 0 {
				t.Fatalf("stranger membership = %+v", got)
			}
			members := s.MembersForUnit("top-1")
			if !members.Found || len(members.Owners) != 1 || members.Owners[0] != "a@example.com" {
				t.Fatalf("members = %+v", members)
			}
			if got := s.MembersForUnit("nope"); got.Found {
				t.Fatalf("unknown unit must not be found: %+v", got)
			}

			// Upsert: create a new unit.
			if dup, err := s.UpsertUnit("", Unit{ID: "top-3", Label: "Top 3"}); err != nil || dup {
				t.Fatalf("create: dup=%v err=%v", dup, err)
			}
			if got := s.UnitCount(); got != 3 {
				t.Fatalf("count after create = %d", got)
			}
			// Creating the same ID again is a duplicate.
			if dup, err := s.UpsertUnit("", Unit{ID: "top-3", Label: "Dup"}); err != nil || !dup {
				t.Fatalf("duplicate create: dup=%v err=%v", dup, err)
			}
			if got := s.UnitCount(); got != 3 {
				t.Fatalf("rejected duplicate must not add: %d", got)
			}
			// Editing a unit in place (same id) is allowed.
			if dup, err := s.UpsertUnit("top-3", Unit{ID: "top-3", Label: "Renamed label"}); err != nil || dup {
				t.Fatalf("in-place edit: dup=%v err=%v", dup, err)
			}
			// Renaming onto an existing id is a duplicate.
			if dup, err := s.UpsertUnit("top-3", Unit{ID: "top-1", Label: "Clash"}); err != nil || !dup {
				t.Fatalf("rename onto existing: dup=%v err=%v", dup, err)
			}
			// A real rename moves the unit.
			if dup, err := s.UpsertUnit("top-3", Unit{ID: "top-9", Label: "Top 9"}); err != nil || dup {
				t.Fatalf("rename: dup=%v err=%v", dup, err)
			}
			ids := map[string]bool{}
			for _, u := range s.List() {
				ids[u.ID] = true
			}
			if ids["top-3"] || !ids["top-9"] {
				t.Fatalf("rename left the old id: %v", ids)
			}
			if got := s.UnitCount(); got != 3 {
				t.Fatalf("rename must not change the count: %d", got)
			}

			// Delete.
			removed, unit, err := s.DeleteUnit("top-9")
			if err != nil || !removed || unit.ID != "top-9" {
				t.Fatalf("delete: removed=%v unit=%+v err=%v", removed, unit, err)
			}
			if got := s.UnitCount(); got != 2 {
				t.Fatalf("count after delete = %d", got)
			}
			if removed, _, err := s.DeleteUnit("top-9"); err != nil || removed {
				t.Fatalf("second delete: removed=%v err=%v", removed, err)
			}

			// Billable weight is derived consistently.
			if got := s.BillableUnitWeight(); got != BillableUnitWeight(s.List()) {
				t.Fatalf("billable weight mismatch: %d", got)
			}
		})
	}
}

func TestSQLUnitImportFromJSON(t *testing.T) {
	jsonStore, err := NewUnitStore(filepath.Join(t.TempDir(), "units.json"))
	if err != nil {
		t.Fatalf("json store: %v", err)
	}
	jsonRepository, _ := BindUnitRepository(jsonStore, testTenantRef("demo"))
	if err := jsonRepository.SetUnits([]Unit{
		{ID: "top-1", Label: "Top 1", MiteigentumsanteilPPM: 500000, OwnerEmails: []string{"a@example.com"}},
		{ID: "top-2", Label: "Top 2", MiteigentumsanteilPPM: 500000},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	database := dbtest.Open(t)
	defer database.Close()
	sqlStore := NewSQLUnitStore(database)
	sqlRepository, _ := BindUnitRepository(sqlStore, testTenantRef("demo"))

	for i := 0; i < 2; i++ {
		if err := sqlStore.ImportUnits(jsonStore); err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
	}
	if got := sqlRepository.UnitCount(); got != 2 {
		t.Fatalf("imported %d units, want 2", got)
	}
	if got := sqlRepository.UnitsForEmail("a@example.com"); len(got) != 1 {
		t.Fatalf("membership after import = %+v", got)
	}
	if got := sqlRepository.BillableUnitWeight(); got != jsonRepository.BillableUnitWeight() {
		t.Fatalf("billable weight drifted after import: %d vs %d", got, jsonRepository.BillableUnitWeight())
	}
}
