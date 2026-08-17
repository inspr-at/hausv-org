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
		if os.Getenv("HAUSV_TEST_POSTGRES_REQUIRED") == "true" {
			t.Fatal("HAUSV_TEST_POSTGRES_DSN is required when HAUSV_TEST_POSTGRES_REQUIRED=true")
		}
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
	if migrationCount != 4 || distinctMigrationCount != 4 {
		t.Fatalf("migration records = %d/%d, want 4/4", migrationCount, distinctMigrationCount)
	}

	assertApplicationRoleCannotBypassRLS(t, database)
	assertTenantIdentity(t, database)
	tables := tenantTables(t, database)
	if len(tables) == 0 {
		t.Fatal("target schema has no tenant-bound tables")
	}
	seedEveryTenantTable(t, database, tables)
	assertUnscopedSessionIsTheMaintenanceView(t, database, tables)
	assertPooledConnectionLosesTenantScope(t, database)
	assertOtherTenantCannotSeeRows(t, database, tables)
	assertRowsWithoutAnIdentityBelongToNobody(t, database)
	assertOrphanRowCanBeAdoptedOnceAndNeverRepointed(t, database)
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

// assertUnscopedSessionIsTheMaintenanceView pins what migration 0003 changed.
//
// Under 0002 an unscoped session saw only rows with a NULL tenant_id, which read
// as "fail closed" but was an accident: it was true only because nothing wrote
// tenant_id. Now that every writer does, the same policy would have hidden the
// entire database from the application, which issues every query on an unscoped
// pooled connection.
//
// So the escape moved from the row to the session, and this is the assertion
// that says so out loud: an unscoped session is the maintenance view and sees
// everything. It is NOT fail-closed, and pretending otherwise in a test name
// would be the more dangerous mistake. The isolation that matters — one scoped
// session cannot see another tenant, and un-owned rows belong to nobody — is
// asserted separately below.
func assertUnscopedSessionIsTheMaintenanceView(t *testing.T, database *sql.DB, tables []string) {
	t.Helper()
	for _, table := range tables {
		var count int
		if err := database.QueryRow(`SELECT count(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("unscoped query %s: %v", table, err)
		}
		if count != 1 {
			t.Fatalf("unscoped %s returned %d rows, want the single seeded row", table, count)
		}
	}
}

// assertRowsWithoutAnIdentityBelongToNobody is the leak 0002 had. A row with a
// NULL tenant_id used to satisfy every scoped tenant's policy, so tenant B was
// served tenant A's un-backfilled rows.
func assertRowsWithoutAnIdentityBelongToNobody(t *testing.T, database *sql.DB) {
	t.Helper()
	if _, err := database.Exec(`INSERT INTO contacts(tenant_id, id) VALUES(NULL, 'orphan')`); err != nil {
		t.Fatalf("seed orphan row: %v", err)
	}
	for _, tenant := range []string{tenantA, tenantB} {
		tx, err := BeginTenantTx(t.Context(), database, tenant, nil)
		if err != nil {
			t.Fatalf("begin scoped transaction: %v", err)
		}
		var count int
		if err := tx.QueryRow(`SELECT count(*) FROM contacts WHERE id='orphan'`).Scan(&count); err != nil {
			tx.Rollback()
			t.Fatalf("scoped orphan query: %v", err)
		}
		tx.Rollback()
		if count != 0 {
			t.Fatalf("tenant %s saw %d rows it does not own", tenant, count)
		}
	}
}

// assertOrphanRowCanBeAdoptedOnceAndNeverRepointed covers the other half of
// migration 0003: the immutability trigger used to reject NULL -> value, which
// made a row the backfill missed permanently unrepairable.
func assertOrphanRowCanBeAdoptedOnceAndNeverRepointed(t *testing.T, database *sql.DB) {
	t.Helper()
	if _, err := database.Exec(`UPDATE contacts SET tenant_id=$1 WHERE id='orphan'`, tenantB); err != nil {
		t.Fatalf("an un-owned row must be adoptable: %v", err)
	}
	if _, err := database.Exec(`UPDATE contacts SET tenant_id=$1 WHERE id='orphan'`, tenantA); err == nil {
		t.Fatal("re-pointing an owned row at another tenant must stay rejected")
	}
	if _, err := database.Exec(`UPDATE contacts SET tenant_id=NULL WHERE id='orphan'`); err == nil {
		t.Fatal("erasing an identity must stay rejected")
	}
	if _, err := database.Exec(`DELETE FROM contacts WHERE id='orphan'`); err != nil {
		t.Fatalf("clean up orphan row: %v", err)
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
	// The scope itself is now the thing to probe. Counting rows no longer
	// distinguishes "scope was dropped" from "scope survived", because an
	// unscoped session legitimately sees the same row; the setting does.
	var reusedPID int
	var leakedScope string
	if err := database.QueryRow(
		`SELECT pg_backend_pid(), coalesce(current_setting('hausv.tenant_id', true), '')`,
	).Scan(&reusedPID, &leakedScope); err != nil {
		t.Fatalf("unscoped reused-connection query: %v", err)
	}
	if reusedPID != scopedPID {
		t.Fatalf("pool did not reuse connection: scoped pid=%d next pid=%d", scopedPID, reusedPID)
	}
	if leakedScope != "" {
		t.Fatalf("reused pooled connection kept the previous tenant scope: %q", leakedScope)
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
