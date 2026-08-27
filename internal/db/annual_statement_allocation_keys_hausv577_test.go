package db

import (
	"path/filepath"
	"testing"
)

// Rows written before 0036 have no allocation_key column. The migration must
// add it, backfill every allocatable row to Nutzwert and leave non-allocatable
// rows without a key, so that "one key per allocatable cost type" holds for
// data that predates the feature.
func TestAnnualStatementAllocationKeyMigrationBackfillsAllocatableRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "allocation-keys.db")
	database := openBeforeMigration(t, path, "0036_annual_statement_allocation_keys.sql")
	if _, err := database.Exec(`INSERT INTO tenant(tenant_id, slug, created_at, updated_at) VALUES('01ARZ3NDEKTSV4RRFFQ69G5FAV','demo','2026-08-27T00:00:00Z','2026-08-27T00:00:00Z')`); err != nil {
		t.Fatalf("insert tenant: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO annual_statement_cost_types(tenant_slug, tenant_id, key, name, allocatable, updated_at, updated_by) VALUES
		('demo','01ARZ3NDEKTSV4RRFFQ69G5FAV','grundsteuer','Grundsteuer',1,'2026-08-27T00:00:00Z','a@example.com'),
		('demo','01ARZ3NDEKTSV4RRFFQ69G5FAV','ruecklage','Rücklage',0,'2026-08-27T00:00:00Z','a@example.com')`); err != nil {
		t.Fatalf("insert pre-migration rows: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	database, err := Open(path)
	if err != nil {
		t.Fatalf("open with migration: %v", err)
	}
	defer database.Close()
	rows, err := database.Query(`SELECT key, allocation_key FROM annual_statement_cost_types ORDER BY key`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	got := map[string]string{}
	for rows.Next() {
		var key, allocationKey string
		if err := rows.Scan(&key, &allocationKey); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got[key] = allocationKey
	}
	if got["grundsteuer"] != "nutzwert" || got["ruecklage"] != "" || len(got) != 2 {
		t.Fatalf("backfill = %v, want grundsteuer→nutzwert and ruecklage→\"\"", got)
	}
}
