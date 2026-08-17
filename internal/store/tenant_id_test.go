package store

import (
	"context"
	"database/sql"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

// tenantSlugPredicate matches tenant_slug used as something a row is SELECTED,
// UPDATED or DELETED by — the position the switch moved off. It deliberately
// does not match the positions tenant_slug legitimately keeps:
//
//	INSERT INTO x(tenant_id, tenant_slug, ...)   column list, still written
//	ON CONFLICT(tenant_slug, id)                 the unique index that exists
//	SELECT tenant_slug, unit_id, ...             projection
//	ORDER BY tenant_slug                         user-visible alphabetical order
//
// all of which are followed by a comma, a closing paren or end-of-string rather
// than a comparison operator.
var tenantSlugPredicate = regexp.MustCompile(`(?i)tenant_slug\s*(=|<>|!=|<|>|\bIN\b|\bLIKE\b)`)

// TestStoreQueriesFilterOnTenantID is the check requirement 6 asks for: it fails
// if a query in this package starts addressing rows by the tenant's renameable
// label again.
//
// It is a Go test rather than a grep script so it parses the package the way the
// compiler does — every string literal in every non-test file, including the
// ones built by concatenating a shared column constant — instead of matching
// lines of text.
//
// Where it is blind, stated here rather than discovered later:
//
//   - It only reads internal/store. internal/energy still filters on
//     tenant_slug by design (its Storage API has no tenant identity to filter
//     by yet) and internal/server's import ledger was switched by hand. Neither
//     is covered.
//   - It sees only string LITERALS. A predicate assembled from a variable at
//     runtime — home_connector.go passes its WHERE clause in as a Go string —
//     is invisible to it.
//   - The four onboarding tables (home_reservations, home_portals,
//     home_connectors, home_connector_readings) scope rows by a column named
//     `slug`, not `tenant_slug`, so nothing here can see those predicates at
//     all. They are listed in tenantIDTables and their tenant_id is written and
//     verified, but their queries are still keyed on the label.
//   - It proves a query does not name tenant_slug in a predicate. It cannot
//     prove the tenant_id being bound is the RIGHT tenant's — that is what the
//     cross-tenant isolation tests are for.
func TestStoreQueriesFilterOnTenantID(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}
	checked := 0
	offenders := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			text := literalText(lit.Value)
			if !strings.Contains(text, "tenant_slug") {
				return true
			}
			checked++
			if match := tenantSlugPredicate.FindString(text); match != "" {
				offenders = append(offenders, fset.Position(lit.Pos()).String()+": "+collapse(text))
			}
			return true
		})
	}
	if checked == 0 {
		// Without this the test would pass on a package it failed to read at
		// all, which is the failure mode a grep script has.
		t.Fatal("no SQL naming tenant_slug was found — the probe is broken, not the package")
	}
	for _, offender := range offenders {
		t.Errorf("store query addresses rows by tenant_slug; use the bound TenantRef's ID\n  %s", offender)
	}
}

func literalText(raw string) string {
	if strings.HasPrefix(raw, "`") {
		return strings.Trim(raw, "`")
	}
	unquoted, err := strconv.Unquote(raw)
	if err != nil {
		return raw
	}
	return unquoted
}

func collapse(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// TestMembershipColumnsArityMatchesItsInsert guards the one constant that is
// concatenated into five statements across two files. Adding tenant_id to it
// changed the arity of every one; a future edit that adds a column without
// adding a placeholder would compile and fail only at runtime.
func TestMembershipColumnsArityMatchesItsInsert(t *testing.T) {
	columns := len(strings.Split(membershipColumns, ","))
	source, err := os.ReadFile(filepath.Join(".", "identity_sql.go"))
	if err != nil {
		t.Fatalf("read identity_sql.go: %v", err)
	}
	values := regexp.MustCompile(`INSERT INTO house_memberships\(` + "`" + `\+membershipColumns\+` + "`" + `\) VALUES\(([^)]*)\)`).
		FindSubmatch(source)
	if values == nil {
		t.Fatal("the house_memberships INSERT no longer matches the probe — update this test with it")
	}
	placeholders := len(strings.Split(string(values[1]), ","))
	if placeholders != columns {
		t.Fatalf("membershipColumns has %d columns but the INSERT binds %d placeholders", columns, placeholders)
	}
	if !strings.Contains(membershipColumns, "tenant_id") {
		t.Fatal("membershipColumns must still carry tenant_id")
	}
	if !strings.Contains(membershipColumns, "tenant_slug") {
		t.Fatal("membershipColumns must still write tenant_slug: the rollback window depends on it")
	}
}

// TestTenantIDTablesMatchTheMigratedSchema keeps the backfill list honest. A
// migration that gives a new table a tenant_id column without registering it
// here would leave that table un-backfilled and unverified — exactly the shape
// of the defect migration 0033 shipped.
func TestTenantIDTablesMatchTheMigratedSchema(t *testing.T) {
	database := dbtest.Open(t)
	inSchema, err := tenantIDTablesInSchema(t.Context(), database)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	declared := make([]string, 0, len(tenantIDTables))
	for _, table := range tenantIDTables {
		declared = append(declared, table.name)
	}
	sort.Strings(declared)
	sort.Strings(inSchema)
	if strings.Join(declared, ",") != strings.Join(inSchema, ",") {
		t.Fatalf("tenantIDTables = %v\nschema has     = %v", declared, inSchema)
	}
}

func tenantIDTablesInSchema(ctx context.Context, database *sql.DB) ([]string, error) {
	var query string
	if dbtest.Backend() == "postgres" {
		query = `SELECT c.table_name FROM information_schema.columns c
			JOIN pg_catalog.pg_tables t ON t.schemaname=c.table_schema AND t.tablename=c.table_name
			WHERE c.table_schema=current_schema() AND c.column_name='tenant_id' AND c.table_name <> 'tenant'`
	} else {
		query = `SELECT m.name FROM sqlite_master m JOIN pragma_table_info(m.name) p
			WHERE m.type='table' AND p.name='tenant_id' AND m.name <> 'tenant'`
	}
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

// TestTenantIDBackfillRepairsRowsWrittenBeforeAnyIdentityExisted reproduces the
// production state migration 0033 actually leaves behind: rows that carry a slug
// and no identity, because the migration ran before `tenant` had a single row.
func TestTenantIDBackfillRepairsRowsWrittenBeforeAnyIdentityExisted(t *testing.T) {
	database := dbtest.Open(t)
	seedSlugOnlyRow(t, database, "announcements", "pre-identity", "a1")
	seedSlugOnlyRow(t, database, "units", "pre-identity", "u1")

	if missing := countMissingTenantIDs(t, database, "announcements"); missing != 1 {
		t.Fatalf("fixture did not reproduce the defect: %d rows without an identity", missing)
	}

	if _, err := EnsureTenantIdentities(t.Context(), database, []TenantIdentity{{Slug: "demo", Name: "Demo"}}); err != nil {
		t.Fatalf("boot: %v", err)
	}

	for _, table := range []string{"announcements", "units"} {
		if missing := countMissingTenantIDs(t, database, table); missing != 0 {
			t.Errorf("%s still has %d rows without an identity after boot", table, missing)
		}
	}
	// The slug was never configured, so the repair had to mint an identity for
	// it. Skipping it would have left the rows invisible to every tenant_id
	// filter with nothing reporting the loss.
	var id string
	if err := database.QueryRow(`SELECT tenant_id FROM tenant WHERE slug='pre-identity'`).Scan(&id); err != nil {
		t.Fatalf("an unconfigured slug present in the data must still get an identity: %v", err)
	}
	var linked string
	if err := database.QueryRow(`SELECT tenant_id FROM announcements WHERE id='a1'`).Scan(&linked); err != nil {
		t.Fatalf("read backfilled row: %v", err)
	}
	if linked != id {
		t.Fatalf("row linked to %q, want %q", linked, id)
	}
}

// TestTenantIDVerificationRefusesToBootWithRowsThatHaveNoIdentity is the guard
// requirement 3 asks for: reads are not switched onto a column that may be
// empty, and "may be empty" is decided by counting rather than by assumption.
func TestTenantIDVerificationRefusesToBootWithRowsThatHaveNoIdentity(t *testing.T) {
	database := dbtest.Open(t)
	// An empty slug cannot be resolved to any identity and cannot have one
	// minted for it, so this row is genuinely unrepairable — which is exactly
	// the case the boot must refuse rather than serve.
	seedSlugOnlyRow(t, database, "announcements", "", "orphan")

	err := BackfillTenantIDs(t.Context(), database)
	if !errors.Is(err, ErrTenantIDMissing) {
		t.Fatalf("backfill error = %v, want ErrTenantIDMissing", err)
	}
	if !strings.Contains(err.Error(), "announcements=1") {
		t.Fatalf("the error must name the table and the count, got: %v", err)
	}

	// EnsureTenantIdentities must fail for the same reason, because that is the
	// call the boot actually makes.
	if _, err := EnsureTenantIdentities(t.Context(), database, []TenantIdentity{{Slug: "demo"}}); !errors.Is(err, ErrTenantIDMissing) {
		t.Fatalf("boot error = %v, want ErrTenantIDMissing", err)
	}
}

// TestTenantIDBackfillIsIdempotent: a second boot must be a no-op, not a
// re-mint. Re-minting would orphan every row referencing the previous identity.
func TestTenantIDBackfillIsIdempotent(t *testing.T) {
	database := dbtest.Open(t)
	seedSlugOnlyRow(t, database, "announcements", "demo", "a1")
	configured := []TenantIdentity{{Slug: "demo", Name: "Demo"}}
	if _, err := EnsureTenantIdentities(t.Context(), database, configured); err != nil {
		t.Fatalf("first boot: %v", err)
	}
	var first string
	if err := database.QueryRow(`SELECT tenant_id FROM announcements WHERE id='a1'`).Scan(&first); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureTenantIdentities(t.Context(), database, configured); err != nil {
		t.Fatalf("second boot: %v", err)
	}
	var second string
	if err := database.QueryRow(`SELECT tenant_id FROM announcements WHERE id='a1'`).Scan(&second); err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("identity changed across boots: %q -> %q", first, second)
	}
}

// TestBoundRepositoriesSeparateTenantsWithDistinctIdentities is the isolation
// property the old fixtures could not test: both tenants used to share one
// hard-coded ULID, so filtering by tenant_id would have returned both rows and
// the test would still have passed.
func TestBoundRepositoriesSeparateTenantsWithDistinctIdentities(t *testing.T) {
	database := testDB(t)
	if testTenantID("demo") == testTenantID("other") {
		t.Fatal("the fixture tenants share an identity; isolation cannot be tested")
	}
	storage := NewSQLAnnouncementStore(database)
	demo, _ := BindAnnouncementRepository(storage, testTenantRef("demo"))
	other, _ := BindAnnouncementRepository(storage, testTenantRef("other"))
	if _, err := demo.Create(Announcement{Title: "Demo", Body: "x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := other.Create(Announcement{Title: "Other", Body: "x"}); err != nil {
		t.Fatal(err)
	}
	if got := demo.List(); len(got) != 1 || got[0].Title != "Demo" {
		t.Fatalf("demo sees %+v", got)
	}
	if got := other.List(); len(got) != 1 || got[0].Title != "Other" {
		t.Fatalf("other sees %+v", got)
	}
	// And the label is still written, so the rollback window is still open.
	var slug string
	if err := database.QueryRow(`SELECT tenant_slug FROM announcements WHERE tenant_id=$1`, testTenantID("demo")).Scan(&slug); err != nil {
		t.Fatalf("read tenant_slug: %v", err)
	}
	if slug != "demo" {
		t.Fatalf("tenant_slug = %q, want demo", slug)
	}
}

func seedSlugOnlyRow(t *testing.T, database *sql.DB, table, slug, id string) {
	t.Helper()
	if _, err := database.ExecContext(t.Context(),
		`INSERT INTO `+table+`(tenant_slug, id, data) VALUES($1, $2, '{}')`, slug, id); err != nil {
		t.Fatalf("seed %s: %v", table, err)
	}
}

func countMissingTenantIDs(t *testing.T, database *sql.DB, table string) int {
	t.Helper()
	var missing int
	if err := database.QueryRowContext(t.Context(),
		`SELECT count(*) FROM `+table+` WHERE tenant_id IS NULL`).Scan(&missing); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return missing
}
