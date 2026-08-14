package store

import (
	"path/filepath"
	"testing"

	"github.com/inspr-at/hausv-org/internal/db"
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
			database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatalf("db open: %v", err)
			}
			t.Cleanup(func() { database.Close() })
			return NewSQLUnitStore(database)
		},
	}

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s := build(t)

			if err := s.SetTenantUnits("demo", []Unit{
				{ID: "top-1", Label: "Top 1", MiteigentumsanteilPPM: 400000, OwnerEmails: []string{"a@example.com"}},
				{ID: "top-2", Label: "Top 2", MiteigentumsanteilPPM: 600000, RenterEmails: []string{"b@example.com"}},
			}); err != nil {
				t.Fatalf("set units: %v", err)
			}
			if got := s.UnitCount("demo"); got != 2 {
				t.Fatalf("count = %d", got)
			}
			// Another tenant is untouched by a tenant-scoped write.
			if err := s.SetTenantUnits("other", []Unit{{ID: "x-1", Label: "X"}}); err != nil {
				t.Fatalf("set other: %v", err)
			}
			if got := s.UnitCount("demo"); got != 2 {
				t.Fatalf("demo count after other tenant write = %d", got)
			}
			if got := s.UnitCount("other"); got != 1 {
				t.Fatalf("other count = %d", got)
			}

			// Membership lookups.
			if got := s.UnitsForEmail("demo", "a@example.com"); len(got) != 1 || got[0].Relation != RoleOwner {
				t.Fatalf("owner membership = %+v", got)
			}
			if got := s.UnitsForEmail("demo", "b@example.com"); len(got) != 1 || got[0].Relation != RoleRenter {
				t.Fatalf("renter membership = %+v", got)
			}
			if got := s.UnitsForEmail("demo", "nobody@example.com"); len(got) != 0 {
				t.Fatalf("stranger membership = %+v", got)
			}
			members := s.MembersForUnit("demo", "top-1")
			if !members.Found || len(members.Owners) != 1 || members.Owners[0] != "a@example.com" {
				t.Fatalf("members = %+v", members)
			}
			if got := s.MembersForUnit("demo", "nope"); got.Found {
				t.Fatalf("unknown unit must not be found: %+v", got)
			}

			// Upsert: create a new unit.
			if dup, err := s.UpsertUnit("demo", "", Unit{ID: "top-3", Label: "Top 3"}); err != nil || dup {
				t.Fatalf("create: dup=%v err=%v", dup, err)
			}
			if got := s.UnitCount("demo"); got != 3 {
				t.Fatalf("count after create = %d", got)
			}
			// Creating the same ID again is a duplicate.
			if dup, err := s.UpsertUnit("demo", "", Unit{ID: "top-3", Label: "Dup"}); err != nil || !dup {
				t.Fatalf("duplicate create: dup=%v err=%v", dup, err)
			}
			if got := s.UnitCount("demo"); got != 3 {
				t.Fatalf("rejected duplicate must not add: %d", got)
			}
			// Editing a unit in place (same id) is allowed.
			if dup, err := s.UpsertUnit("demo", "top-3", Unit{ID: "top-3", Label: "Renamed label"}); err != nil || dup {
				t.Fatalf("in-place edit: dup=%v err=%v", dup, err)
			}
			// Renaming onto an existing id is a duplicate.
			if dup, err := s.UpsertUnit("demo", "top-3", Unit{ID: "top-1", Label: "Clash"}); err != nil || !dup {
				t.Fatalf("rename onto existing: dup=%v err=%v", dup, err)
			}
			// A real rename moves the unit.
			if dup, err := s.UpsertUnit("demo", "top-3", Unit{ID: "top-9", Label: "Top 9"}); err != nil || dup {
				t.Fatalf("rename: dup=%v err=%v", dup, err)
			}
			ids := map[string]bool{}
			for _, u := range s.ListTenant("demo") {
				ids[u.ID] = true
			}
			if ids["top-3"] || !ids["top-9"] {
				t.Fatalf("rename left the old id: %v", ids)
			}
			if got := s.UnitCount("demo"); got != 3 {
				t.Fatalf("rename must not change the count: %d", got)
			}

			// Delete.
			removed, unit, err := s.DeleteUnit("demo", "top-9")
			if err != nil || !removed || unit.ID != "top-9" {
				t.Fatalf("delete: removed=%v unit=%+v err=%v", removed, unit, err)
			}
			if got := s.UnitCount("demo"); got != 2 {
				t.Fatalf("count after delete = %d", got)
			}
			if removed, _, err := s.DeleteUnit("demo", "top-9"); err != nil || removed {
				t.Fatalf("second delete: removed=%v err=%v", removed, err)
			}

			// Billable weight is derived consistently.
			if got := s.BillableUnitWeight("demo"); got != BillableUnitWeight(s.ListTenant("demo")) {
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
	if err := jsonStore.SetTenantUnits("demo", []Unit{
		{ID: "top-1", Label: "Top 1", MiteigentumsanteilPPM: 500000, OwnerEmails: []string{"a@example.com"}},
		{ID: "top-2", Label: "Top 2", MiteigentumsanteilPPM: 500000},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	defer database.Close()
	sqlStore := NewSQLUnitStore(database)

	for i := 0; i < 2; i++ {
		if err := sqlStore.ImportUnits(jsonStore); err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
	}
	if got := sqlStore.UnitCount("demo"); got != 2 {
		t.Fatalf("imported %d units, want 2", got)
	}
	if got := sqlStore.UnitsForEmail("demo", "a@example.com"); len(got) != 1 {
		t.Fatalf("membership after import = %+v", got)
	}
	if got := sqlStore.BillableUnitWeight("demo"); got != jsonStore.BillableUnitWeight("demo") {
		t.Fatalf("billable weight drifted after import: %d vs %d", got, jsonStore.BillableUnitWeight("demo"))
	}
}
