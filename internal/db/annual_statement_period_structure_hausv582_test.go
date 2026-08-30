package db

import (
	"path/filepath"
	"testing"
)

func TestAnnualStatementPeriodStructureMigrationBackfillsExistingPeriods(t *testing.T) {
	path := filepath.Join(t.TempDir(), "period-structure.db")
	database := openBeforeMigration(t, path, "0039_annual_statement_period_structure.sql")
	const tenantID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	if _, err := database.Exec(`INSERT INTO tenant(tenant_id,slug,created_at,updated_at)
		VALUES(?,'demo','2026-08-30T00:00:00Z','2026-08-30T00:00:00Z')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO annual_statement_periods(
		tenant_slug,tenant_id,year,starts_on,ends_on,updated_at,updated_by)
		VALUES('demo',?,2026,'2026-01-01','2026-12-31','2026-08-30T00:00:00Z','manager@example.com')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO annual_statement_cost_types(
		tenant_slug,tenant_id,key,name,allocatable,allocation_key,updated_at,updated_by)
		VALUES('demo',?,'grundsteuer','Grundsteuer',1,'nutzwert','2026-08-30T00:00:00Z','manager@example.com')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO units(tenant_slug,tenant_id,id,data) VALUES(
		'demo',?,'top-1','{"id":"top-1","tenant":"demo","label":"Top 1","miteigentumsanteil":1000000,"usable_area_m2_hundredths":7500,"usable_area_recorded":true,"persons":2,"persons_recorded":true}')`, tenantID); err != nil {
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
	var name, allocationKey string
	if err := database.QueryRow(`SELECT name,allocation_key FROM annual_statement_period_cost_types
		WHERE tenant_id=? AND period_year=2026 AND key='grundsteuer'`, tenantID).Scan(&name, &allocationKey); err != nil {
		t.Fatal(err)
	}
	if name != "Grundsteuer" || allocationKey != "nutzwert" {
		t.Fatalf("backfilled cost type = %q/%q", name, allocationKey)
	}
	var mea, area, areaRecorded, persons, personsRecorded int
	if err := database.QueryRow(`SELECT miteigentumsanteil_ppm,usable_area_m2_hundredths,usable_area_recorded,persons,persons_recorded
		FROM annual_statement_period_unit_bases WHERE tenant_id=? AND period_year=2026 AND unit_id='top-1'`, tenantID).
		Scan(&mea, &area, &areaRecorded, &persons, &personsRecorded); err != nil {
		t.Fatal(err)
	}
	if mea != 1_000_000 || area != 7_500 || areaRecorded != 1 || persons != 2 || personsRecorded != 1 {
		t.Fatalf("backfilled unit basis = mea=%d area=%d/%d persons=%d/%d", mea, area, areaRecorded, persons, personsRecorded)
	}
}
