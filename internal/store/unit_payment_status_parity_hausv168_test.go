package store

import (
	"path/filepath"
	"testing"

	"github.com/markus-barta/hausv-org/internal/db"
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
			database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatalf("db open: %v", err)
			}
			t.Cleanup(func() { database.Close() })
			return NewSQLUnitPaymentStatusStore(database)
		},
	}

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s := build(t)

			if _, ok := s.Get("jhw22", "w-01"); ok {
				t.Fatal("unknown (tenant,unit) must not be found")
			}
			if _, err := s.Set(UnitPaymentStatus{TenantSlug: "", UnitID: "w-01", Status: "bezahlt"}); err == nil {
				t.Fatal("missing tenant must error")
			}

			set, err := s.Set(UnitPaymentStatus{TenantSlug: "jhw22", UnitID: "W 01", Status: "bezahlt", UpdatedBy: "admin@example.com"})
			if err != nil {
				t.Fatalf("set: %v", err)
			}
			if set.Status != "bezahlt" || set.UpdatedAt.IsZero() {
				t.Fatalf("set returned %+v", set)
			}

			got, ok := s.Get("jhw22", "w-01") // normalized unit lookup
			if !ok || got.Status != "bezahlt" || got.UpdatedBy != "admin@example.com" {
				t.Fatalf("get mismatch: %+v ok=%v", got, ok)
			}

			// Upsert same (tenant,unit): status changes, still one row.
			if _, err := s.Set(UnitPaymentStatus{TenantSlug: "jhw22", UnitID: "w-01", Status: "teilbezahlt"}); err != nil {
				t.Fatalf("upsert: %v", err)
			}
			// A second unit + a different tenant.
			_, _ = s.Set(UnitPaymentStatus{TenantSlug: "jhw22", UnitID: "w-02", Status: "offen"})
			_, _ = s.Set(UnitPaymentStatus{TenantSlug: "other", UnitID: "w-01", Status: "bezahlt"})

			list := s.ListTenant("jhw22")
			if len(list) != 2 {
				t.Fatalf("ListTenant(jhw22) = %d rows, want 2 (tenant-scoped): %+v", len(list), list)
			}
			byUnit := map[string]string{}
			for _, it := range list {
				byUnit[it.UnitID] = it.Status
			}
			if byUnit["w-01"] != "teilbezahlt" || byUnit["w-02"] != "offen" {
				t.Fatalf("ListTenant content mismatch: %+v", byUnit)
			}
		})
	}
}

func TestSQLUnitPaymentImportFromJSON(t *testing.T) {
	jsonStore, err := NewUnitPaymentStatusStore(filepath.Join(t.TempDir(), "ups.json"))
	if err != nil {
		t.Fatalf("json store: %v", err)
	}
	_, _ = jsonStore.Set(UnitPaymentStatus{TenantSlug: "jhw22", UnitID: "w-01", Status: "bezahlt"})
	_, _ = jsonStore.Set(UnitPaymentStatus{TenantSlug: "jhw22", UnitID: "w-02", Status: "offen"})

	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	defer database.Close()
	sqlStore := NewSQLUnitPaymentStatusStore(database)

	// Newer SQLite write survives re-import.
	if _, err := sqlStore.Set(UnitPaymentStatus{TenantSlug: "jhw22", UnitID: "w-01", Status: "ueberfaellig"}); err != nil {
		t.Fatalf("pre-set: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := sqlStore.ImportStatuses(jsonStore); err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
	}
	if got, _ := sqlStore.Get("jhw22", "w-01"); got.Status != "ueberfaellig" {
		t.Fatalf("import clobbered newer SQLite write: %+v", got)
	}
	if got, ok := sqlStore.Get("jhw22", "w-02"); !ok || got.Status != "offen" {
		t.Fatalf("import missed json-only row: %+v ok=%v", got, ok)
	}
}
