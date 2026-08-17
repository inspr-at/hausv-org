package store

import (
	"path/filepath"
	"testing"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func TestUnitPaymentStatusStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) UnitPaymentStatusStorage{
		"json": func(t *testing.T) UnitPaymentStatusStorage {
			s, err := NewUnitPaymentStatusStore(filepath.Join(t.TempDir(), "ups.json"))
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) UnitPaymentStatusStorage {
			database := dbtest.Open(t)
			t.Cleanup(func() { database.Close() })
			return NewSQLUnitPaymentStatusStore(database)
		},
	}

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			storage := build(t)
			s, ok := BindUnitPaymentStatusRepository(storage, testTenantRef("demo"))
			if !ok {
				t.Fatal("bind demo repository")
			}
			other, ok := BindUnitPaymentStatusRepository(storage, testTenantRef("other"))
			if !ok {
				t.Fatal("bind other repository")
			}
			if unscoped, ok := BindUnitPaymentStatusRepository(storage, testTenantRef("")); ok || unscoped != nil {
				t.Fatal("empty tenant must not produce a repository")
			}

			if _, ok := s.Get("w-01"); ok {
				t.Fatal("unknown (tenant,unit) must not be found")
			}

			set, err := s.Set(UnitPaymentStatus{TenantSlug: "other", UnitID: "W 01", Status: "bezahlt", UpdatedBy: "admin@example.com"})
			if err != nil {
				t.Fatalf("set: %v", err)
			}
			if set.Status != "bezahlt" || set.UpdatedAt.IsZero() {
				t.Fatalf("set returned %+v", set)
			}

			got, ok := s.Get("w-01") // normalized unit lookup
			if !ok || got.Status != "bezahlt" || got.UpdatedBy != "admin@example.com" {
				t.Fatalf("get mismatch: %+v ok=%v", got, ok)
			}

			// Upsert same (tenant,unit): status changes, still one row.
			if _, err := s.Set(UnitPaymentStatus{TenantSlug: "demo", UnitID: "w-01", Status: "teilbezahlt"}); err != nil {
				t.Fatalf("upsert: %v", err)
			}
			// A second unit + a different tenant.
			_, _ = s.Set(UnitPaymentStatus{TenantSlug: "demo", UnitID: "w-02", Status: "offen"})
			_, _ = other.Set(UnitPaymentStatus{UnitID: "w-01", Status: "bezahlt"})

			list := s.List()
			if len(list) != 2 {
				t.Fatalf("ListTenant(demo) = %d rows, want 2 (tenant-scoped): %+v", len(list), list)
			}
			byUnit := map[string]string{}
			for _, it := range list {
				byUnit[it.UnitID] = it.Status
			}
			if byUnit["w-01"] != "teilbezahlt" || byUnit["w-02"] != "offen" {
				t.Fatalf("ListTenant content mismatch: %+v", byUnit)
			}
			if got, ok := other.Get("w-01"); !ok || got.Status != "bezahlt" {
				t.Fatalf("other tenant record missing: %+v ok=%v", got, ok)
			}
		})
	}
}

func TestSQLUnitPaymentImportFromJSON(t *testing.T) {
	jsonStore, err := NewUnitPaymentStatusStore(filepath.Join(t.TempDir(), "ups.json"))
	if err != nil {
		t.Fatalf("json store: %v", err)
	}
	jsonRepository, _ := BindUnitPaymentStatusRepository(jsonStore, testTenantRef("demo"))
	_, _ = jsonRepository.Set(UnitPaymentStatus{UnitID: "w-01", Status: "bezahlt"})
	_, _ = jsonRepository.Set(UnitPaymentStatus{UnitID: "w-02", Status: "offen"})

	database := dbtest.Open(t)
	defer database.Close()
	sqlStore := NewSQLUnitPaymentStatusStore(database)
	sqlRepository, _ := BindUnitPaymentStatusRepository(sqlStore, testTenantRef("demo"))

	// Newer SQLite write survives re-import.
	if _, err := sqlRepository.Set(UnitPaymentStatus{UnitID: "w-01", Status: "ueberfaellig"}); err != nil {
		t.Fatalf("pre-set: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := sqlStore.ImportStatuses(jsonStore); err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
	}
	if got, _ := sqlRepository.Get("w-01"); got.Status != "ueberfaellig" {
		t.Fatalf("import clobbered newer SQLite write: %+v", got)
	}
	if got, ok := sqlRepository.Get("w-02"); !ok || got.Status != "offen" {
		t.Fatalf("import missed json-only row: %+v ok=%v", got, ok)
	}
}
