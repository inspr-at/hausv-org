package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	scopedTenantA = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	scopedTenantB = "01BX5ZZKBKACTAV9WEVGEMMVRZ"
)

func scopedPostgres(t *testing.T) (*Scoped, *sql.DB) {
	t.Helper()
	baseDSN := strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN"))
	if baseDSN == "" {
		if os.Getenv("HAUSV_TEST_POSTGRES_REQUIRED") == "true" {
			t.Fatal("HAUSV_TEST_POSTGRES_DSN is required when HAUSV_TEST_POSTGRES_REQUIRED=true")
		}
		t.Skip("set HAUSV_TEST_POSTGRES_DSN to a disposable database owned by a NOSUPERUSER NOBYPASSRLS role")
	}
	cfg := Config{
		Backend:         BackendPostgres,
		DSN:             isolatedPostgresSchema(t, baseDSN),
		ConnectTimeout:  3 * time.Second,
		MaxOpenConns:    2,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: time.Minute,
	}
	database, err := OpenConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	scoped, err := NewScoped(cfg, database)
	if err != nil {
		t.Fatalf("new scoped: %v", err)
	}
	t.Cleanup(func() { _ = scoped.Close() })
	return scoped, database
}

func scopeOf(t *testing.T, handle Handle) string {
	t.Helper()
	var scope string
	if err := handle.QueryRow(`SELECT coalesce(current_setting('hausv.tenant_id', true), '')`).Scan(&scope); err != nil {
		t.Fatalf("read tenant scope: %v", err)
	}
	return scope
}

// TRAP (a). stdlib.OpenDB takes the ConnConfig BY VALUE, which reads like a
// copy and is not one where it matters: RuntimeParams is a map, and a struct
// copy copies the map HEADER. Build lane A from the template, then set the
// template's tenant for lane B, and lane A — which has not connected yet,
// because database/sql is lazy — is now pointed at tenant B too. Nothing errors,
// nothing logs, and the first query lane A ever runs returns another tenant's
// rows.
//
// The lanes are therefore created before either is used, which is exactly the
// order a request-serving process produces.
func TestEachLaneKeepsItsOwnTenantWhenAnotherLaneIsBuilt(t *testing.T) {
	scoped, _ := scopedPostgres(t)

	laneA := scoped.For(scopedTenantA)
	laneB := scoped.For(scopedTenantB)

	if got := scopeOf(t, laneA); got != scopedTenantA {
		t.Fatalf("lane A reports tenant %q, want %q — building lane B re-pointed it", got, scopedTenantA)
	}
	if got := scopeOf(t, laneB); got != scopedTenantB {
		t.Fatalf("lane B reports tenant %q, want %q", got, scopedTenantB)
	}
}

// TRAP (b). A lane rebuilt from the DSN alone loses every tuning decision that
// was applied to the parsed config AFTER parsing — statement_timeout above all,
// which openPostgres injects as a runtime param. The lane then runs without the
// 30-second cap the app pool has, so one pathological query on a tenant lane can
// hold a backend open indefinitely while the same query on the process pool is
// cut off. It is invisible until it matters.
func TestLaneCarriesTheSameStatementTimeoutAsTheProcessPool(t *testing.T) {
	scoped, database := scopedPostgres(t)

	var poolTimeout string
	if err := database.QueryRow(`SHOW statement_timeout`).Scan(&poolTimeout); err != nil {
		t.Fatalf("read pool statement_timeout: %v", err)
	}
	if poolTimeout == "0" {
		t.Fatal("the process pool itself has no statement timeout; this test cannot mean anything")
	}

	var laneTimeout string
	if err := scoped.For(scopedTenantA).QueryRow(`SHOW statement_timeout`).Scan(&laneTimeout); err != nil {
		t.Fatalf("read lane statement_timeout: %v", err)
	}
	if laneTimeout != poolTimeout {
		t.Fatalf("lane statement_timeout = %q, process pool = %q — the lane lost the pool's tuning", laneTimeout, poolTimeout)
	}
}

// The startup-packet placement is what makes a session-reset hook unnecessary.
// If any of these reset the scope, a pooled connection could be handed to the
// next borrower unscoped — which under the maintenance-view policy is the whole
// database — so the claim is measured rather than assumed.
func TestLaneScopeSurvivesEveryResetAConnectionCanBeGiven(t *testing.T) {
	scoped, _ := scopedPostgres(t)
	lane := scoped.For(scopedTenantA)

	for index, reset := range []string{`RESET ALL`, `RESET "hausv.tenant_id"`, `SET "hausv.tenant_id" = DEFAULT`, `DISCARD ALL`} {
		conn, err := lane.(*sql.DB).Conn(t.Context())
		if err != nil {
			t.Fatalf("check out lane connection: %v", err)
		}
		if _, err := conn.ExecContext(t.Context(), reset); err != nil {
			conn.Close()
			t.Fatalf("%s: %v", reset, err)
		}
		var scope string
		// The read text differs per iteration on purpose. DISCARD ALL also drops
		// the connection's PREPARED STATEMENTS, which pgx still has in its
		// statement cache, so re-running an identical query on that same
		// connection fails with SQLSTATE 26000. That is a pgx caching detail and
		// not the scope claim under test — but it is the reason application code
		// must never issue DISCARD ALL on a pooled connection.
		read := fmt.Sprintf(`SELECT /* reset %d */ coalesce(current_setting('hausv.tenant_id', true), '')`, index)
		if err := conn.QueryRowContext(t.Context(), read).Scan(&scope); err != nil {
			conn.Close()
			t.Fatalf("read scope after %s: %v", reset, err)
		}
		conn.Close()
		if scope != scopedTenantA {
			t.Fatalf("scope after %s = %q, want %q — the lane needs a session-reset hook after all", reset, scope, scopedTenantA)
		}
	}
}

// A transaction started on a lane inherits the lane's scope: the seam has to
// work for the stores that use Begin, not only the ones that use Exec.
func TestTransactionOnALaneInheritsTheLaneScope(t *testing.T) {
	scoped, _ := scopedPostgres(t)
	tx, err := scoped.For(scopedTenantB).Begin()
	if err != nil {
		t.Fatalf("begin on lane: %v", err)
	}
	defer tx.Rollback()
	var scope string
	if err := tx.QueryRow(`SELECT coalesce(current_setting('hausv.tenant_id', true), '')`).Scan(&scope); err != nil {
		t.Fatalf("read scope in transaction: %v", err)
	}
	if scope != scopedTenantB {
		t.Fatalf("transaction scope = %q, want %q", scope, scopedTenantB)
	}
}

// A tenant id that is not a ULID cannot be allowed to become "no scope": under
// the maintenance-view policy that is the entire database. It becomes a value no
// row can carry instead.
func TestUnusableTenantIDFailsClosedInsteadOfUnscoped(t *testing.T) {
	scoped, _ := scopedPostgres(t)
	for _, bad := range []string{"", "   ", "demo", "01ARZ3NDEKTSV4RRFFQ69G5FA", strings.Repeat("Z", 26)} {
		if got := scopeOf(t, scoped.For(bad)); got != tenantSentinel {
			t.Fatalf("tenant id %q produced scope %q, want the fail-closed sentinel %q", bad, got, tenantSentinel)
		}
	}
}

// The maintenance lane declares itself in the startup packet, and the process
// pool declares nothing — so once the policy flips, the process pool sees
// nothing and only the lane that asked for it can cross tenants.
func TestUnscopedLaneDeclaresItselfAndTheProcessPoolDoesNot(t *testing.T) {
	scoped, database := scopedPostgres(t)

	var declared string
	if err := scoped.Unscoped("schema migration").
		QueryRow(`SELECT coalesce(current_setting('hausv.cross_tenant', true), '')`).Scan(&declared); err != nil {
		t.Fatalf("read maintenance declaration: %v", err)
	}
	if declared != crossTenantOn {
		t.Fatalf("maintenance lane declaration = %q, want %q", declared, crossTenantOn)
	}
	if got := scopeOf(t, scoped.Unscoped("schema migration")); got != "" {
		t.Fatalf("maintenance lane must not also carry a tenant scope, got %q", got)
	}

	var poolTenant, poolCross string
	if err := database.QueryRow(`
		SELECT coalesce(current_setting('hausv.tenant_id', true), ''),
		       coalesce(current_setting('hausv.cross_tenant', true), '')`).Scan(&poolTenant, &poolCross); err != nil {
		t.Fatalf("read process pool scope: %v", err)
	}
	if poolTenant != "" || poolCross != "" {
		t.Fatalf("process pool declares tenant=%q cross_tenant=%q, want neither", poolTenant, poolCross)
	}
}

// Asking for the same tenant twice must reuse the pool, not open a second one:
// a lane per call is how a process runs out of backends.
func TestTheSameTenantGetsTheSameLane(t *testing.T) {
	scoped, _ := scopedPostgres(t)
	if scoped.For(scopedTenantA) != scoped.For(scopedTenantA) {
		t.Fatal("two calls for one tenant produced two pools")
	}
	if scoped.Unscoped("first") != scoped.Unscoped("second") {
		t.Fatal("two maintenance calls produced two pools")
	}
}

// On SQLite there are no lanes and no RLS. Every accessor must hand back the
// IDENTICAL pool the stores already hold — pointer identity, not "a pool that
// also works", because a second *sql.DB over one SQLite file is a second write
// lock and a different transaction view.
func TestSQLiteHandsBackTheOneProcessPool(t *testing.T) {
	database, err := Open(t.TempDir() + "/scoped.db")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	scoped, err := NewScoped(Config{DSN: "unused"}, database)
	if err != nil {
		t.Fatalf("new scoped: %v", err)
	}
	for name, handle := range map[string]Handle{
		"For(A)":    scoped.For(scopedTenantA),
		"For(B)":    scoped.For(scopedTenantB),
		"For(junk)": scoped.For("not-a-ulid"),
		"Unscoped":  scoped.Unscoped("test"),
	} {
		if handle != Handle(database) {
			t.Fatalf("%s returned a different pool than the process pool", name)
		}
	}
}
