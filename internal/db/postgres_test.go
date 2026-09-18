package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
)

const (
	tenantA = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	tenantB = "01BX5ZZKBKACTAV9WEVGEMMVRZ"
	tenantC = "01BX5ZZKBKACTAV9WEVGEMMVS0"
)

var expectedTenantTables = []string{
	"announcement_reads", "announcements", "annual_statement_consumption_evidence", "annual_statement_cost_types", "annual_statement_deliveries", "annual_statement_period_cost_types", "annual_statement_period_unit_bases", "annual_statement_periods", "annual_statement_prepayments", "annual_statement_receipts", "annual_statement_runs", "attachments", "ballots", "contacts", "documents",
	"energy_assets", "energy_entity_mappings", "energy_imports", "energy_intervals",
	"energy_maintenance_plans", "energy_measures", "energy_tariff_assessments", "events", "handovers",
	"home_connector_readings", "home_connectors", "home_portals", "home_profiles", "home_reservations",
	"house_memberships", "integration_imports", "issues", "parking", "unit_payment_status", "units",
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
	// The expectation is DERIVED from the embedded migrations, not written
	// down. A literal here made adding a migration look like a regression in
	// this test rather than the thing it was: what this check is actually for
	// is that a second run records each migration ONCE, and a hand-maintained
	// count cannot say that any better than counting the files can.
	wantMigrations := len(embeddedPostgresMigrations(t))
	if wantMigrations == 0 {
		t.Fatal("no postgres migrations are embedded — the probe is broken, not the schema")
	}
	var migrationCount, distinctMigrationCount int
	if err := database.QueryRow(`SELECT count(*), count(DISTINCT version) FROM schema_migrations`).
		Scan(&migrationCount, &distinctMigrationCount); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if migrationCount != wantMigrations || distinctMigrationCount != wantMigrations {
		t.Fatalf("migration records = %d/%d, want %d/%d after two runs — a migration was applied twice",
			migrationCount, distinctMigrationCount, wantMigrations, wantMigrations)
	}

	assertApplicationRoleCannotBypassRLS(t, database)
	assertTenantIdentity(t, database)
	tables := tenantTables(t, database)
	if len(tables) == 0 {
		t.Fatal("target schema has no tenant-bound tables")
	}
	seedEveryTenantTable(t, database, tables)
	assertUnscopedSessionSeesNothingAndWritesNothing(t, database, tables)
	assertOnlyTheDeclaredMaintenanceLaneSeesEverything(t, database, tables)
	assertPooledConnectionLosesTenantScope(t, database)
	assertOtherTenantCannotSeeRows(t, database, tables)
	assertTenantLaneCannotWriteAnotherTenantsRow(t, database)
	assertNoGovernedRowCanExistWithoutAnIdentity(t, database, tables)
	assertPreTenantReservationBelongsToNobodyUntilAdopted(t, database)
}

func TestPostgresAnnualStatementPeriodStructureMigrationBackfillsLegacyCatalogs(t *testing.T) {
	baseDSN := strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN"))
	if baseDSN == "" {
		if os.Getenv("HAUSV_TEST_POSTGRES_REQUIRED") == "true" {
			t.Fatal("HAUSV_TEST_POSTGRES_DSN is required when HAUSV_TEST_POSTGRES_REQUIRED=true")
		}
		t.Skip("set HAUSV_TEST_POSTGRES_DSN to exercise the PostgreSQL period-snapshot migration")
	}
	cfg := Config{
		Backend: BackendPostgres, DSN: isolatedPostgresSchema(t, baseDSN),
		ConnectTimeout: 3 * time.Second, StatementTimeout: 10 * time.Second,
		MaxOpenConns: 1, MaxIdleConns: 1,
	}
	database, err := OpenConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("open initial postgres schema: %v", err)
	}
	// 0012 is currently the last PostgreSQL migration. Rewind only that
	// migration so the fixture has the exact legacy schema and the production
	// runner, not test-only SQL, performs the backfill on reopen.
	if _, err := database.Exec(`DROP TABLE annual_statement_period_unit_bases, annual_statement_period_cost_types`); err != nil {
		t.Fatalf("rewind period structure tables: %v", err)
	}
	if _, err := database.Exec(`DELETE FROM schema_migrations WHERE version='0012_annual_statement_period_structure.sql'`); err != nil {
		t.Fatalf("rewind period structure migration record: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO tenant(tenant_id,slug,name) VALUES
		($1,'legacy-empty','Legacy Empty'),($2,'legacy-custom','Legacy Custom')`, tenantA, tenantB); err != nil {
		t.Fatalf("seed legacy tenants: %v", err)
	}
	tx, err := database.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`SET LOCAL hausv.cross_tenant = 'on'`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO annual_statement_periods(
		tenant_slug,tenant_id,year,starts_on,ends_on,updated_at,updated_by) VALUES
		('legacy-empty',$1,2026,'2026-01-01','2026-12-31','2026-08-30T00:00:00Z','legacy@example.com'),
		('legacy-empty',$1,2025,'2025-01-01','2025-12-31','2025-08-30T00:00:00Z','older@example.com'),
		('legacy-custom',$2,2026,'2026-01-01','2026-12-31','2026-08-29T00:00:00Z','period@example.com')`, tenantA, tenantB); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO annual_statement_cost_types(
		tenant_slug,tenant_id,key,name,allocatable,allocation_key,updated_at,updated_by) VALUES
		('legacy-custom',$1,'grundsteuer','Grundsteuer individuell',false,'','2026-08-29T00:00:00Z','custom@example.com'),
		('legacy-custom',$1,'sonderkosten','Sonderkosten',true,'personen','2026-08-29T01:00:00Z','custom@example.com')`, tenantB); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO units(tenant_slug,tenant_id,id,data) VALUES
		('legacy-empty',$1,'top-1','{"id":"top-1","tenant":"legacy-empty","label":"Top 1","miteigentumsanteil":1000000,"usable_area_m2_hundredths":7500,"usable_area_recorded":true,"persons":2,"persons_recorded":true}')`, tenantA); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	database, err = OpenConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("reapply period structure migration: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	assertPostgresMigratedAnnualStatementCostTypes(t, database, tenantA, 2026, legacyDefaultAnnualStatementCostTypes("2026-08-30T00:00:00Z", "legacy@example.com"))
	assertPostgresMigratedAnnualStatementCostTypes(t, database, tenantA, 2025, legacyDefaultAnnualStatementCostTypes("2025-08-30T00:00:00Z", "older@example.com"))
	assertPostgresMigratedAnnualStatementCostTypes(t, database, tenantB, 2026, customizedAnnualStatementCostTypes)
	tx, err = BeginTenantTx(t.Context(), database, tenantA, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var mutableCatalogRows, mea, area, persons int
	var areaRecorded, personsRecorded bool
	if err := tx.QueryRow(`SELECT count(*) FROM annual_statement_cost_types WHERE tenant_id=$1`, tenantA).Scan(&mutableCatalogRows); err != nil {
		t.Fatal(err)
	}
	if mutableCatalogRows != 0 {
		t.Fatalf("migration persisted %d default rows into the empty mutable catalog", mutableCatalogRows)
	}
	if err := tx.QueryRow(`SELECT miteigentumsanteil_ppm,usable_area_m2_hundredths,usable_area_recorded,persons,persons_recorded
		FROM annual_statement_period_unit_bases WHERE tenant_id=$1 AND period_year=2026 AND unit_id='top-1'`, tenantA).
		Scan(&mea, &area, &areaRecorded, &persons, &personsRecorded); err != nil {
		t.Fatal(err)
	}
	if mea != 1_000_000 || area != 7_500 || !areaRecorded || persons != 2 || !personsRecorded {
		t.Fatalf("backfilled unit basis = mea=%d area=%d/%t persons=%d/%t", mea, area, areaRecorded, persons, personsRecorded)
	}
}

func assertPostgresMigratedAnnualStatementCostTypes(t *testing.T, database *sql.DB, tenantID string, year int, want []migratedAnnualStatementCostType) {
	t.Helper()
	tx, err := BeginTenantTx(t.Context(), database, tenantID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT key,name,allocatable,allocation_key,updated_at,updated_by
		FROM annual_statement_period_cost_types WHERE tenant_id=$1 AND period_year=$2 ORDER BY key`, tenantID, year)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := []migratedAnnualStatementCostType{}
	for rows.Next() {
		var item migratedAnnualStatementCostType
		if err := rows.Scan(&item.Key, &item.Name, &item.Allocatable, &item.AllocationKey, &item.UpdatedAt, &item.UpdatedBy); err != nil {
			t.Fatal(err)
		}
		got = append(got, item)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("period cost types for %s/%d = %+v, want %+v", tenantID, year, got, want)
	}
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

// Delivery rows have required addressing and outcome fields rather than empty
// defaults. Use a valid row so the generic probe exercises RLS, not NOT NULL.
func tenantTableSmokeInsert(table string) string {
	if table == "annual_statement_deliveries" {
		return `INSERT INTO annual_statement_deliveries(tenant_id,tenant_slug,id,run_id,revision,party_id,unit_id,document_id,sha256,recipient,sent_at,status,error,actor,attempt) VALUES($1,'rls-fixture','delivery','run',1,'party@example.test','top-1','document','hash','party@example.test','2026-09-06T18:00:00Z','sent','','manager@example.test',1)`
	}
	return `INSERT INTO ` + table + `(tenant_id) VALUES($1)`
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
	// The period-scoped structure tables deliberately have a composite foreign
	// key to their period. Seed that relationship explicitly; a one-column
	// smoke row would test the FK instead of the RLS behavior this fixture owns.
	if _, err := tx.Exec(`INSERT INTO annual_statement_periods(tenant_id,tenant_slug,year)
		VALUES($1,'rls-fixture',2026)`, tenantA); err != nil {
		t.Fatalf("seed prerequisite annual_statement_periods: %v", err)
	}
	if _, err := tx.Exec(`INSERT INTO annual_statement_period_cost_types(tenant_id,tenant_slug,period_year,key)
		VALUES($1,'rls-fixture',2026,'rls-fixture')`, tenantA); err != nil {
		t.Fatalf("seed annual_statement_period_cost_types: %v", err)
	}
	if _, err := tx.Exec(`INSERT INTO annual_statement_period_unit_bases(tenant_id,tenant_slug,period_year,unit_id)
		VALUES($1,'rls-fixture',2026,'rls-fixture')`, tenantA); err != nil {
		t.Fatalf("seed annual_statement_period_unit_bases: %v", err)
	}
	for _, table := range tables {
		if table == "home_profiles" || table == "energy_assets" || table == "home_reservations" || table == "home_connectors" ||
			table == "annual_statement_periods" || table == "annual_statement_period_cost_types" || table == "annual_statement_period_unit_bases" {
			continue
		}
		if !regexp.MustCompile(`^[a-z_]+$`).MatchString(table) {
			t.Fatalf("unsafe catalog table name %q", table)
		}
		if _, err := tx.Exec(tenantTableSmokeInsert(table), tenantA); err != nil {
			t.Fatalf("seed %s: %v", table, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit seed transaction: %v", err)
	}
}

// assertUnscopedSessionSeesNothingAndWritesNothing pins what migration 0006
// changed, and it is the assertion migration 0003 said it could not make yet.
//
// Under 0003 an unscoped session — one carrying neither hausv.tenant_id nor
// hausv.cross_tenant — was the maintenance view and saw everything, because the
// application issued every query on an unscoped pooled connection and a policy
// that hid rows from it would have hidden the whole database. Every SQL surface
// on the serve path now runs on a lane, so the escape moved from "declared
// nothing" to "declared cross-tenant", and the process pool — which declares
// nothing — reads zero rows from every governed table and cannot write one.
//
// The tenant registry is the deliberate exception (0002/0003/0006 all exclude
// it): the boot path mints identities on the pool before any lane exists.
func assertUnscopedSessionSeesNothingAndWritesNothing(t *testing.T, database *sql.DB, tables []string) {
	t.Helper()
	if got := scopeOfSession(t, database); got != "" {
		t.Fatalf("the process pool carries a scope %q; this assertion is about an undeclared session", got)
	}
	for _, table := range tables {
		var count int
		if err := database.QueryRow(`SELECT count(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("unscoped query %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("unscoped %s returned %d rows, want 0: an undeclared session must be fail-closed", table, count)
		}
		// The write side, per statement kind. INSERT is refused by WITH CHECK;
		// UPDATE and DELETE simply find nothing to touch, because USING hides
		// every row from them.
		_, err := database.Exec(tenantTableSmokeInsert(table), tenantA)
		if code := sqlState(err); code != "42501" {
			t.Fatalf("unscoped INSERT into %s: err=%v (SQLSTATE %q), want the policy's 42501", table, err, code)
		}
		for _, statement := range []string{
			`UPDATE ` + table + ` SET tenant_id=tenant_id`,
			`DELETE FROM ` + table,
		} {
			result, err := database.Exec(statement)
			if err != nil {
				t.Fatalf("unscoped %q: %v", statement, err)
			}
			if affected, _ := result.RowsAffected(); affected != 0 {
				t.Fatalf("unscoped %q touched %d rows, want 0", statement, affected)
			}
		}
	}
	var tenants int
	if err := database.QueryRow(`SELECT count(*) FROM tenant`).Scan(&tenants); err != nil {
		t.Fatalf("unscoped query tenant: %v", err)
	}
	if tenants != 2 {
		t.Fatalf("the tenant registry must stay reachable from the pool, saw %d rows, want 2", tenants)
	}
}

// assertOnlyTheDeclaredMaintenanceLaneSeesEverything is the escape hatch, and
// its exact shape: hausv.cross_tenant='on', the value db.Scoped.Unscoped sends
// in the startup packet — not merely "set", not any other spelling.
func assertOnlyTheDeclaredMaintenanceLaneSeesEverything(t *testing.T, database *sql.DB, tables []string) {
	t.Helper()
	for _, tc := range []struct {
		declared string
		want     int
	}{
		{declared: "on", want: 1},
		{declared: "off", want: 0},
		{declared: "yes", want: 0},
		{declared: "ON", want: 0},
		{declared: "", want: 0},
	} {
		tx, err := database.BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatalf("begin declared transaction: %v", err)
		}
		// The value has been chosen from a fixed list above; SET LOCAL does not
		// take parameters.
		if _, err := tx.Exec(`SET LOCAL hausv.cross_tenant = '` + tc.declared + `'`); err != nil {
			tx.Rollback()
			t.Fatalf("declare cross_tenant=%q: %v", tc.declared, err)
		}
		for _, table := range tables {
			var count int
			if err := tx.QueryRow(`SELECT count(*) FROM ` + table).Scan(&count); err != nil {
				tx.Rollback()
				t.Fatalf("cross_tenant=%q query %s: %v", tc.declared, table, err)
			}
			if count != tc.want {
				tx.Rollback()
				t.Fatalf("cross_tenant=%q sees %d rows in %s, want %d", tc.declared, count, table, tc.want)
			}
		}
		tx.Rollback()
	}
}

// assertTenantLaneCannotWriteAnotherTenantsRow is the write half of isolation:
// WITH CHECK refuses an INSERT that names another tenant, and USING hides the
// other tenant's rows from UPDATE and DELETE.
func assertTenantLaneCannotWriteAnotherTenantsRow(t *testing.T, database *sql.DB) {
	t.Helper()
	tx, err := BeginTenantTx(t.Context(), database, tenantB, nil)
	if err != nil {
		t.Fatalf("begin tenant B transaction: %v", err)
	}
	_, err = tx.Exec(`INSERT INTO contacts(tenant_id, id) VALUES($1, 'smuggled')`, tenantA)
	// A failed statement poisons the transaction on PostgreSQL, and the pool
	// under this test has ONE connection: the rest needs a fresh transaction and
	// this one must be released first, or the next Begin blocks forever.
	tx.Rollback()
	if code := sqlState(err); code != "42501" {
		t.Fatalf("tenant B inserted a row for tenant A: err=%v (SQLSTATE %q), want 42501", err, code)
	}
	tx, err = BeginTenantTx(t.Context(), database, tenantB, nil)
	if err != nil {
		t.Fatalf("begin second tenant B transaction: %v", err)
	}
	defer tx.Rollback()
	for _, statement := range []string{
		`UPDATE contacts SET active=false WHERE tenant_id='` + tenantA + `'`,
		`DELETE FROM contacts WHERE tenant_id='` + tenantA + `'`,
	} {
		result, err := tx.Exec(statement)
		if err != nil {
			t.Fatalf("tenant B %q: %v", statement, err)
		}
		if affected, _ := result.RowsAffected(); affected != 0 {
			t.Fatalf("tenant B %q touched %d of tenant A's rows", statement, affected)
		}
	}
}

// assertNoGovernedRowCanExistWithoutAnIdentity is the loud half of 0006: NOT
// NULL on every governed table but the pre-tenant one, read from the catalog
// AND exercised, because the migration is catalog-driven and a table it missed
// would otherwise be found by an orphan in production rather than here.
//
// home_reservations is the stated exemption. A reservation precedes the house,
// so its identity is minted at activation; it is asserted nullable HERE so that
// widening the exemption is a visible change to this list, not a drift.
func assertNoGovernedRowCanExistWithoutAnIdentity(t *testing.T, database *sql.DB, tables []string) {
	t.Helper()
	for _, table := range tables {
		var notNull bool
		if err := database.QueryRow(`
			SELECT a.attnotnull FROM pg_attribute a
			JOIN pg_class c ON c.oid=a.attrelid
			JOIN pg_namespace n ON n.oid=c.relnamespace
			WHERE n.nspname=current_schema() AND c.relname=$1 AND a.attname='tenant_id'`, table).Scan(&notNull); err != nil {
			t.Fatalf("inspect %s.tenant_id: %v", table, err)
		}
		wantNotNull := table != "home_reservations"
		if notNull != wantNotNull {
			t.Fatalf("%s.tenant_id NOT NULL = %v, want %v", table, notNull, wantNotNull)
		}
	}
	// Exercised, on the one lane the policy lets through, so the refusal is the
	// column's and not the policy's.
	tx, err := database.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatalf("begin maintenance transaction: %v", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`SET LOCAL hausv.cross_tenant = 'on'`); err != nil {
		t.Fatalf("declare maintenance lane: %v", err)
	}
	_, err = tx.Exec(`INSERT INTO contacts(tenant_id, id) VALUES(NULL, 'orphan')`)
	if code := sqlState(err); code != "23502" {
		t.Fatalf("an orphan row was accepted: err=%v (SQLSTATE %q), want 23502", err, code)
	}
}

// assertPreTenantReservationBelongsToNobodyUntilAdopted keeps, for the one
// table that still admits a NULL tenant_id, the two properties 0003 established
// on every table: an identity-less row is visible to NO tenant lane and only to
// the maintenance lane, and it can be given an identity once — activation is
// exactly that transition — but never re-pointed or erased afterwards.
func assertPreTenantReservationBelongsToNobodyUntilAdopted(t *testing.T, database *sql.DB) {
	t.Helper()
	maintenance := func(statement string, args ...any) (sql.Result, error) {
		tx, err := database.BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatalf("begin maintenance transaction: %v", err)
		}
		defer tx.Rollback()
		if _, err := tx.Exec(`SET LOCAL hausv.cross_tenant = 'on'`); err != nil {
			t.Fatalf("declare maintenance lane: %v", err)
		}
		result, err := tx.Exec(statement, args...)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("commit maintenance transaction: %v", err)
		}
		return result, nil
	}
	if _, err := maintenance(`INSERT INTO home_reservations(slug, tenant_id) VALUES('pre-tenant', NULL)`); err != nil {
		t.Fatalf("seed pre-tenant reservation: %v", err)
	}
	countAs := func(tenant string) int {
		tx, err := BeginTenantTx(t.Context(), database, tenant, nil)
		if err != nil {
			t.Fatalf("begin scoped transaction: %v", err)
		}
		defer tx.Rollback()
		var count int
		if err := tx.QueryRow(`SELECT count(*) FROM home_reservations WHERE slug='pre-tenant'`).Scan(&count); err != nil {
			t.Fatalf("scoped reservation query: %v", err)
		}
		return count
	}
	for _, tenant := range []string{tenantA, tenantB} {
		if count := countAs(tenant); count != 0 {
			t.Fatalf("tenant %s saw %d pre-tenant reservations it does not own", tenant, count)
		}
	}
	if _, err := maintenance(`UPDATE home_reservations SET tenant_id=$1 WHERE slug='pre-tenant'`, tenantB); err != nil {
		t.Fatalf("a pre-tenant reservation must be adoptable at activation: %v", err)
	}
	if got := countAs(tenantB); got != 1 {
		t.Fatalf("after adoption tenant B sees %d reservations, want 1", got)
	}
	if got := countAs(tenantA); got != 0 {
		t.Fatalf("after adoption tenant A sees %d of tenant B's reservations", got)
	}
	if _, err := maintenance(`UPDATE home_reservations SET tenant_id=$1 WHERE slug='pre-tenant'`, tenantA); err == nil {
		t.Fatal("re-pointing an owned reservation at another tenant must stay rejected")
	}
	if _, err := maintenance(`UPDATE home_reservations SET tenant_id=NULL WHERE slug='pre-tenant'`); err == nil {
		t.Fatal("erasing an identity must stay rejected")
	}
	if _, err := maintenance(`DELETE FROM home_reservations WHERE slug='pre-tenant'`); err != nil {
		t.Fatalf("clean up pre-tenant reservation: %v", err)
	}
}

// scopeOfSession reads the tenant scope a handle's session carries, empty when
// none.
func scopeOfSession(t *testing.T, database *sql.DB) string {
	t.Helper()
	var scope string
	if err := database.QueryRow(`SELECT coalesce(current_setting('hausv.tenant_id', true), '')`).Scan(&scope); err != nil {
		t.Fatalf("read session scope: %v", err)
	}
	return scope
}

// sqlState extracts the SQLSTATE of a PostgreSQL error, empty for nil or a
// non-PostgreSQL error, so an assertion can name the exact refusal it wants
// instead of accepting any error.
func sqlState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
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
	// The scope itself is the thing to probe, not a row count: since 0006 an
	// unscoped session sees nothing, so a count of zero here would be
	// indistinguishable from a scope that survived onto a session that then
	// saw a tenant that owns no contacts. The setting says it directly.
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

// embeddedPostgresMigrations lists the migration files the binary carries, so
// the idempotence check above counts what actually ran.
func embeddedPostgresMigrations(t *testing.T) []string {
	t.Helper()
	entries, err := postgresMigrationsFS.ReadDir("postgres/migrations")
	if err != nil {
		t.Fatalf("read embedded postgres migrations: %v", err)
	}
	names := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".sql") {
			names = append(names, name)
		}
	}
	return names
}
