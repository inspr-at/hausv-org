package store

import (
	"path/filepath"
	"testing"
)

func TestHAUSV619ReplaceTenantUnitsKeepsOtherTenants(t *testing.T) {
	path := filepath.Join(t.TempDir(), "units.json")
	units, err := NewUnitStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := units.ReplaceTenantUnits(t.Context(), "other", []Unit{{Label: "Top 1"}}); err != nil {
		t.Fatalf("seed other: %v", err)
	}
	if err := units.ReplaceTenantUnits(t.Context(), "demo", []Unit{{Label: "Top 1", UnitType: "Wohnung", OwnerEmails: []string{"OWNER@EXAMPLE.COM"}}}); err != nil {
		t.Fatalf("seed demo: %v", err)
	}
	if err := units.ReplaceTenantUnits(t.Context(), "demo", []Unit{{Label: "Stellplatz 1", UnitType: "Stellplatz", RenterEmails: []string{"renter@example.com"}}}); err != nil {
		t.Fatalf("replace demo: %v", err)
	}
	reloaded, err := NewUnitStore(path)
	if err != nil {
		t.Fatal(err)
	}
	for tenant, want := range map[string][]string{
		"demo":  {"stellplatz-1"},
		"other": {"top-1"},
	} {
		repository, ok := BindUnitRepository(reloaded, testTenantRef(tenant))
		if !ok {
			t.Fatalf("bind %s", tenant)
		}
		got := repository.List()
		if len(got) != len(want) || got[0].ID != want[0] {
			t.Errorf("%s units = %#v, want %v", tenant, got, want)
		}
	}
}
