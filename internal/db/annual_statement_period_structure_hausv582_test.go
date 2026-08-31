package db

import (
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"
)

type migratedAnnualStatementCostType struct {
	Key           string
	Name          string
	Allocatable   bool
	AllocationKey string
	UpdatedAt     string
	UpdatedBy     string
}

func legacyDefaultAnnualStatementCostTypes(updatedAt, updatedBy string) []migratedAnnualStatementCostType {
	return []migratedAnnualStatementCostType{
		{Key: "gartenpflege", Name: "Gartenpflege", Allocatable: true, AllocationKey: "nutzwert", UpdatedAt: updatedAt, UpdatedBy: updatedBy},
		{Key: "gebaeudeversicherung", Name: "Gebäudeversicherung", Allocatable: true, AllocationKey: "nutzwert", UpdatedAt: updatedAt, UpdatedBy: updatedBy},
		{Key: "grundsteuer", Name: "Grundsteuer", Allocatable: true, AllocationKey: "nutzwert", UpdatedAt: updatedAt, UpdatedBy: updatedBy},
		{Key: "hausbetreuung", Name: "Hausbetreuung", Allocatable: true, AllocationKey: "nutzwert", UpdatedAt: updatedAt, UpdatedBy: updatedBy},
		{Key: "muellabfuhr", Name: "Müllabfuhr", Allocatable: true, AllocationKey: "nutzwert", UpdatedAt: updatedAt, UpdatedBy: updatedBy},
	}
}

var customizedAnnualStatementCostTypes = []migratedAnnualStatementCostType{
	{Key: "grundsteuer", Name: "Grundsteuer individuell", Allocatable: false, UpdatedAt: "2026-08-29T00:00:00Z", UpdatedBy: "custom@example.com"},
	{Key: "sonderkosten", Name: "Sonderkosten", Allocatable: true, AllocationKey: "personen", UpdatedAt: "2026-08-29T01:00:00Z", UpdatedBy: "custom@example.com"},
}

func TestAnnualStatementPeriodStructureMigrationBackfillsLegacyCatalogsSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "period-structure.db")
	database := openBeforeMigration(t, path, "0039_annual_statement_period_structure.sql")
	const emptyTenantID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	const customTenantID = "01BX5ZZKBKACTAV9WEVGEMMVRZ"
	if _, err := database.Exec(`INSERT INTO tenant(tenant_id,slug,created_at,updated_at)
		VALUES(?,'legacy-empty','2026-08-30T00:00:00Z','2026-08-30T00:00:00Z'),
		      (?,'legacy-custom','2026-08-29T00:00:00Z','2026-08-29T00:00:00Z')`, emptyTenantID, customTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO annual_statement_periods(
		tenant_slug,tenant_id,year,starts_on,ends_on,updated_at,updated_by)
		VALUES('legacy-empty',?,2026,'2026-01-01','2026-12-31','2026-08-30T00:00:00Z','legacy@example.com'),
		      ('legacy-empty',?,2025,'2025-01-01','2025-12-31','2025-08-30T00:00:00Z','older@example.com'),
		      ('legacy-custom',?,2026,'2026-01-01','2026-12-31','2026-08-29T00:00:00Z','period@example.com')`, emptyTenantID, emptyTenantID, customTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO annual_statement_cost_types(
		tenant_slug,tenant_id,key,name,allocatable,allocation_key,updated_at,updated_by)
		VALUES('legacy-custom',?,'grundsteuer','Grundsteuer individuell',0,'','2026-08-29T00:00:00Z','custom@example.com'),
		      ('legacy-custom',?,'sonderkosten','Sonderkosten',1,'personen','2026-08-29T01:00:00Z','custom@example.com')`, customTenantID, customTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO units(tenant_slug,tenant_id,id,data) VALUES(
		'legacy-empty',?,'top-1','{"id":"top-1","tenant":"legacy-empty","label":"Top 1","miteigentumsanteil":1000000,"usable_area_m2_hundredths":7500,"usable_area_recorded":true,"persons":2,"persons_recorded":true}')`, emptyTenantID); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	database, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	assertMigratedAnnualStatementCostTypes(t, database, emptyTenantID, 2026, legacyDefaultAnnualStatementCostTypes("2026-08-30T00:00:00Z", "legacy@example.com"))
	assertMigratedAnnualStatementCostTypes(t, database, emptyTenantID, 2025, legacyDefaultAnnualStatementCostTypes("2025-08-30T00:00:00Z", "older@example.com"))
	assertMigratedAnnualStatementCostTypes(t, database, customTenantID, 2026, customizedAnnualStatementCostTypes)
	var emptyCatalogRows int
	if err := database.QueryRow(`SELECT count(*) FROM annual_statement_cost_types WHERE tenant_id=?`, emptyTenantID).Scan(&emptyCatalogRows); err != nil {
		t.Fatal(err)
	}
	if emptyCatalogRows != 0 {
		t.Fatalf("migration persisted %d default rows into the empty mutable catalog", emptyCatalogRows)
	}
	var mea, area, areaRecorded, persons, personsRecorded int
	if err := database.QueryRow(`SELECT miteigentumsanteil_ppm,usable_area_m2_hundredths,usable_area_recorded,persons,persons_recorded
		FROM annual_statement_period_unit_bases WHERE tenant_id=? AND period_year=2026 AND unit_id='top-1'`, emptyTenantID).
		Scan(&mea, &area, &areaRecorded, &persons, &personsRecorded); err != nil {
		t.Fatal(err)
	}
	if mea != 1_000_000 || area != 7_500 || areaRecorded != 1 || persons != 2 || personsRecorded != 1 {
		t.Fatalf("backfilled unit basis = mea=%d area=%d/%d persons=%d/%d", mea, area, areaRecorded, persons, personsRecorded)
	}
}

func assertMigratedAnnualStatementCostTypes(t *testing.T, database *sql.DB, tenantID string, year int, want []migratedAnnualStatementCostType) {
	t.Helper()
	rows, err := database.Query(`SELECT key,name,allocatable,allocation_key,updated_at,updated_by
		FROM annual_statement_period_cost_types WHERE tenant_id=? AND period_year=? ORDER BY key`, tenantID, year)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := []migratedAnnualStatementCostType{}
	for rows.Next() {
		var item migratedAnnualStatementCostType
		if err := rows.Scan(&item.Key, &item.Name, &item.Allocatable, &item.AllocationKey, &item.UpdatedAt, &item.UpdatedBy); err != nil {
			t.Fatal(err)
		}
		got = append(got, item)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("period cost types for %s/%d = %+v, want %+v", tenantID, year, got, want)
	}
}
