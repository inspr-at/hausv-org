package energy_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/energy"
)

// Every energy table carries a tenant_id column and is covered by the
// PostgreSQL row-level-security policy, but until now nothing here wrote one:
// every column list was slug-only, so every row the product created after
// migration 0033 had a NULL identity.
//
// A NULL there is not a cosmetic gap. On PostgreSQL such a row was visible to
// every tenant under the old policy, and once the boot-time completeness check
// runs, a single one of them refuses the boot outright. So this test exercises
// every SQL writer in this package and then asks the database — rather than the
// code — whether an identity was recorded.
//
// Where it is blind: it proves the column is POPULATED, not that the value is
// the right tenant's. The energy package still filters its reads on tenant_slug
// (its Storage API takes a bare slug and has no tenant reference to filter by),
// so a wrong id here would not surface in a read. That is the follow-up this
// test is meant to make visible rather than hide.
func TestEveryEnergyWriteRecordsATenantIdentity(t *testing.T) {
	database, err := appdb.Open(filepath.Join(t.TempDir(), "energy-tenant-id.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	store := energy.NewSQLStore(database)

	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	const tenant = "haus-a"

	if err := store.SaveProfile(energy.DefaultProfile(tenant, now)); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	assetID := energy.StableAssetID(tenant, "pv")
	if err := store.UpsertAsset(energy.Asset{
		ID: assetID, TenantSlug: tenant, Kind: "pv", Name: "PV", Confirmed: true,
	}); err != nil {
		t.Fatalf("save asset: %v", err)
	}
	if err := store.UpsertMapping(energy.EntityMapping{
		ID: "mapping-a", TenantSlug: tenant, EntityID: "sensor.pv_power", AssetID: assetID,
		Metric: energy.MetricPVPower, DisplayName: "PV-Leistung", Unit: "kW", Confirmed: true,
	}); err != nil {
		t.Fatalf("save mapping: %v", err)
	}
	if err := store.PutInterval(energy.Interval{
		TenantSlug: tenant, StartsAt: now, Duration: 15 * time.Minute,
		ImportKWh: 0.5, AverageKW: 2, Quality: energy.QualityMeasured, Source: "home-assistant",
	}); err != nil {
		t.Fatalf("put interval: %v", err)
	}
	inserted, err := store.PutImport(
		energy.ImportRecord{
			ID: "import-a", TenantSlug: tenant, Filename: "export.csv", SHA256: "abc123",
			Format: "smart-meter-csv", Payload: []byte("timestamp;import_kwh\n"), ImportedAt: now,
		},
		[]energy.Interval{{
			TenantSlug: tenant, StartsAt: now.Add(time.Hour), Duration: 15 * time.Minute,
			ImportKWh: 0.25, AverageKW: 1, Quality: energy.QualityMeasured, Source: "smart-meter",
		}},
	)
	if err != nil || !inserted {
		t.Fatalf("put import inserted=%v err=%v", inserted, err)
	}
	completed := now.Add(-24 * time.Hour)
	if err := store.UpsertMaintenance(energy.MaintenancePlan{
		ID: "maintenance-a", TenantSlug: tenant, AssetID: assetID, Title: "PV-Sichtprüfung",
		IntervalMonths: 12, LastCompletedAt: &completed, NextDueAt: now.AddDate(0, 1, 0), Active: true,
	}); err != nil {
		t.Fatalf("save maintenance: %v", err)
	}
	if err := store.SaveTariffAssessment(energy.TariffAssessment{
		ID: "tariff-a", TenantSlug: tenant, AssessmentMonth: "2026-07", ProfileID: "at-grid-power",
		ProfileVersion: "draft-2027-v1", ProfileStatus: "draft", SourceURL: "https://example.invalid/rules",
		PeakKW: 8.25, BilledKW: 8.25, AnnualPowerEUR: 99, DataQuality: energy.QualityMeasured, CreatedAt: now,
	}); err != nil {
		t.Fatalf("save tariff assessment: %v", err)
	}
	if err := store.UpsertMeasure(energy.Measure{
		ID: "measure-a", TenantSlug: tenant, IssueID: "issue-a", RecommendationID: "peak",
		Title: "Lastspitze glätten", Status: energy.MeasureCompleted,
	}); err != nil {
		t.Fatalf("save measure: %v", err)
	}

	// The table list is DERIVED, not written down. A literal could be shrunk in
	// the same edit that broke a writer: binding any(nil) for tenant_id in
	// SaveTariffAssessment and deleting "energy_tariff_assessments" from the
	// literal left the whole SQLite suite green, and the AST audit next door
	// missed it too because that INSERT carries no ON CONFLICT clause. Asking the
	// package which tables it writes, and the database which of those are
	// tenant-scoped, removes the place the entry could be deleted from.
	tables := tenantScopedTablesWrittenHere(t, database)
	if len(tables) < 8 {
		t.Fatalf("only %d tenant-scoped INSERT targets were found in this package (%v) — "+
			"the probe is broken, not the package", len(tables), tables)
	}
	for _, table := range tables {
		var rows, withIdentity int
		if err := database.QueryRow(
			`SELECT count(*), count(tenant_id) FROM `+table).Scan(&rows, &withIdentity); err != nil {
			t.Fatalf("inspect %s: %v", table, err)
		}
		if rows == 0 {
			// Without this the loop would report success for a table the
			// fixture never actually wrote to.
			t.Errorf("%s has no rows — the fixture does not exercise this writer", table)
			continue
		}
		if withIdentity != rows {
			t.Errorf("%s: %d of %d rows have no tenant identity", table, rows-withIdentity, rows)
		}
	}
	assertIdentityMatchesSlug(t, database, tenant, tables)
}

// insertedTable pulls the target table out of an INSERT statement.
var insertedTable = regexp.MustCompile(`(?is)INSERT\s+INTO\s+([a-z_]+)`)

// tenantScopedTablesWrittenHere is the closure check the coverage literal could
// not be: it reads this package's own SQL for INSERT targets and keeps the ones
// the database says carry a tenant_id column.
//
// Two sources, neither of them a list a person maintains. Adding a writer to
// this package adds its table here automatically; removing a table from
// coverage requires removing the writer, which is the change that would have
// made the coverage unnecessary in the first place.
//
// Where it is blind: it scans the source as TEXT, so an INSERT whose table name
// is assembled from a variable at runtime is invisible to it, and a table name
// written inside a comment would be counted as a write. It also says nothing
// about tables OTHER packages write — internal/store asks the same question of
// itself, and the two answers partition the tenant-scoped tables between them.
func tenantScopedTablesWrittenHere(t *testing.T, database *sql.DB) []string {
	t.Helper()
	scoped := map[string]bool{}
	rows, err := database.Query(`SELECT m.name FROM sqlite_master m JOIN pragma_table_info(m.name) p
		WHERE m.type='table' AND p.name='tenant_id' AND m.name <> 'tenant'`)
	if err != nil {
		t.Fatalf("read the tenant-scoped tables from the schema: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		scoped[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table names: %v", err)
	}

	found := map[string]bool{}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, match := range insertedTable.FindAllStringSubmatch(string(source), -1) {
			table := strings.ToLower(match[1])
			if scoped[table] {
				found[table] = true
			}
		}
	}
	out := make([]string, 0, len(found))
	for table := range found {
		out = append(out, table)
	}
	sort.Strings(out)
	return out
}

func assertIdentityMatchesSlug(t *testing.T, database *sql.DB, slug string, tables []string) {
	t.Helper()
	var tenantID string
	if err := database.QueryRow(`SELECT tenant_id FROM tenant WHERE slug=$1`, slug).Scan(&tenantID); err != nil {
		t.Fatalf("the writers must have minted an identity for %s: %v", slug, err)
	}
	for _, table := range tables {
		var mismatched int
		if err := database.QueryRow(
			`SELECT count(*) FROM `+table+` WHERE tenant_slug=$1 AND tenant_id<>$2`, slug, tenantID).Scan(&mismatched); err != nil {
			t.Fatalf("inspect %s: %v", table, err)
		}
		if mismatched != 0 {
			t.Errorf("%s: %d rows carry an identity that is not %s's", table, mismatched, slug)
		}
		// The symmetric question. Without it the check above is vacuous: a
		// writer that stopped binding tenant_slug would match zero rows and
		// report zero mismatches, and the rollback window would be closed with
		// nothing saying so.
		var rows, addressable int
		if err := database.QueryRow(
			`SELECT count(*), count(CASE WHEN tenant_slug=$1 THEN 1 END) FROM `+table, slug).
			Scan(&rows, &addressable); err != nil {
			t.Fatalf("inspect %s: %v", table, err)
		}
		if addressable != rows {
			t.Errorf("%s: %d of %d rows are not addressable by tenant_slug=%q — "+
				"the rollback window is closed and nothing said so", table, rows-addressable, rows, slug)
		}
	}
}
