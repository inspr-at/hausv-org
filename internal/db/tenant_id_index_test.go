package db

import (
	"context"
	"database/sql"
	"os"
	"sort"
	"strings"
	"testing"
)

// TestEveryTenantIDColumnIsIndexedOnBothEngines closes the gap the query switch
// opened on PostgreSQL.
//
// SQLite migration 0033 gave all 25 tenant-scoped tables a tenant_id column AND
// an index on it. The PostgreSQL migrations gave them the column and no index at
// all — nine index references to tenant_slug, zero to tenant_id. Every store
// read now filters on tenant_id, so on PostgreSQL every one of them fell back to
// a sequential scan the moment the switch landed. It is not a live regression
// while production runs SQLite, and it must not survive to the cutover.
//
// It asks both engines the same question so neither can drift again.
//
// Where it is blind: it requires an index whose LEADING column is tenant_id and
// says nothing about the rest of the key, so a composite index that starts with
// tenant_id satisfies it even if the trailing columns are wrong for the actual
// queries. It also proves an index EXISTS, not that the planner chooses it.
func TestEveryTenantIDColumnIsIndexedOnBothEngines(t *testing.T) {
	ctx := t.Context()

	sqliteDB, err := OpenConfig(ctx, Config{DSN: t.TempDir() + "/indexes.db"})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer sqliteDB.Close()
	assertTenantIDIndexes(ctx, t, "sqlite", sqliteDB, sqliteTenantIDTables, sqliteTenantIDIndexedTables)

	baseDSN := strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN"))
	if baseDSN == "" {
		if os.Getenv("HAUSV_TEST_POSTGRES_REQUIRED") == "true" {
			t.Fatal("HAUSV_TEST_POSTGRES_DSN is required when HAUSV_TEST_POSTGRES_REQUIRED=true")
		}
		t.Skip("set HAUSV_TEST_POSTGRES_DSN to a disposable database to check the PostgreSQL half")
	}
	pgDB, err := OpenConfig(ctx, Config{Backend: BackendPostgres, DSN: isolatedPostgresSchema(t, baseDSN)})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pgDB.Close()
	assertTenantIDIndexes(ctx, t, "postgres", pgDB, postgresTenantIDTables, postgresTenantIDIndexedTables)
}

func assertTenantIDIndexes(ctx context.Context, t *testing.T, engine string, database *sql.DB, tablesQuery, indexedQuery string) {
	t.Helper()
	tables, err := readNames(ctx, database, tablesQuery)
	if err != nil {
		t.Fatalf("%s: read tables carrying tenant_id: %v", engine, err)
	}
	if len(tables) == 0 {
		// Without this the test would report success for a schema it failed to
		// read, which is the failure mode the migration itself had.
		t.Fatalf("%s: no table carries a tenant_id column — the probe is broken, not the schema", engine)
	}
	indexed, err := readNames(ctx, database, indexedQuery)
	if err != nil {
		t.Fatalf("%s: read tenant_id indexes: %v", engine, err)
	}
	has := map[string]bool{}
	for _, name := range indexed {
		has[name] = true
	}
	missing := []string{}
	for _, table := range tables {
		if !has[table] {
			missing = append(missing, table)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("%s: %d of %d tenant-scoped tables have no index leading with tenant_id: %s\n"+
			"every store read filters on that column", engine, len(missing), len(tables), strings.Join(missing, " "))
	}
}

func readNames(ctx context.Context, database *sql.DB, query string) ([]string, error) {
	rows, err := database.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

const sqliteTenantIDTables = `SELECT m.name FROM sqlite_master m JOIN pragma_table_info(m.name) p
	WHERE m.type='table' AND p.name='tenant_id' AND m.name <> 'tenant'`

// seqno=0 is the leading column of the index, which is the only position that
// makes an equality filter on tenant_id cheap.
const sqliteTenantIDIndexedTables = `SELECT DISTINCT m.name FROM sqlite_master m
	JOIN pragma_index_list(m.name) l JOIN pragma_index_info(l.name) i
	WHERE m.type='table' AND m.name <> 'tenant' AND i.seqno=0 AND i.name='tenant_id'`

const postgresTenantIDTables = `SELECT c.table_name
	FROM information_schema.columns c
	JOIN pg_catalog.pg_tables t ON t.schemaname=c.table_schema AND t.tablename=c.table_name
	WHERE c.table_schema=current_schema() AND c.column_name='tenant_id' AND c.table_name <> 'tenant'`

const postgresTenantIDIndexedTables = `SELECT DISTINCT c.relname
	FROM pg_index x
	JOIN pg_class c ON c.oid = x.indrelid
	JOIN pg_namespace n ON n.oid = c.relnamespace
	JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = x.indkey[0]
	WHERE n.nspname = current_schema() AND a.attname = 'tenant_id' AND c.relname <> 'tenant'`
