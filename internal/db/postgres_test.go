package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

const (
	tenantA = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	tenantB = "01BX5ZZKBKACTAV9WEVGEMMVRZ"
	tenantC = "01BX5ZZKBKACTAV9WEVGEMMVS0"
)

var expectedTenantTables = []string{
	"announcement_reads", "announcements", "attachments", "ballots", "contacts", "documents",
	"energy_assets", "energy_entity_mappings", "energy_imports", "energy_intervals",
	"energy_maintenance_plans", "energy_measures", "energy_tariff_assessments", "events", "handovers",
	"home_connector_readings", "home_connectors", "home_portals", "home_profiles", "home_reservations",
	"house_memberships", "integration_imports", "issues", "unit_payment_status", "units",
}

func TestOpenConfigDefaultsToSQLite(t *testing.T) {
	database, err := OpenConfig(t.Context(), Config{DSN: t.TempDir() + "/default.db"})
	if err != nil {
		t.Fatalf("open default backend: %v", err)
	}
	database.Close()
}

func TestOpenConfigRejectsUnknownBackend(t *testing.T) {
	if _, err := OpenConfig(t.Context(), Config{Backend: "postgre", DSN: "unused"}); err == nil {
		t.Fatal("unknown backend must fail closed")
	}
}

func TestBeginTenantTxRejectsInvalidTenantBeforeOpeningTransaction(t *testing.T) {
	if _, err := BeginTenantTx(t.Context(), &sql.DB{}, "demo'; RESET ALL; --", nil); err == nil {
		t.Fatal("non-ULID tenant scope must be rejected")
	}
}

func TestPostgresTargetSchemaAndRLS(t *testing.T) {
	baseDSN := strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN"))
	if baseDSN == "" {
		t.Skip("set HAUSV_TEST_POSTGRES_DSN to a disposable database owned by a NOSUPERUSER NOBYPASSRLS role")
	}
	dsn := isolatedPostgresSchema(t, baseDSN)
	cfg := Config{
		Backend:          BackendPostgres,
		DSN:              dsn,
		ConnectTimeout:   3 * time.Second,
		StatementTimeout: 10 * time.Second,
		MaxOpenConns:     1,
		MaxIdleConns:     1,
		ConnMaxLifetime:  time.Minute,
		ConnMaxIdleTime:  time.Minute,
	}

	database, err := OpenConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("open and migrate empty postgres database: %v", err)
	}
	if got := database.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("max open connections = %d, want 1", got)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close first migration connection: %v", err)
	}

	// Opening the same schema applies the migration runner a second time. It
	// must succeed without duplicating migration records or schema objects.
	database, err = OpenConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("second idempotent migration run: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	var migrationCount, distinctMigrationCount int
	if err := database.QueryRow(`SELECT count(*), count(DISTINCT version) FROM schema_migrations`).
		Scan(&migrationCount, &distinctMigrationCount); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if migrationCount != 2 || distinctMigrationCount != 2 {
		t.Fatalf("migration records = %d/%d, want 2/2", migrationCount, distinctMigrationCount)
	}

	assertApplicationRoleCannotBypassRLS(t, database)
	assertTenantIdentity(t, database)
	tables := tenantTables(t, database)
	if len(tables) == 0 {
		t.Fatal("target schema has no tenant-bound tables")
	}
	seedEveryTenantTable(t, database, tables)
	assertEveryTenantTableFailsClosed(t, database, tables)
	assertPooledConnectionLosesTenantScope(t, database)
	assertOtherTenantCannotSeeRows(t, database, tables)
}

func isolatedPostgresSchema(t *testing.T, dsn string) string {
	t.Helper()
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse test postgres DSN: %v", err)
	}
	admin := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { _ = admin.Close() })
	schema := fmt.Sprintf("hausv_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
		t.Fatalf("create isolated test schema: %v", err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE`) })
	if parsed, err := url.Parse(dsn); err == nil && (parsed.Scheme == "postgres" || parsed.Scheme == "postgresql") {
		query := parsed.Query()
		query.Set("search_path", schema)
		parsed.RawQuery = query.Encode()
		return parsed.String()
	}
	return dsn + " search_path=" + schema
}

func assertApplicationRoleCannotBypassRLS(t *testing.T, database *sql.DB) {
	t.Helper()
	var superuser, bypassRLS bool
	if err := database.QueryRow(`SELECT rolsuper, rolbypassrls FROM pg_roles WHERE rolname=current_user`).
		Scan(&superuser, &bypassRLS); err != nil {
		t.Fatalf("inspect application role: %v", err)
	}
	if superuser || bypassRLS {
		t.Fatalf("application role is privileged: superuser=%v bypassrls=%v", superuser, bypassRLS)
	}
}

func assertTenantIdentity(t *testing.T, database *sql.DB) {
	t.Helper()
	if _, err := database.Exec(`INSERT INTO tenant(tenant_id,slug,name) VALUES($1,'haus-a','Haus A'),($2,'haus-b','Haus B')`, tenantA, tenantB); err != nil {
		t.Fatalf("insert tenants: %v", err)
	}
	if _, err := database.Exec(`UPDATE tenant SET slug='haus-a-neu' WHERE tenant_id=$1`, tenantA); err != nil {
		t.Fatalf("slug must remain mutable: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO tenant(tenant_id,slug) VALUES($1,'haus-a-neu')`, tenantC); err == nil {
		t.Fatal("tenant slug must remain unique")
	}
	if _, err := database.Exec(`UPDATE tenant SET tenant_id=$2 WHERE tenant_id=$1`, tenantA, tenantC); err == nil {
		t.Fatal("tenant_id update must be rejected as immutable")
	}
}

func tenantTables(t *testing.T, database *sql.DB) []string {
	t.Helper()
	rows, err := database.Query(`
		SELECT c.table_name
		FROM information_schema.columns c
		JOIN pg_class pc ON pc.relname=c.table_name
		JOIN pg_namespace pn ON pn.oid=pc.relnamespace AND pn.nspname=c.table_schema
		WHERE c.table_schema=current_schema() AND c.column_name='tenant_id' AND c.table_name <> 'tenant'
		ORDER BY c.table_name`)
	if err != nil {
		t.Fatalf("list tenant tables: %v", err)
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatalf("scan tenant table: %v", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate tenant tables: %v", err)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("close tenant table rows: %v", err)
	}
	if got, want := strings.Join(tables, ","), strings.Join(expectedTenantTables, ","); got != want {
		t.Fatalf("tenant-bound tables = %s, want %s", got, want)
	}
	for _, table := range tables {
		var enabled, forced bool
		var policies int
		if err := database.QueryRow(`
			SELECT c.relrowsecurity, c.relforcerowsecurity,
			       (SELECT count(*) FROM pg_policies p WHERE p.schemaname=current_schema() AND p.tablename=$1)
			FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
			WHERE n.nspname=current_schema() AND c.relname=$1`, table).Scan(&enabled, &forced, &policies); err != nil {
			t.Fatalf("inspect RLS on %s: %v", table, err)
		}
		if !enabled || !forced || policies != 1 {
			t.Fatalf("%s RLS: enabled=%v forced=%v policies=%d", table, enabled, forced, policies)
		}
	}
	return tables
}

func seedEveryTenantTable(t *testing.T, database *sql.DB, tables []string) {
	t.Helper()
	if _, err := database.Exec(`INSERT INTO persons(id,email) VALUES('','rls-fixture@example.test')`); err != nil {
		t.Fatalf("seed person: %v", err)
	}
	tx, err := BeginTenantTx(t.Context(), database, tenantA, nil)
	if err != nil {
		t.Fatalf("begin seed transaction: %v", err)
	}
	for _, table := range []string{"home_profiles", "energy_assets", "home_reservations", "home_connectors"} {
		if _, err := tx.Exec(`INSERT INTO `+table+`(tenant_id) VALUES($1)`, tenantA); err != nil {
			t.Fatalf("seed prerequisite %s: %v", table, err)
		}
	}
	for _, table := range tables {
		if table == "home_profiles" || table == "energy_assets" || table == "home_reservations" || table == "home_connectors" {
			continue
		}
		if !regexp.MustCompile(`^[a-z_]+$`).MatchString(table) {
			t.Fatalf("unsafe catalog table name %q", table)
		}
		if _, err := tx.Exec(`INSERT INTO `+table+`(tenant_id) VALUES($1)`, tenantA); err != nil {
			t.Fatalf("seed %s: %v", table, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit seed transaction: %v", err)
	}
}

func assertEveryTenantTableFailsClosed(t *testing.T, database *sql.DB, tables []string) {
	t.Helper()
	for _, table := range tables {
		var count int
		if err := database.QueryRow(`SELECT count(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("unscoped query %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("unscoped %s returned %d rows, want zero", table, count)
		}
	}
}

func assertPooledConnectionLosesTenantScope(t *testing.T, database *sql.DB) {
	t.Helper()
	tx, err := BeginTenantTx(t.Context(), database, tenantA, nil)
	if err != nil {
		t.Fatalf("begin scoped transaction: %v", err)
	}
	var scopedPID, count int
	if err := tx.QueryRow(`SELECT pg_backend_pid(), (SELECT count(*) FROM contacts)`).Scan(&scopedPID, &count); err != nil {
		t.Fatalf("scoped query: %v", err)
	}
	if count != 1 {
		t.Fatalf("scoped contacts count = %d, want 1", count)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit scoped transaction: %v", err)
	}
	var reusedPID int
	if err := database.QueryRow(`SELECT pg_backend_pid(), (SELECT count(*) FROM contacts)`).Scan(&reusedPID, &count); err != nil {
		t.Fatalf("unscoped reused-connection query: %v", err)
	}
	if reusedPID != scopedPID {
		t.Fatalf("pool did not reuse connection: scoped pid=%d next pid=%d", scopedPID, reusedPID)
	}
	if count != 0 {
		t.Fatalf("reused pooled connection leaked previous tenant scope: %d rows", count)
	}

	tx, err = database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin bypass attempt: %v", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`SET LOCAL row_security = off`); err != nil {
		t.Fatalf("disable row_security setting: %v", err)
	}
	if err := tx.QueryRow(`SELECT count(*) FROM contacts`).Scan(&count); err == nil {
		t.Fatal("non-superuser unexpectedly bypassed forced RLS")
	}
}

func assertOtherTenantCannotSeeRows(t *testing.T, database *sql.DB, tables []string) {
	t.Helper()
	tx, err := BeginTenantTx(t.Context(), database, tenantB, nil)
	if err != nil {
		t.Fatalf("begin other-tenant transaction: %v", err)
	}
	defer tx.Rollback()
	for _, table := range tables {
		var count int
		if err := tx.QueryRow(`SELECT count(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("other tenant query %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("other tenant saw %d rows in %s", count, table)
		}
	}
}
