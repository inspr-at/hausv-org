package energy_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/store"
)

// Each energy write HAUSV-779 moved onto For(tenant) stays visible to its own
// house and invisible to the other house's lane.
func TestEnergyWritesDenyTheOtherHouseLane(t *testing.T) {
	database, lanes := openEnergyLanes(t)
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	sqlStore := energy.NewSQLStore(lanes)
	for _, slug := range []string{"haus-a", "haus-b"} {
		if err := sqlStore.SaveProfile(energy.DefaultProfile(slug, now)); err != nil {
			t.Fatalf("save %s: %v", slug, err)
		}
	}
	if err := sqlStore.UpsertAsset(energy.Asset{
		ID: "asset-a", TenantSlug: "haus-a", Kind: "pv", Name: "PV", Confirmed: true,
	}); err != nil {
		t.Fatalf("asset: %v", err)
	}
	if err := sqlStore.UpsertMapping(energy.EntityMapping{
		ID: "mapping-a", TenantSlug: "haus-a", EntityID: "sensor.pv", AssetID: "asset-a",
		Metric: energy.MetricPVPower, DisplayName: "PV", Unit: "kW", Confirmed: true,
	}); err != nil {
		t.Fatalf("mapping: %v", err)
	}
	if err := sqlStore.PutInterval(energy.Interval{
		TenantSlug: "haus-a", StartsAt: now, Duration: 15 * time.Minute,
		ImportKWh: 0.5, AverageKW: 2, Quality: energy.QualityMeasured, Source: "home-assistant",
	}); err != nil {
		t.Fatalf("interval: %v", err)
	}
	inserted, err := sqlStore.PutImport(energy.ImportRecord{
		ID: "import-a", TenantSlug: "haus-a", Filename: "export.csv", SHA256: "sha-a",
		Format: "smart-meter-csv", Payload: []byte("timestamp;import_kwh\n"), ImportedAt: now,
	}, nil)
	if err != nil || !inserted {
		t.Fatalf("import inserted=%v err=%v", inserted, err)
	}
	if err := sqlStore.UpsertMaintenance(energy.MaintenancePlan{
		ID: "plan-a", TenantSlug: "haus-a", AssetID: "asset-a", Title: "Sichtprüfung",
		IntervalMonths: 12, NextDueAt: now.AddDate(0, 1, 0), Active: true,
	}); err != nil {
		t.Fatalf("maintenance: %v", err)
	}
	if err := sqlStore.UpsertMeasure(energy.Measure{
		ID: "measure-a", TenantSlug: "haus-a", IssueID: "issue-a", Title: "Lastspitze", Status: energy.MeasureCompleted,
	}); err != nil {
		t.Fatalf("measure: %v", err)
	}

	bound := sqlStore.ForTenant(store.TenantRef{ID: tenantIDOf(t, database, "haus-b"), Slug: "haus-b"})
	if _, _, err := bound.Profile("haus-a"); !errors.Is(err, energy.ErrTenantScopeMismatch) {
		t.Fatalf("bound house B reading house A = %v", err)
	}
	if err := bound.SaveProfile(energy.DefaultProfile("haus-a", now.Add(time.Hour))); !errors.Is(err, energy.ErrTenantScopeMismatch) {
		t.Fatalf("bound house B writing house A = %v", err)
	}
	if err := bound.UpsertAsset(energy.Asset{ID: "asset-a", TenantSlug: "haus-a", Kind: "pv", Confirmed: true}); !errors.Is(err, energy.ErrTenantScopeMismatch) {
		t.Fatalf("bound house B asset write = %v", err)
	}

	if dbtest.Backend() == appdb.BackendPostgres {
		lane := lanes.For(store.TenantRef{ID: tenantIDOf(t, database, "haus-b"), Slug: "haus-b"})
		assertEnergyLaneCount(t, lane, `SELECT count(*) FROM home_profiles WHERE tenant_slug=$1 AND home_key=$2`, "haus-a", "default")
		assertEnergyLaneMiss(t, lane, `UPDATE home_profiles SET household_name=$1 WHERE tenant_slug=$2`, "fremd", "haus-a")
		assertEnergyLaneCount(t, lane, `SELECT count(*) FROM energy_assets WHERE id=$1`, "asset-a")
		assertEnergyLaneMiss(t, lane, `UPDATE energy_assets SET name=$1 WHERE id=$2`, "fremd", "asset-a")
		assertEnergyLaneCount(t, lane, `SELECT count(*) FROM energy_entity_mappings WHERE entity_id=$1`, "sensor.pv")
		assertEnergyLaneMiss(t, lane, `DELETE FROM energy_entity_mappings WHERE entity_id=$1`, "sensor.pv")
		assertEnergyLaneCount(t, lane, `SELECT count(*) FROM energy_intervals WHERE tenant_slug=$1`, "haus-a")
		assertEnergyLaneMiss(t, lane, `DELETE FROM energy_intervals WHERE tenant_slug=$1`, "haus-a")
		assertEnergyLaneCount(t, lane, `SELECT count(*) FROM energy_imports WHERE sha256=$1`, "sha-a")
		assertEnergyLaneMiss(t, lane, `DELETE FROM energy_imports WHERE sha256=$1`, "sha-a")
		assertEnergyLaneCount(t, lane, `SELECT count(*) FROM energy_maintenance_plans WHERE id=$1`, "plan-a")
		assertEnergyLaneMiss(t, lane, `DELETE FROM energy_maintenance_plans WHERE id=$1`, "plan-a")
		assertEnergyLaneCount(t, lane, `SELECT count(*) FROM energy_measures WHERE issue_id=$1`, "issue-a")
		assertEnergyLaneMiss(t, lane, `DELETE FROM energy_measures WHERE issue_id=$1`, "issue-a")
	}

	profile, found, err := sqlStore.Profile("haus-a")
	if err != nil || !found || profile.HouseholdName == "fremd" {
		t.Fatalf("house A profile after the other lane = %+v found=%v err=%v", profile, found, err)
	}
	assets, err := sqlStore.ListAssets("haus-a")
	if err != nil || len(assets) != 1 || assets[0].Name == "fremd" {
		t.Fatalf("house A assets after the other lane = %+v err=%v", assets, err)
	}
	mappings, err := sqlStore.ListMappings("haus-a")
	if err != nil || len(mappings) != 1 {
		t.Fatalf("house A mappings = %+v err=%v", mappings, err)
	}
	intervals, err := sqlStore.ListIntervals("haus-a", now.Add(-time.Second), time.Time{})
	if err != nil || len(intervals) != 1 {
		t.Fatalf("house A intervals = %+v err=%v", intervals, err)
	}
	imports, err := sqlStore.ListImports("haus-a")
	if err != nil || len(imports) != 1 {
		t.Fatalf("house A imports = %+v err=%v", imports, err)
	}
	plans, err := sqlStore.ListMaintenance("haus-a")
	if err != nil || len(plans) != 1 {
		t.Fatalf("house A maintenance = %+v err=%v", plans, err)
	}
	measures, err := sqlStore.ListMeasures("haus-a")
	if err != nil || len(measures) != 1 {
		t.Fatalf("house A measures = %+v err=%v", measures, err)
	}
}

func tenantIDOf(t *testing.T, database *sql.DB, slug string) string {
	t.Helper()
	var id string
	if err := database.QueryRow(`SELECT tenant_id FROM tenant WHERE slug=$1`, slug).Scan(&id); err != nil {
		t.Fatalf("tenant id %s: %v", slug, err)
	}
	return id
}

func assertEnergyLaneCount(t *testing.T, lane appdb.Handle, query string, args ...any) {
	t.Helper()
	var n int
	if err := lane.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	if n != 0 {
		t.Fatalf("%s returned %d rows, want 0", query, n)
	}
}

func assertEnergyLaneMiss(t *testing.T, lane appdb.Handle, query string, args ...any) {
	t.Helper()
	result, err := lane.Exec(query, args...)
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("%s rows affected: %v", query, err)
	}
	if n != 0 {
		t.Fatalf("%s changed %d rows, want 0", query, n)
	}
}
