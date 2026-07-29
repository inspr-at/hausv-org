package db

import (
	"path/filepath"
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

func TestConsumptionMappingMigrationOnlyCorrectsLegacyBatteryHeuristic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "energy-mapping.db")
	database, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	now := "2026-07-29T10:00:00Z"
	if _, err := database.Exec(`INSERT INTO home_profiles(tenant_slug,created_at,updated_at) VALUES('jhw22',?,?)`, now, now); err != nil {
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
			VALUES(?,'jhw22',?,'battery-power',?,'W','power',1,?,?, '')`,
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
