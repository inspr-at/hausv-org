package db

import (
	"context"
	"database/sql"
	"os"
	"sort"
	"strings"
	"testing"
)

// The two engines are migrated by two SEPARATE sets of SQL: 32 files under
// migrations/ for SQLite, 2 under postgres/migrations/ for PostgreSQL. The
// PostgreSQL pair was written once, in Phase 0, as a picture of the intended end
// state — and then thirty SQLite migrations landed on top of it without it moving.
//
// Nothing detected that. Both suites were green: the SQLite suite because it never
// opens PostgreSQL, and the PostgreSQL schema test because it asserts what the
// target file declares rather than what the application needs. So the drift was
// invisible until the store suite ran against PostgreSQL and 82 tests failed.
//
// This test is the missing contradiction. It migrates both engines from scratch and
// requires their table and column sets to agree. It does not care which side is
// wrong — only that they differ, which is always a defect in one of them.
//
// Where it is blind, deliberately and stated here rather than discovered later:
//   - Column TYPES are reported but not enforced. The engines legitimately spell
//     types differently (TEXT vs text, INTEGER vs bigint) and a faithful comparison
//     needs an equivalence table that would itself become something to maintain.
//     Presence is the unambiguous half, and it is the half that was broken.
//   - Indexes, triggers, defaults and constraints are not compared. RLS and the
//     tenant_id trigger have their own test in TestPostgresTargetSchemaAndRLS.
//   - It proves the two schemas agree with each other, NOT that either matches what
//     the Go code queries. A column both sides are missing passes here.
func TestSchemaAgreementBetweenEngines(t *testing.T) {
	baseDSN := strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN"))
	if baseDSN == "" {
		if os.Getenv("HAUSV_TEST_POSTGRES_REQUIRED") == "true" {
			t.Fatal("HAUSV_TEST_POSTGRES_DSN is required when HAUSV_TEST_POSTGRES_REQUIRED=true")
		}
		t.Skip("set HAUSV_TEST_POSTGRES_DSN to a disposable database to compare the two schemas")
	}

	ctx := t.Context()

	sqliteDB, err := OpenConfig(ctx, Config{DSN: t.TempDir() + "/agreement.db"})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer sqliteDB.Close()

	pgDB, err := OpenConfig(ctx, Config{Backend: BackendPostgres, DSN: isolatedPostgresSchema(t, baseDSN)})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pgDB.Close()

	lite, err := sqliteSchema(ctx, sqliteDB)
	if err != nil {
		t.Fatalf("read sqlite schema: %v", err)
	}
	pg, err := postgresSchema(ctx, pgDB)
	if err != nil {
		t.Fatalf("read postgres schema: %v", err)
	}

	if len(lite) == 0 || len(pg) == 0 {
		t.Fatalf("a schema came back empty (sqlite %d tables, postgres %d) — the probe is broken, not the schema", len(lite), len(pg))
	}

	missingTables, extraTables := diffKeys(lite, pg)
	for _, name := range missingTables {
		t.Errorf("table %q exists in SQLite but not in PostgreSQL", name)
	}
	for _, name := range extraTables {
		t.Errorf("table %q exists in PostgreSQL but not in SQLite", name)
	}

	// Columns, for the tables both engines have. Reported per table so the output is
	// a work list rather than a wall.
	for _, table := range sortedKeys(lite) {
		pgCols, ok := pg[table]
		if !ok {
			continue // already reported above
		}
		missing, extra := diffKeys(lite[table], pgCols)
		if len(missing) > 0 {
			t.Errorf("%s: columns missing from PostgreSQL: %s", table, strings.Join(missing, ", "))
		}
		if len(extra) > 0 {
			t.Errorf("%s: columns only in PostgreSQL: %s", table, strings.Join(extra, ", "))
		}
	}
}

// schema maps table name -> column name -> declared type.
type schema map[string]map[string]string

func sqliteSchema(ctx context.Context, database *sql.DB) (schema, error) {
	rows, err := database.QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	out := schema{}
	for _, table := range tables {
		if ignoredTable(table) {
			continue
		}
		// PRAGMA does not accept a placeholder, and table names here come from
		// sqlite_master rather than from input.
		colRows, err := database.QueryContext(ctx, `PRAGMA table_info("`+table+`")`)
		if err != nil {
			return nil, err
		}
		cols := map[string]string{}
		for colRows.Next() {
			var (
				cid        int
				name, typ  string
				notNull    int
				dflt       sql.NullString
				primaryKey int
			)
			if err := colRows.Scan(&cid, &name, &typ, &notNull, &dflt, &primaryKey); err != nil {
				colRows.Close()
				return nil, err
			}
			cols[name] = strings.ToLower(typ)
		}
		if err := colRows.Err(); err != nil {
			colRows.Close()
			return nil, err
		}
		colRows.Close()
		out[table] = cols
	}
	return out, nil
}

func postgresSchema(ctx context.Context, database *sql.DB) (schema, error) {
	// current_schema() keeps this inside the per-test schema minted by
	// isolatedPostgresSchema rather than reading the whole database.
	rows, err := database.QueryContext(ctx,
		`SELECT table_name, column_name, data_type
		   FROM information_schema.columns
		  WHERE table_schema = current_schema()
		  ORDER BY table_name, column_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := schema{}
	for rows.Next() {
		var table, column, typ string
		if err := rows.Scan(&table, &column, &typ); err != nil {
			return nil, err
		}
		if ignoredTable(table) {
			continue
		}
		if out[table] == nil {
			out[table] = map[string]string{}
		}
		out[table][column] = strings.ToLower(typ)
	}
	return out, rows.Err()
}

// ignoredTable drops bookkeeping that legitimately differs between the engines.
func ignoredTable(name string) bool {
	return name == "schema_migrations"
}

// diffKeys returns keys present in want but not got, and keys present in got but
// not want. Generic over the value type so it serves both tables and columns.
func diffKeys[V any](want, got map[string]V) (missing, extra []string) {
	for key := range want {
		if _, ok := got[key]; !ok {
			missing = append(missing, key)
		}
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			extra = append(extra, key)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return missing, extra
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
