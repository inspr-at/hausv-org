package db

import (
	"database/sql"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestOpenRunsBaselineMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	database, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	// The baseline table exists and round-trips.
	if _, err := database.Exec("INSERT INTO app_meta(key, value) VALUES('schema', 'baseline')"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	var got string
	if err := database.QueryRow("SELECT value FROM app_meta WHERE key = 'schema'").Scan(&got); err != nil {
		t.Fatalf("select: %v", err)
	}
	if got != "baseline" {
		t.Fatalf("value = %q, want baseline", got)
	}

	// The baseline migration is recorded exactly once (robust to later
	// migrations being added over time).
	var baseline int
	if err := database.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = '0001_baseline.sql'").Scan(&baseline); err != nil {
		t.Fatalf("count baseline: %v", err)
	}
	if baseline != 1 {
		t.Fatalf("baseline recorded %d times, want 1", baseline)
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	first, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	if _, err := first.Exec("INSERT INTO app_meta(key, value) VALUES('k', 'v')"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	first.Close()

	// Re-opening runs the migration runner again: it must be a no-op (no error,
	// no duplicate migration rows) and must not touch existing data.
	second, err := Open(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer second.Close()

	// Reopening must not duplicate any migration row (idempotent), regardless of
	// how many migrations exist.
	var maxPerVersion int
	if err := second.QueryRow("SELECT COALESCE(MAX(c), 0) FROM (SELECT COUNT(*) c FROM schema_migrations GROUP BY version)").Scan(&maxPerVersion); err != nil {
		t.Fatalf("duplicate check: %v", err)
	}
	if maxPerVersion != 1 {
		t.Fatalf("a migration was applied more than once after reopen (max %d)", maxPerVersion)
	}
	var value string
	if err := second.QueryRow("SELECT value FROM app_meta WHERE key = 'k'").Scan(&value); err != nil || value != "v" {
		t.Fatalf("existing data lost after reopen: value=%q err=%v", value, err)
	}
}

func TestMultiStatementExecRunsEveryStatement(t *testing.T) {
	// The migration runner execs each file as one string; modernc.org/sqlite must
	// run EVERY statement, else a multi-statement migration would silently skip
	// its later statements. This also covers a ";" inside a comment.
	database, err := Open(filepath.Join(t.TempDir(), "multi.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()
	if _, err := database.Exec(
		"-- a comment; with a semicolon\nCREATE TABLE t1(x INTEGER);\nCREATE TABLE t2(y INTEGER);",
	); err != nil {
		t.Fatalf("multi-statement exec: %v", err)
	}
	for _, tbl := range []string{"t1", "t2"} {
		var n int
		if err := database.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tbl,
		).Scan(&n); err != nil || n != 1 {
			t.Fatalf("table %s missing after multi-statement exec (n=%d err=%v)", tbl, n, err)
		}
	}
}

func TestOpenRequiresPath(t *testing.T) {
	if _, err := Open("   "); err == nil {
		t.Fatal("empty path must error")
	}
}

func TestForeignKeysEnforced(t *testing.T) {
	// Proves the foreign_keys pragma is live on the connection (it is per-conn,
	// which is why we set it via the DSN, not a one-off Exec).
	path := filepath.Join(t.TempDir(), "fk.db")
	database, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	if _, err := database.Exec(`CREATE TABLE parent(id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	if _, err := database.Exec(`CREATE TABLE child(id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parent(id))`); err != nil {
		t.Fatalf("create child: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO child(id, parent_id) VALUES (1, 999)`); err == nil {
		t.Fatal("insert violating a foreign key must fail with foreign_keys ON")
	}
}

func TestHomeProfileUnitMigrationPreservesExistingRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "home-profile-unit.db")
	database, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	now := "2026-07-29T15:00:00Z"
	if _, err := database.Exec(`INSERT INTO home_profiles(tenant_slug,household_name,created_at,updated_at)
		VALUES
			('legacy-home','Bestehendes Zuhause',?,?),
			('ambiguous-home','Mehrdeutiges Zuhause',?,?)`,
		now, now, now, now,
	); err != nil {
		t.Fatalf("insert existing profiles: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO units(tenant_slug,id,data) VALUES
		('legacy-home','einheit-12','{"id":"einheit-12","tenant":"legacy-home","label":"Einheit 12","unit_type":"residential","miteigentumsanteil":10}'),
		('ambiguous-home','top-1','{"id":"top-1","tenant":"ambiguous-home","label":"Top 1","unit_type":"residential","miteigentumsanteil":10}'),
		('ambiguous-home','top-2','{"id":"top-2","tenant":"ambiguous-home","label":"Top 2","unit_type":"residential","miteigentumsanteil":10}')`); err != nil {
		t.Fatalf("insert existing units: %v", err)
	}
	if _, err := database.Exec(`ALTER TABLE home_profiles DROP COLUMN unit_id`); err != nil {
		t.Fatalf("restore pre-migration schema: %v", err)
	}
	if _, err := database.Exec(`DELETE FROM schema_migrations WHERE version='0024_home_profile_unit.sql'`); err != nil {
		t.Fatalf("reset migration marker: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	database, err = Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer database.Close()

	var householdName, unitID string
	if err := database.QueryRow(`SELECT household_name,unit_id FROM home_profiles WHERE tenant_slug='legacy-home'`).
		Scan(&householdName, &unitID); err != nil {
		t.Fatalf("load migrated profile: %v", err)
	}
	if householdName != "Bestehendes Zuhause" || unitID != "einheit-12" {
		t.Fatalf("migrated profile: household_name=%q unit_id=%q", householdName, unitID)
	}
	if err := database.QueryRow(`SELECT unit_id FROM home_profiles WHERE tenant_slug='ambiguous-home'`).
		Scan(&unitID); err != nil {
		t.Fatalf("load ambiguous migrated profile: %v", err)
	}
	if unitID != "" {
		t.Fatalf("ambiguous migrated profile was silently linked to %q", unitID)
	}

	var notNull int
	var defaultValue string
	if err := database.QueryRow(`SELECT "notnull",dflt_value FROM pragma_table_info('home_profiles') WHERE name='unit_id'`).
		Scan(&notNull, &defaultValue); err != nil {
		t.Fatalf("inspect unit_id column: %v", err)
	}
	if notNull != 1 || defaultValue != "''" {
		t.Fatalf("unit_id schema: notnull=%d default=%q", notNull, defaultValue)
	}
}

func TestConsumptionMappingMigrationOnlyCorrectsLegacyBatteryHeuristic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "energy-mapping.db")
	database, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	now := "2026-07-29T10:00:00Z"
	if _, err := database.Exec(`INSERT INTO home_profiles(tenant_slug,created_at,updated_at) VALUES('demo',?,?)`, now, now); err != nil {
		t.Fatalf("insert profile: %v", err)
	}
	for _, item := range []struct {
		id, entity, name string
	}{
		{"legacy-load", "sensor.sonnenbatterie_state_consumption_current", "Home Current Consumption"},
		{"real-battery", "sensor.battery_discharge_power", "Battery Discharge Power"},
	} {
		if _, err := database.Exec(`INSERT INTO energy_entity_mappings
			(id,tenant_slug,entity_id,metric,display_name,unit,device_class,confirmed,created_at,updated_at,asset_id)
			VALUES(?,'demo',?,'battery-power',?,'W','power',1,?,?, '')`,
			item.id, item.entity, item.name, now, now,
		); err != nil {
			t.Fatalf("insert mapping %s: %v", item.id, err)
		}
	}
	if _, err := database.Exec(`DELETE FROM schema_migrations WHERE version='0023_energy_consumption_mapping.sql'`); err != nil {
		t.Fatalf("reset migration marker: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	database, err = Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer database.Close()
	var legacy, battery string
	if err := database.QueryRow(`SELECT metric FROM energy_entity_mappings WHERE id='legacy-load'`).Scan(&legacy); err != nil {
		t.Fatalf("legacy query: %v", err)
	}
	if err := database.QueryRow(`SELECT metric FROM energy_entity_mappings WHERE id='real-battery'`).Scan(&battery); err != nil {
		t.Fatalf("battery query: %v", err)
	}
	if legacy != "load-power" || battery != "battery-power" {
		t.Fatalf("migration metrics: legacy=%q battery=%q", legacy, battery)
	}
}

func TestEnergyHomeScopeMigrationPreservesLegacyDefaultHome(t *testing.T) {
	path := filepath.Join(t.TempDir(), "home-scope.db")
	database := openBeforeMigration(t, path, "0026_energy_home_scope.sql")
	now := "2026-08-13T09:00:00Z"
	statements := []string{
		`INSERT INTO home_profiles(tenant_slug,unit_id,household_name,agreed_power_kw,created_at,updated_at) VALUES('demo','einheit-12','Dachwohnung',15,?,?)`,
		`INSERT INTO energy_assets(id,tenant_slug,kind,name,created_at,updated_at) VALUES('asset-demo-battery','demo','battery','Speicher',?,?)`,
		`INSERT INTO energy_entity_mappings(id,tenant_slug,entity_id,asset_id,metric,created_at,updated_at) VALUES('mapping-1','demo','sensor.battery','asset-demo-battery','battery-power',?,?)`,
		`INSERT INTO energy_intervals(tenant_slug,starts_at,import_kwh,average_kw,created_at) VALUES('demo',?,1.25,5,?)`,
		`INSERT INTO energy_imports(id,tenant_slug,filename,sha256,format,payload,imported_at) VALUES('import-1','demo','legacy.csv','sha-1','csv',x'01',?)`,
		`INSERT INTO energy_maintenance_plans(id,tenant_slug,asset_id,title,interval_months,next_due_at,created_at,updated_at) VALUES('maintenance-1','demo','asset-demo-battery','Wartung',12,?,?,?)`,
		`INSERT INTO energy_tariff_assessments(id,tenant_slug,assessment_month,profile_id,profile_version,profile_status,source_url,peak_kw,billed_kw,annual_power_eur,data_quality,created_at) VALUES('tariff-1','demo','2026-08','p','1','active','https://example.test',5,5,100,'measured',?)`,
		`INSERT INTO energy_measures(id,tenant_slug,issue_id,recommendation_id,title,created_at,updated_at) VALUES('measure-1','demo','issue-1','rec-1','Maßnahme',?,?)`,
	}
	for _, statement := range statements {
		args := make([]any, strings.Count(statement, "?"))
		for i := range args {
			args[i] = now
		}
		if _, err := database.Exec(statement, args...); err != nil {
			t.Fatalf("seed legacy row: %v\n%s", err, statement)
		}
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	database, err := Open(path)
	if err != nil {
		t.Fatalf("apply home-scope migration: %v", err)
	}
	defer database.Close()
	for _, table := range []string{
		"home_profiles", "energy_assets", "energy_entity_mappings", "energy_intervals",
		"energy_imports", "energy_maintenance_plans", "energy_tariff_assessments", "energy_measures",
	} {
		var count int
		if err := database.QueryRow(`SELECT COUNT(*) FROM ` + table + ` WHERE tenant_slug='demo' AND home_key='default'`).Scan(&count); err != nil || count != 1 {
			t.Fatalf("%s default-home rows=%d err=%v", table, count, err)
		}
	}
	var household, unit string
	var agreed float64
	if err := database.QueryRow(`SELECT household_name,unit_id,agreed_power_kw FROM home_profiles WHERE tenant_slug='demo' AND home_key='default'`).Scan(&household, &unit, &agreed); err != nil {
		t.Fatalf("read migrated profile: %v", err)
	}
	if household != "Dachwohnung" || unit != "einheit-12" || agreed != 15 {
		t.Fatalf("profile changed during migration: household=%q unit=%q agreed=%v", household, unit, agreed)
	}
	var foreignKeyErrors int
	if err := database.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&foreignKeyErrors); err != nil || foreignKeyErrors != 0 {
		t.Fatalf("foreign key check: count=%d err=%v", foreignKeyErrors, err)
	}
}

func TestHomeEntitlementEndMigrationGrandfathersExistingProfiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "home-entitlement.db")
	database := openBeforeMigration(t, path, "0031_home_entitlement_end.sql")
	const started = "2025-04-30T21:15:00Z"
	if _, err := database.Exec(`INSERT INTO home_profiles
		(tenant_slug,home_key,free_started_at,created_at,updated_at)
		VALUES('existing','default',?,?,?),('not-started','default',NULL,?,?)`,
		started, started, started, started, started); err != nil {
		t.Fatalf("seed pre-migration profiles: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close pre-migration database: %v", err)
	}

	database, err := Open(path)
	if err != nil {
		t.Fatalf("apply entitlement migration: %v", err)
	}
	defer database.Close()
	var until string
	if err := database.QueryRow(`SELECT free_until_at FROM home_profiles WHERE tenant_slug='existing'`).Scan(&until); err != nil {
		t.Fatalf("read grandfathered end: %v", err)
	}
	if until != "2028-04-30T21:15:00Z" {
		t.Fatalf("grandfathered end = %q, want existing three-year promise", until)
	}
	var missing sql.NullString
	if err := database.QueryRow(`SELECT free_until_at FROM home_profiles WHERE tenant_slug='not-started'`).Scan(&missing); err != nil {
		t.Fatalf("read unstarted entitlement: %v", err)
	}
	if missing.Valid {
		t.Fatalf("migration started a new entitlement: %q", missing.String)
	}
}

func openBeforeMigration(t *testing.T, path, stopBefore string) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open pre-migration db: %v", err)
	}
	if _, err := database.Exec(`CREATE TABLE schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT (datetime('now')))`); err != nil {
		database.Close()
		t.Fatalf("create migration ledger: %v", err)
	}
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		database.Close()
		t.Fatalf("read migrations: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") && !strings.HasPrefix(entry.Name(), ".") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		if name == stopBefore {
			break
		}
		raw, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			database.Close()
			t.Fatalf("read migration %s: %v", name, err)
		}
		tx, err := database.Begin()
		if err == nil {
			_, err = tx.Exec(string(raw))
		}
		if err == nil {
			_, err = tx.Exec(`INSERT INTO schema_migrations(version) VALUES(?)`, name)
		}
		if err == nil {
			err = tx.Commit()
		} else if tx != nil {
			tx.Rollback()
		}
		if err != nil {
			database.Close()
			t.Fatalf("apply migration %s: %v", name, err)
		}
	}
	return database
}
